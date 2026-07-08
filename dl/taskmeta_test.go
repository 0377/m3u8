package dl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTaskMetaSaveLoad(t *testing.T) {
	dir := t.TempDir()
	meta := &TaskMeta{
		Version:      taskMetaVersion,
		URL:          "https://example.com/index.m3u8",
		Filename:     "myvideo",
		SegmentCount: 100,
		CreatedAt:    "2026-07-08T10:00:00+08:00",
	}
	if err := SaveTaskMeta(dir, meta); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadTaskMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.URL != meta.URL || loaded.Filename != meta.Filename || loaded.SegmentCount != meta.SegmentCount {
		t.Fatalf("loaded meta mismatch: %+v", loaded)
	}
	if loaded.Version != meta.Version {
		t.Fatalf("version mismatch: got %d, want %d", loaded.Version, meta.Version)
	}
	if loaded.CreatedAt != meta.CreatedAt {
		t.Fatalf("created_at mismatch: got %q, want %q", loaded.CreatedAt, meta.CreatedAt)
	}
}

func TestLoadTaskMeta_invalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, taskMetaFileName)
	if err := os.WriteFile(path, []byte("{not valid json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadTaskMeta(dir)
	if err == nil {
		t.Fatal("expected parse error for invalid JSON")
	}
}

func TestLoadTaskMeta_notExist(t *testing.T) {
	dir := t.TempDir()
	meta, err := LoadTaskMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if meta != nil {
		t.Fatalf("expected nil meta, got %+v", meta)
	}
}

func TestValidateTaskMeta_mismatch(t *testing.T) {
	existing := &TaskMeta{URL: "https://a.com/v1.m3u8", Filename: "v1", SegmentCount: 500}
	current := &TaskMeta{URL: "https://a.com/v2.m3u8", Filename: "v2", SegmentCount: 480}
	err := ValidateTaskMeta(existing, current)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateTaskMeta_match(t *testing.T) {
	meta := &TaskMeta{URL: "https://a.com/v1.m3u8", Filename: "v1", SegmentCount: 500}
	if err := ValidateTaskMeta(meta, meta); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveTaskMeta_notExist(t *testing.T) {
	dir := t.TempDir()
	if err := RemoveTaskMeta(dir); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveTaskMeta(t *testing.T) {
	dir := t.TempDir()
	meta := &TaskMeta{Version: taskMetaVersion, URL: "u", Filename: "f", SegmentCount: 1, CreatedAt: "t"}
	if err := SaveTaskMeta(dir, meta); err != nil {
		t.Fatal(err)
	}
	if err := RemoveTaskMeta(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, taskMetaFileName)); !os.IsNotExist(err) {
		t.Fatalf("expected meta file removed, stat err=%v", err)
	}
}
