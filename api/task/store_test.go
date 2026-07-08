package task

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0377/m3u8/api"
)

func TestStoreSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "tasks"))
	rec := &api.TaskRecord{
		TaskID:    "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		URL:       "https://example.com/index.m3u8",
		Filename:  "main",
		Status:    api.TaskStatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.Save(rec); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load(rec.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.URL != rec.URL {
		t.Fatalf("want URL %q, got %q", rec.URL, loaded.URL)
	}
}

func TestStoreListAll(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "tasks"))
	for _, id := range []string{
		"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"b2c3d4e5-f6a7-8901-bcde-f12345678901",
	} {
		if err := s.Save(&api.TaskRecord{TaskID: id, Status: api.TaskStatusPending, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 tasks, got %d", len(all))
	}
}

func TestStoreTaskDir(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "tasks"))
	taskID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	taskDir := s.TaskDir(taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(taskDir); err != nil {
		t.Fatal(err)
	}
}
