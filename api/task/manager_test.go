package task

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/0377/m3u8/api"
	"github.com/google/uuid"
)

func newTestM3U8Server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	serveM3U8 := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:10\n#EXTINF:10.0,\nseg0.ts\n#EXT-X-ENDLIST\n")
	}
	mux.HandleFunc("/index.m3u8", serveM3U8)
	mux.HandleFunc("/a.m3u8", serveM3U8)
	mux.HandleFunc("/b.m3u8", serveM3U8)
	return httptest.NewServer(mux)
}

func newTestManager(t *testing.T, cfg Config) *Manager {
	t.Helper()
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestManagerCreateAndGet(t *testing.T) {
	srv := newTestM3U8Server(t)
	defer srv.Close()

	dir := t.TempDir()
	m := newTestManager(t, Config{
		DataDir:  dir,
		MaxTasks: 3,
		TaskTTL:  24 * time.Hour,
	})
	rec, err := m.Create(&api.CreateTaskRequest{
		URL:      srv.URL + "/index.m3u8",
		Filename: "main",
	}, 10)
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.Get(rec.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != api.TaskStatusPending {
		t.Fatalf("want pending, got %s", got.Status)
	}
}

func TestManagerRecoverNoDeadlock(t *testing.T) {
	dir := t.TempDir()
	m := newTestManager(t, Config{DataDir: dir, MaxTasks: 2, TaskTTL: time.Hour})

	now := time.Now().UTC()
	for i := 0; i < 4; i++ {
		rec := &api.TaskRecord{
			TaskID:    uuid.New().String(),
			Status:    api.TaskStatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := m.store.Save(rec); err != nil {
			t.Fatal(err)
		}
	}

	m.StartWorkers(2)

	done := make(chan struct{})
	go func() {
		if err := m.Recover(); err != nil {
			t.Errorf("Recover: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Recover deadlocked")
	}
	time.Sleep(300 * time.Millisecond)
}

func TestManagerListEmptyOffset(t *testing.T) {
	dir := t.TempDir()
	m := newTestManager(t, Config{DataDir: dir, MaxTasks: 1, TaskTTL: time.Hour})
	tasks, err := m.List("", 20, 100)
	if err != nil {
		t.Fatal(err)
	}
	if tasks == nil {
		t.Fatal("want non-nil empty slice")
	}
	if len(tasks) != 0 {
		t.Fatalf("want empty slice, got %d", len(tasks))
	}
}

func TestManagerMaxTasksLimit(t *testing.T) {
	srv := newTestM3U8Server(t)
	defer srv.Close()

	dir := t.TempDir()
	m := newTestManager(t, Config{DataDir: dir, MaxTasks: 1, TaskTTL: time.Hour})
	_, err := m.Create(&api.CreateTaskRequest{URL: srv.URL + "/a.m3u8"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	all, _ := m.store.ListAll()
	all[0].Status = api.TaskStatusRunning
	_ = m.store.Save(all[0])
	_, err = m.Create(&api.CreateTaskRequest{URL: srv.URL + "/b.m3u8"}, 10)
	if err != api.ErrTooManyTasks {
		t.Fatalf("want ErrTooManyTasks, got %v", err)
	}
}

func TestManagerMaxTasksPendingBlocks(t *testing.T) {
	srv := newTestM3U8Server(t)
	defer srv.Close()

	dir := t.TempDir()
	m := newTestManager(t, Config{DataDir: dir, MaxTasks: 1, TaskTTL: time.Hour})
	_, err := m.Create(&api.CreateTaskRequest{URL: srv.URL + "/a.m3u8"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Create(&api.CreateTaskRequest{URL: srv.URL + "/b.m3u8"}, 10)
	if err != api.ErrTooManyTasks {
		t.Fatalf("want ErrTooManyTasks for pending slot, got %v", err)
	}
}

func TestManagerMaxTasksTOCTOU(t *testing.T) {
	srv := newTestM3U8Server(t)
	defer srv.Close()

	dir := t.TempDir()
	m := newTestManager(t, Config{DataDir: dir, MaxTasks: 1, TaskTTL: time.Hour})

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := m.Create(&api.CreateTaskRequest{URL: srv.URL + "/a.m3u8"}, 10)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	var success, rejected int
	for err := range results {
		if err == nil {
			success++
		} else if err == api.ErrTooManyTasks {
			rejected++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if success != 1 || rejected != 1 {
		t.Fatalf("want 1 success and 1 rejection, got success=%d rejected=%d", success, rejected)
	}
}

func TestManagerCancelPendingClearsParsedCache(t *testing.T) {
	srv := newTestM3U8Server(t)
	defer srv.Close()

	dir := t.TempDir()
	m := newTestManager(t, Config{DataDir: dir, MaxTasks: 2, TaskTTL: time.Hour})

	rec, err := m.Create(&api.CreateTaskRequest{URL: srv.URL + "/index.m3u8"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.parsed.Load(rec.TaskID); !ok {
		t.Fatal("parsed cache should be populated after Create")
	}

	if err := m.Cancel(rec.TaskID); err != nil {
		t.Fatal(err)
	}

	m.StartWorkers(1)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := m.parsed.Load(rec.TaskID); !ok {
			got, err := m.Get(rec.TaskID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != api.TaskStatusCancelled {
				t.Fatalf("want cancelled, got %s", got.Status)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("parsed cache not cleared after cancel")
}

func TestManagerCancelAfterDownloadMarksCancelled(t *testing.T) {
	tsData := make([]byte, 188)
	tsData[0] = 0x47

	mux := http.NewServeMux()
	mux.HandleFunc("/index.m3u8", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:10\n#EXTINF:10.0,\n/seg0.ts\n#EXT-X-ENDLIST\n")
	})
	mux.HandleFunc("/seg0.ts", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write(tsData)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	m := newTestManager(t, Config{DataDir: dir, MaxTasks: 1, TaskTTL: time.Hour})
	m.StartWorkers(1)

	toMP4 := false
	rec, err := m.Create(&api.CreateTaskRequest{
		URL:         srv.URL + "/index.m3u8",
		Filename:    "out",
		Concurrency: 1,
		ToMP4:       &toMP4,
	}, 10)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond)
	if err := m.Cancel(rec.TaskID); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		got, err := m.Get(rec.TaskID)
		if err != nil {
			t.Fatal(err)
		}
		switch got.Status {
		case api.TaskStatusCancelled:
			return
		case api.TaskStatusCompleted:
			t.Fatal("cancelled task was marked completed")
		case api.TaskStatusFailed:
			t.Fatalf("task failed: %s", got.Error)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("task did not reach cancelled status")
}

func TestManagerRecoverRespectsMaxTasks(t *testing.T) {
	dir := t.TempDir()
	m := newTestManager(t, Config{DataDir: dir, MaxTasks: 2, TaskTTL: time.Hour})

	now := time.Now().UTC()
	for i := 0; i < 4; i++ {
		rec := &api.TaskRecord{
			TaskID:    uuid.New().String(),
			Status:    api.TaskStatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := m.store.Save(rec); err != nil {
			t.Fatal(err)
		}
	}

	if err := m.Recover(); err != nil {
		t.Fatal(err)
	}

	dispatched := 0
	m.dispatched.Range(func(_, _ any) bool {
		dispatched++
		return true
	})
	if dispatched > 2 {
		t.Fatalf("want at most 2 dispatched on recover, got %d", dispatched)
	}
}

func TestManagerRecoverMaxTasksOneDrainsBacklog(t *testing.T) {
	tsData := make([]byte, 188)
	tsData[0] = 0x47

	mux := http.NewServeMux()
	serveM3U8 := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:10\n#EXTINF:10.0,\nseg0.ts\n#EXT-X-ENDLIST\n")
	}
	mux.HandleFunc("/index.m3u8", serveM3U8)
	mux.HandleFunc("/seg0.ts", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(tsData)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	m := newTestManager(t, Config{DataDir: dir, MaxTasks: 1, TaskTTL: time.Hour})
	m.StartWorkers(1)

	now := time.Now().UTC()
	url := srv.URL + "/index.m3u8"
	toMP4 := false
	for i := 0; i < 2; i++ {
		rec := &api.TaskRecord{
			TaskID:      uuid.New().String(),
			URL:         url,
			Filename:    "out",
			Concurrency: 1,
			ToMP4:       toMP4,
			Status:      api.TaskStatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := m.store.Save(rec); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Recover(); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		all, err := m.store.ListAll()
		if err != nil {
			t.Fatal(err)
		}
		completed := 0
		for _, rec := range all {
			if rec.Status == api.TaskStatusCompleted {
				completed++
			}
		}
		if completed == 2 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("backlog not fully drained with maxTasks=1")
}

func TestManagerCancelUndispatchedReleasesSlot(t *testing.T) {
	dir := t.TempDir()
	m := newTestManager(t, Config{DataDir: dir, MaxTasks: 1, TaskTTL: time.Hour})

	now := time.Now().UTC()
	ids := make([]string, 2)
	for i := 0; i < 2; i++ {
		ids[i] = uuid.New().String()
		rec := &api.TaskRecord{
			TaskID:    ids[i],
			Status:    api.TaskStatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := m.store.Save(rec); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Recover(); err != nil {
		t.Fatal(err)
	}

	m.mu.Lock()
	before := m.activeSlots
	m.mu.Unlock()

	var undispatchedID string
	for _, id := range ids {
		if _, ok := m.dispatched.Load(id); !ok {
			undispatchedID = id
			break
		}
	}
	if undispatchedID == "" {
		t.Fatal("expected one undispatched pending task")
	}

	if err := m.Cancel(undispatchedID); err != nil {
		t.Fatal(err)
	}

	m.mu.Lock()
	after := m.activeSlots
	m.mu.Unlock()
	if after != before-1 {
		t.Fatalf("activeSlots %d -> %d, want %d", before, after, before-1)
	}
}

func TestManagerShutdown(t *testing.T) {
	srv := newTestM3U8Server(t)
	defer srv.Close()

	dir := t.TempDir()
	m := newTestManager(t, Config{DataDir: dir, MaxTasks: 2, TaskTTL: time.Hour})
	m.StartWorkers(2)

	_, err := m.Create(&api.CreateTaskRequest{URL: srv.URL + "/index.m3u8"}, 10)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	if err := m.Shutdown(ctx2); err != nil {
		t.Fatalf("second Shutdown: %v", err)
	}
}
