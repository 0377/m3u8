package parse

import (
	"strings"
	"testing"
)

func TestResolveByteRangeOffsets(t *testing.T) {
	playlist := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXTINF:10.0,
#EXT-X-BYTERANGE:100@0
media.ts
#EXTINF:10.0,
#EXT-X-BYTERANGE:50
media.ts
#EXTINF:10.0,
#EXT-X-BYTERANGE:25@200
media.ts
#EXTINF:10.0,
#EXT-X-BYTERANGE:10
other.ts
#EXT-X-ENDLIST
`
	m3u8, err := parse(strings.NewReader(playlist))
	if err != nil {
		t.Fatal(err)
	}
	if len(m3u8.Segments) != 4 {
		t.Fatalf("segments=%d, want 4", len(m3u8.Segments))
	}
	want := []struct {
		offset, length uint64
	}{
		{0, 100},
		{100, 50},
		{200, 25},
		{0, 10},
	}
	for i, w := range want {
		seg := m3u8.Segments[i]
		if seg.Offset != w.offset || seg.Length != w.length {
			t.Fatalf("segment %d: offset=%d length=%d, want offset=%d length=%d",
				i, seg.Offset, seg.Length, w.offset, w.length)
		}
	}
}

func TestByteRangeWithoutExtInf(t *testing.T) {
	playlist := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXTINF:10.0,
#EXT-X-BYTERANGE:100@0
media.ts
#EXT-X-BYTERANGE:50
media.ts
#EXT-X-ENDLIST
`
	m3u8, err := parse(strings.NewReader(playlist))
	if err != nil {
		t.Fatal(err)
	}
	if len(m3u8.Segments) != 2 {
		t.Fatalf("segments=%d, want 2", len(m3u8.Segments))
	}
	if m3u8.Segments[0].Offset != 0 || m3u8.Segments[0].Length != 100 {
		t.Fatalf("segment 0: offset=%d length=%d", m3u8.Segments[0].Offset, m3u8.Segments[0].Length)
	}
	if m3u8.Segments[1].Offset != 100 || m3u8.Segments[1].Length != 50 {
		t.Fatalf("segment 1: offset=%d length=%d", m3u8.Segments[1].Offset, m3u8.Segments[1].Length)
	}
}
