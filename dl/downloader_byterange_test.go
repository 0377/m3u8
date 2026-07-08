package dl

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newByteRangeM3U8Server(t *testing.T) *httptest.Server {
	t.Helper()
	media := []byte{
		0x00, 0x00, 0x47, 0x01, 0x02,
		0x47, 0x03, 0x04, 0x05,
		0x47, 0x06, 0x07,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/index.m3u8", func(w http.ResponseWriter, r *http.Request) {
		playlist := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXTINF:10.0,
#EXT-X-BYTERANGE:3@2
media.ts
#EXTINF:10.0,
#EXT-X-BYTERANGE:4
media.ts
#EXTINF:10.0,
#EXT-X-BYTERANGE:3@9
media.ts
#EXT-X-ENDLIST
`
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		_, _ = w.Write([]byte(playlist))
	})
	mux.HandleFunc("/media.ts", func(w http.ResponseWriter, r *http.Request) {
		ra := r.Header.Get("Range")
		if ra == "" {
			http.Error(w, "range required", http.StatusBadRequest)
			return
		}
		var start, end uint64
		if _, err := fmt.Sscanf(ra, "bytes=%d-%d", &start, &end); err != nil {
			http.Error(w, "bad range", http.StatusBadRequest)
			return
		}
		if end >= uint64(len(media)) {
			end = uint64(len(media)) - 1
		}
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(media[start : end+1])
	})
	return httptest.NewServer(mux)
}

func TestDownloadByteRangePlaylist(t *testing.T) {
	srv := newByteRangeM3U8Server(t)
	defer srv.Close()

	dir := t.TempDir()
	playlistURL := srv.URL + "/index.m3u8"
	d, err := NewTask(dir, playlistURL, "video", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Start(1, false, 1); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "video.ts")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x47, 0x01, 0x02, 0x47, 0x03, 0x04, 0x05, 0x47, 0x06, 0x07}
	if string(data) != string(want) {
		t.Fatalf("merged ts = %v, want %v", data, want)
	}
}

func TestDownloadByteRangeUsesRangeHeader(t *testing.T) {
	var ranges []string
	media := []byte{0x47, 0x01, 0x02, 0x47, 0x03}
	mux := http.NewServeMux()
	mux.HandleFunc("/index.m3u8", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.TrimSpace(`
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXTINF:10.0,
#EXT-X-BYTERANGE:2@0
media.ts
#EXT-X-ENDLIST
`))
	})
	mux.HandleFunc("/media.ts", func(w http.ResponseWriter, r *http.Request) {
		ranges = append(ranges, r.Header.Get("Range"))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(media[0:2])
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	d, err := NewTask(dir, srv.URL+"/index.m3u8", "video", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Start(1, false, 1); err != nil {
		t.Fatal(err)
	}
	if len(ranges) != 1 || ranges[0] != "bytes=0-1" {
		t.Fatalf("ranges=%v, want [bytes=0-1]", ranges)
	}
}
