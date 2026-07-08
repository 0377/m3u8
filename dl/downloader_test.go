package dl

import "testing"

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
