package dl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestProgressReporterCalled(t *testing.T) {
	d := &Downloader{segLen: 10, finish: 0}
	var calls []int
	d.SetProgressReporter(func(done, total int, message string) {
		calls = append(calls, done)
	})
	d.reportProgress("downloading")
	if len(calls) != 1 || calls[0] != 0 {
		t.Fatalf("expected one call with done=0, got %v", calls)
	}
}

func TestStartRespectsCancel(t *testing.T) {
	d := &Downloader{segLen: 100, finish: 0}
	ctx, cancel := context.WithCancel(context.Background())
	d.SetCancelContext(ctx)
	cancel()
	err := d.Start(1, false, 0)
	if err != context.Canceled {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func newVODM3U8Server(t *testing.T, segmentCount int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/index.m3u8", func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		b.WriteString("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:10\n")
		for i := 0; i < segmentCount; i++ {
			fmt.Fprintf(&b, "#EXTINF:10.0,\nseg%d.ts\n", i)
		}
		b.WriteString("#EXT-X-ENDLIST\n")
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		fmt.Fprint(w, b.String())
	})
	for i := 0; i < segmentCount; i++ {
		mux.HandleFunc(fmt.Sprintf("/seg%d.ts", i), func(w http.ResponseWriter, r *http.Request) {
			seg := []byte{0x47}
			w.Header().Set("Content-Type", "video/mp2t")
			_, _ = w.Write(seg)
		})
	}
	return httptest.NewServer(mux)
}

func TestResumeSkip(t *testing.T) {
	srv := newVODM3U8Server(t, 3)
	defer srv.Close()

	dir := t.TempDir()
	url := srv.URL + "/index.m3u8"
	filename := "video"

	d1, err := NewTask(dir, url, filename, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(d1.queue) != 3 {
		t.Fatalf("first NewTask queue len=%d, want 3", len(d1.queue))
	}
	meta, err := LoadTaskMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if meta == nil {
		t.Fatal("expected task meta to be created")
	}
	if meta.Filename != filename || meta.SegmentCount != 3 || meta.URL != url {
		t.Fatalf("unexpected meta: %+v", meta)
	}

	tsDir := filepath.Join(dir, tsFolderName)
	for _, idx := range []int{0, 1} {
		fPath := filepath.Join(tsDir, tsFilename(idx))
		if err := os.WriteFile(fPath, []byte{0x47}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	d2, err := NewTask(dir, url, filename, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(d2.queue) != 1 {
		t.Fatalf("second NewTask queue len=%d, want 1", len(d2.queue))
	}
	if d2.queue[0] != 2 {
		t.Fatalf("queue[0]=%d, want 2", d2.queue[0])
	}
	if got := atomic.LoadInt32(&d2.finish); got != 2 {
		t.Fatalf("finish=%d, want 2", got)
	}
}

func TestResumeSkip_metaMismatch(t *testing.T) {
	srv := newVODM3U8Server(t, 3)
	defer srv.Close()

	dir := t.TempDir()
	url := srv.URL + "/index.m3u8"

	if _, err := NewTask(dir, url, "video1", nil, nil); err != nil {
		t.Fatal(err)
	}

	_, err := NewTask(dir, url, "video2", nil, nil)
	if err == nil {
		t.Fatal("expected error for filename mismatch")
	}
}
