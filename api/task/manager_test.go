package task

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0377/m3u8/api"
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

func TestManagerCreateAndGet(t *testing.T) {
	srv := newTestM3U8Server(t)
	defer srv.Close()

	dir := t.TempDir()
	m := NewManager(Config{
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

func TestManagerMaxTasksLimit(t *testing.T) {
	srv := newTestM3U8Server(t)
	defer srv.Close()

	dir := t.TempDir()
	m := NewManager(Config{DataDir: dir, MaxTasks: 1, TaskTTL: time.Hour})
	_, err := m.Create(&api.CreateTaskRequest{URL: srv.URL + "/a.m3u8"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	all, _ := m.store.ListAll()
	all[0].Status = api.TaskStatusRunning
	_ = m.store.Save(all[0])
	_, err = m.Create(&api.CreateTaskRequest{URL: srv.URL + "/b.m3u8"}, 10)
	if err != ErrTooManyTasks {
		t.Fatalf("want ErrTooManyTasks, got %v", err)
	}
}
