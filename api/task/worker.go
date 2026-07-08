package task

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/0377/m3u8/api"
	"github.com/0377/m3u8/dl"
	"github.com/0377/m3u8/tool"
)

func (m *Manager) StartWorkers(n int) {
	for i := 0; i < n; i++ {
		go m.workerLoop()
	}
}

func (m *Manager) workerLoop() {
	for rec := range m.workCh {
		if fresh, err := m.store.Load(rec.TaskID); err == nil {
			if fresh.Cancelled {
				if fresh.Status == api.TaskStatusPending {
					fresh.Status = api.TaskStatusCancelled
					fresh.UpdatedAt = time.Now().UTC()
					_ = m.store.Save(fresh)
				}
				continue
			}
			rec = fresh
		}
		m.runTask(rec)
	}
}

func (m *Manager) runTask(rec *api.TaskRecord) {
	if rec.Cancelled {
		rec.Status = api.TaskStatusCancelled
		rec.UpdatedAt = time.Now().UTC()
		_ = m.store.Save(rec)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.onCancel[rec.TaskID] = cancel
	m.running++
	m.mu.Unlock()

	defer func() {
		cancel()
		m.mu.Lock()
		m.running--
		delete(m.onCancel, rec.TaskID)
		m.mu.Unlock()
	}()

	rec.Status = api.TaskStatusRunning
	rec.Message = "downloading"
	rec.UpdatedAt = time.Now().UTC()
	_ = m.store.Save(rec)

	taskDir := m.store.TaskDir(rec.TaskID)
	httpCfg, err := taskHTTPConfig(rec.Proxy)
	if err != nil {
		m.failTask(rec, err.Error())
		return
	}
	var downloader *dl.Downloader
	if result := m.takeParsedResult(rec.TaskID); result != nil {
		downloader, err = dl.NewTaskFromResult(taskDir, rec.URL, rec.Filename, result, httpCfg, m.cryptSvc)
	} else {
		downloader, err = dl.NewTask(taskDir, rec.URL, rec.Filename, httpCfg, m.cryptSvc)
	}
	if err != nil {
		m.failTask(rec, err.Error())
		return
	}

	var progressMu sync.Mutex
	downloader.SetProgressReporter(func(done, total int, message string) {
		if ctx.Err() != nil {
			return
		}
		progressMu.Lock()
		rec.SegmentDone = done
		rec.SegmentTotal = total
		if total > 0 {
			rec.Progress = float64(done) / float64(total) * 100
		}
		rec.Message = message
		rec.UpdatedAt = time.Now().UTC()
		snapshot := *rec
		progressMu.Unlock()
		_ = m.store.Save(&snapshot)
	})

	const maxRetry = 10
	downloader.SetCancelContext(ctx)
	if err := downloader.Start(rec.Concurrency, rec.ToMP4, maxRetry); err != nil {
		if rec.Cancelled || errors.Is(err, context.Canceled) {
			m.cancelTask(rec)
		} else {
			m.failTask(rec, err.Error())
		}
		return
	}

	now := time.Now().UTC()
	expires := now.Add(m.cfg.TaskTTL)
	rec.Status = api.TaskStatusCompleted
	rec.Progress = 100
	rec.Message = "completed"
	rec.CompletedAt = &now
	rec.ExpiresAt = &expires
	rec.UpdatedAt = now

	baseName, _ := tool.ResolveOutputBaseName(rec.Filename)
	var outFile string
	format := "ts"
	if rec.ToMP4 {
		outFile = filepath.Join(taskDir, baseName+".mp4")
		format = "mp4"
	} else {
		outFile = filepath.Join(taskDir, baseName+".ts")
	}
	fi, err := os.Stat(outFile)
	if err != nil {
		m.failTask(rec, "output file not found: "+err.Error())
		return
	}
	rec.Output = &api.OutputInfo{
		Filename:  filepath.Base(outFile),
		SizeBytes: fi.Size(),
		Format:    format,
	}
	_ = m.store.Save(rec)
}

func (m *Manager) failTask(rec *api.TaskRecord, msg string) {
	rec.Status = api.TaskStatusFailed
	rec.Error = msg
	rec.UpdatedAt = time.Now().UTC()
	_ = m.store.Save(rec)
}

func (m *Manager) cancelTask(rec *api.TaskRecord) {
	rec.Status = api.TaskStatusCancelled
	rec.UpdatedAt = time.Now().UTC()
	_ = m.store.Save(rec)
}
