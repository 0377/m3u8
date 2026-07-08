package task

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/0377/m3u8/api"
	"github.com/0377/m3u8/parse"
	"github.com/0377/m3u8/tool"
	"github.com/google/uuid"
)

var (
	ErrTooManyTasks = errors.New("too many running tasks")
	ErrTaskNotFound = errors.New("task not found")
)

type Config struct {
	DataDir  string
	MaxTasks int
	TaskTTL  time.Duration
}

type Manager struct {
	store    *Store
	cfg      Config
	mu       sync.Mutex
	running  int
	workCh   chan *api.TaskRecord
	onCancel map[string]context.CancelFunc
}

func NewManager(cfg Config) *Manager {
	maxTasks := cfg.MaxTasks
	if maxTasks <= 0 {
		maxTasks = 1
	}
	return &Manager{
		store:    NewStore(filepath.Join(cfg.DataDir, "tasks")),
		cfg:      cfg,
		workCh:   make(chan *api.TaskRecord, maxTasks),
		onCancel: make(map[string]context.CancelFunc),
	}
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
	if m.countRunningLocked() >= m.cfg.MaxTasks {
		m.mu.Unlock()
		return nil, ErrTooManyTasks
	}
	m.mu.Unlock()

	result, err := parse.FromURL(req.URL)
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
		Status:       api.TaskStatusPending,
		SegmentTotal: len(result.M3u8.Segments),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := m.store.Save(rec); err != nil {
		return nil, err
	}

	m.enqueue(rec)
	return rec, nil
}

func (m *Manager) Get(taskID string) (*api.TaskRecord, error) {
	rec, err := m.store.Load(taskID)
	if err != nil {
		if err.Error() == "task not found" {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return rec, nil
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
		return nil, nil
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
	return m.store.Save(rec)
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

func (m *Manager) Recover() error {
	all, err := m.store.ListAll()
	if err != nil {
		return err
	}
	for _, rec := range all {
		switch rec.Status {
		case api.TaskStatusRunning:
			rec.Status = api.TaskStatusPending
			rec.UpdatedAt = time.Now().UTC()
			if err := m.store.Save(rec); err != nil {
				return err
			}
			m.enqueue(rec)
		case api.TaskStatusPending:
			m.enqueue(rec)
		}
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

func (m *Manager) countRunningLocked() int {
	all, err := m.store.ListAll()
	if err != nil {
		return m.running
	}
	count := 0
	for _, rec := range all {
		if rec.Status == api.TaskStatusRunning {
			count++
		}
	}
	return count
}

func (m *Manager) enqueue(rec *api.TaskRecord) {
	select {
	case m.workCh <- rec:
	default:
	}
}
