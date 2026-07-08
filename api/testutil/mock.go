package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

func fakeTSSegment() []byte {
	seg := make([]byte, 188)
	seg[0] = 0x47
	return seg
}

// NewM3U8Server returns an httptest server serving a VOD m3u8 playlist
// and fake TS segments with MPEG-TS sync byte 0x47.
func NewM3U8Server(segmentCount int) *httptest.Server {
	if segmentCount <= 0 {
		segmentCount = 1
	}

	tsData := fakeTSSegment()
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
			w.Header().Set("Content-Type", "video/mp2t")
			_, _ = w.Write(tsData)
		})
	}

	return httptest.NewServer(mux)
}
