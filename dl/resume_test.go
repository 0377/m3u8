package dl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanCompletedSegments(t *testing.T) {
	dir := t.TempDir()
	tsFolder := filepath.Join(dir, tsFolderName)
	if err := os.MkdirAll(tsFolder, 0755); err != nil {
		t.Fatal(err)
	}
	// 0.ts 和 2.ts 已完成
	for _, idx := range []int{0, 2} {
		f := filepath.Join(tsFolder, tsFilename(idx))
		if err := os.WriteFile(f, []byte{0x47}, 0644); err != nil {
			t.Fatal(err)
		}
	}
	// 1.ts_tmp 残留
	tmp := filepath.Join(tsFolder, tsFilename(1)+tsTempFileSuffix)
	if err := os.WriteFile(tmp, []byte("partial"), 0644); err != nil {
		t.Fatal(err)
	}

	completed, err := scanCompletedSegments(tsFolder, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(completed) != 2 {
		t.Fatalf("expected 2 completed, got %d", len(completed))
	}
	if _, ok := completed[0]; !ok {
		t.Fatal("expected segment 0 completed")
	}
	if _, ok := completed[2]; !ok {
		t.Fatal("expected segment 2 completed")
	}
	// _tmp 应被删除
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatal("expected tmp file removed")
	}
}

func TestScanCompletedSegments_emptyFile(t *testing.T) {
	dir := t.TempDir()
	tsFolder := filepath.Join(dir, tsFolderName)
	if err := os.MkdirAll(tsFolder, 0755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(tsFolder, tsFilename(0))
	if err := os.WriteFile(f, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	completed, err := scanCompletedSegments(tsFolder, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(completed) != 0 {
		t.Fatalf("expected 0 completed, got %d", len(completed))
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Fatal("expected empty file removed")
	}
}

func TestScanCompletedSegments_invalidSyncByte(t *testing.T) {
	dir := t.TempDir()
	tsFolder := filepath.Join(dir, tsFolderName)
	if err := os.MkdirAll(tsFolder, 0755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(tsFolder, tsFilename(0))
	tmp := f + tsTempFileSuffix
	if err := os.WriteFile(f, []byte{0x00, 0x01, 0x02}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmp, []byte("partial"), 0644); err != nil {
		t.Fatal(err)
	}

	completed, err := scanCompletedSegments(tsFolder, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(completed) != 0 {
		t.Fatalf("expected 0 completed, got %d", len(completed))
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Fatal("expected invalid file removed")
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatal("expected tmp sibling removed")
	}
}

func TestBuildQueue(t *testing.T) {
	completed := map[int]struct{}{0: {}, 2: {}}
	queue := buildQueue(4, completed)
	want := []int{1, 3}
	if len(queue) != len(want) {
		t.Fatalf("queue len=%d, want %d", len(queue), len(want))
	}
	for i, v := range want {
		if queue[i] != v {
			t.Fatalf("queue[%d]=%d, want %d", i, queue[i], v)
		}
	}
}

func TestBuildQueue_allCompleted(t *testing.T) {
	completed := map[int]struct{}{0: {}, 1: {}}
	queue := buildQueue(2, completed)
	if len(queue) != 0 {
		t.Fatalf("expected empty queue, got %v", queue)
	}
}
