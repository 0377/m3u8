package dl

import (
	"context"
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
