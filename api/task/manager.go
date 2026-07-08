package task

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/0377/m3u8/api"
	"github.com/0377/m3u8/crypt"
	_ "github.com/0377/m3u8/crypt/provider"
	"github.com/0377/m3u8/parse"
	"github.com/0377/m3u8/tool"
	"github.com/google/uuid"
)

type Config struct {
	DataDir  string
	MaxTasks int
	TaskTTL  time.Duration
}

type Manager struct {
	store    *Store
	cfg      Config
	cryptSvc *crypt.Service
	mu       sync.Mutex
	running  int
	activeSlots int
	shuttingDown bool
	shutdownOnce sync.Once
	workerWg sync.WaitGroup
	workCh   chan *api.TaskRecord
	onCancel map[string]context.CancelFunc
	parsed   sync.Map // taskID -> *parse.Result, populated at Create
	dispatched sync.Map // taskID -> struct{}, tasks sent to workCh but not yet running
}

func NewManager(cfg Config) (*Manager, error) {
	maxTasks := cfg.MaxTasks
	if maxTasks <= 0 {
		maxTasks = 1
	}
	_, cryptSvc, err := crypt.BuildService("", crypt.ServiceOptions{})
	if err != nil {
		return nil, err
	}
	return &Manager{
		store:    NewStore(filepath.Join(cfg.DataDir, "tasks")),
		cfg:      cfg,
		cryptSvc: cryptSvc,
		workCh:   make(chan *api.TaskRecord, maxTasks),
		onCancel: make(map[string]context.CancelFunc),
	}, nil
}

func (m *Manager) Create(req *api.CreateTaskRequest, maxRetry int) (*api.TaskRecord, error) {
	_ = maxRetry

	if req == nil || strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("url is required")
	}

	filename, err := tool.ResolveOutputBaseName(req.Filename)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	if m.shuttingDown {
		m.mu.Unlock()
		return nil, fmt.Errorf("manager is shutting down")
	}
	if m.activeSlots >= m.cfg.MaxTasks {
		m.mu.Unlock()
		return nil, api.ErrTooManyTasks
	}
	m.activeSlots++
	m.mu.Unlock()

	slotReserved := true
	defer func() {
		if slotReserved {
			m.releaseSlot()
		}
	}()

	httpCfg, err := taskHTTPConfig(req.Proxy)
	if err != nil {
		return nil, err
	}

	result, err := parse.FromURL(req.URL, httpCfg, m.cryptSvc)
	if err != nil {
		return nil, err
	}

	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 25
	}

	toMP4 := true
	if req.ToMP4 != nil {
		toMP4 = *req.ToMP4
	}

	now := time.Now().UTC()
	rec := &api.TaskRecord{
		TaskID:       uuid.New().String(),
		URL:          req.URL,
		Filename:     filename,
		Concurrency:  concurrency,
		ToMP4:        toMP4,
		Proxy:        req.Proxy,
		Status:       api.TaskStatusPending,
		SegmentTotal: len(result.M3u8.Segments),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := m.store.Save(rec); err != nil {
		return nil, err
	}

	slotReserved = false
	m.parsed.Store(rec.TaskID, result)
	if !m.enqueue(rec) {
		m.takeParsedResult(rec.TaskID)
		m.mu.Lock()
		if m.activeSlots > 0 {
			m.activeSlots--
		}
		m.mu.Unlock()
		rec.Status = api.TaskStatusCancelled
		rec.UpdatedAt = time.Now().UTC()
		_ = m.store.Save(rec)
		return nil, fmt.Errorf("manager is shutting down")
	}
	return rec, nil
}

func (m *Manager) Get(taskID string) (*api.TaskRecord, error) {
	return m.store.Load(taskID)
}

func (m *Manager) List(status string, limit, offset int) ([]*api.TaskRecord, error) {
	all, err := m.store.ListAll()
	if err != nil {
		return nil, err
	}

	var filtered []*api.TaskRecord
	for _, rec := range all {
		if status == "" || string(rec.Status) == status {
			filtered = append(filtered, rec)
		}
	}

	if offset >= len(filtered) {
		return []*api.TaskRecord{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], nil
}

func (m *Manager) Cancel(taskID string) error {
	rec, err := m.Get(taskID)
	if err != nil {
		return err
	}

	switch rec.Status {
	case api.TaskStatusCompleted, api.TaskStatusFailed, api.TaskStatusExpired, api.TaskStatusCancelled:
		return fmt.Errorf("task cannot be cancelled in status %s", rec.Status)
	}

	m.mu.Lock()
	if cancel, ok := m.onCancel[taskID]; ok {
		cancel()
	}
	m.mu.Unlock()

	rec.Cancelled = true
	rec.UpdatedAt = time.Now().UTC()
	if rec.Status == api.TaskStatusPending {
		rec.Status = api.TaskStatusCancelled
	}
	if err := m.store.Save(rec); err != nil {
		return err
	}

	m.mu.Lock()
	_, dispatched := m.dispatched.Load(taskID)
	_, running := m.onCancel[taskID]
	m.mu.Unlock()
	if !dispatched && !running {
		m.takeParsedResult(taskID)
		m.releaseSlot()
	}
	return nil
}

func (m *Manager) ToResponse(rec *api.TaskRecord) api.TaskResponse {
	resp := api.TaskResponse{
		TaskID:       rec.TaskID,
		Status:       rec.Status,
		Progress:     rec.Progress,
		Message:      rec.Message,
		SegmentTotal: rec.SegmentTotal,
		SegmentDone:  rec.SegmentDone,
		Error:        rec.Error,
		Output:       rec.Output,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
		CompletedAt:  rec.CompletedAt,
		ExpiresAt:    rec.ExpiresAt,
	}
	if rec.Status == api.TaskStatusCompleted {
		resp.DownloadURL = fmt.Sprintf("/api/v1/tasks/%s/download", rec.TaskID)
	}
	return resp
}

func (m *Manager) TaskDir(taskID string) string {
	return m.store.TaskDir(taskID)
}

func (m *Manager) CryptService() *crypt.Service {
	return m.cryptSvc
}

// Close releases shared resources (external decryptor processes, etc.).
func (m *Manager) Close() error {
	if m.cryptSvc != nil {
		return m.cryptSvc.Close()
	}
	return nil
}

// Shutdown cancels running tasks, waits for workers to drain, then closes resources.
func (m *Manager) Shutdown(ctx context.Context) error {
	var shutdownErr error
	m.shutdownOnce.Do(func() {
		m.mu.Lock()
		m.shuttingDown = true
		for _, cancel := range m.onCancel {
			cancel()
		}
		m.mu.Unlock()

		close(m.workCh)

		done := make(chan struct{})
		go func() {
			m.workerWg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			shutdownErr = ctx.Err()
		}

		if closeErr := m.Close(); closeErr != nil && shutdownErr == nil {
			shutdownErr = closeErr
		}
	})
	return shutdownErr
}

func (m *Manager) takeParsedResult(taskID string) *parse.Result {
	v, ok := m.parsed.LoadAndDelete(taskID)
	if !ok {
		return nil
	}
	return v.(*parse.Result)
}

func (m *Manager) Recover() error {
	all, err := m.store.ListAll()
	if err != nil {
		return err
	}

	var pending []*api.TaskRecord
	for _, rec := range all {
		switch rec.Status {
		case api.TaskStatusRunning:
			rec.Status = api.TaskStatusPending
			rec.UpdatedAt = time.Now().UTC()
			if err := m.store.Save(rec); err != nil {
				return err
			}
			pending = append(pending, rec)
		case api.TaskStatusPending:
			pending = append(pending, rec)
		}
	}

	m.mu.Lock()
	m.activeSlots = len(pending)
	avail := m.cfg.MaxTasks - m.pipelineLocked()
	if avail < 0 {
		avail = 0
	}
	toEnqueue := avail
	if toEnqueue > len(pending) {
		toEnqueue = len(pending)
	}
	m.mu.Unlock()

	for i := 0; i < toEnqueue; i++ {
		m.enqueue(pending[i])
	}
	return nil
}

func (m *Manager) CleanupExpired() error {
	all, err := m.store.ListAll()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, rec := range all {
		if rec.ExpiresAt == nil || !now.After(*rec.ExpiresAt) {
			continue
		}
		rec.Status = api.TaskStatusExpired
		rec.UpdatedAt = now
		_ = m.store.Save(rec)
		if err := m.store.Delete(rec.TaskID); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			_ = m.CleanupExpired()
		}
	}()
}

func (m *Manager) pipelineLocked() int {
	return m.running + m.countDispatchedLocked()
}

func (m *Manager) countDispatchedLocked() int {
	count := 0
	m.dispatched.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

func (m *Manager) releaseSlot() {
	m.mu.Lock()
	if m.activeSlots > 0 {
		m.activeSlots--
	}
	m.mu.Unlock()
	m.dispatchPending()
}

func (m *Manager) dispatchPending() {
	m.mu.Lock()
	if m.shuttingDown {
		m.mu.Unlock()
		return
	}
	slots := m.cfg.MaxTasks - m.pipelineLocked()
	m.mu.Unlock()
	if slots <= 0 {
		return
	}

	all, err := m.store.ListAll()
	if err != nil {
		return
	}

	for _, rec := range all {
		if slots <= 0 {
			return
		}
		if rec.Status != api.TaskStatusPending || rec.Cancelled {
			continue
		}
		if _, ok := m.dispatched.Load(rec.TaskID); ok {
			continue
		}
		m.enqueue(rec)
		slots--
	}
}

func taskHTTPConfig(proxy string) (*tool.HTTPConfig, error) {
	return tool.HTTPConfigFrom(nil, "", proxy, false)
}

func (m *Manager) enqueue(rec *api.TaskRecord) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.shuttingDown {
		return false
	}
	m.dispatched.Store(rec.TaskID, struct{}{})
	m.workCh <- rec
	return true
}
