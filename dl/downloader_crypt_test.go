package dl

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/0377/m3u8/crypt"
	"github.com/0377/m3u8/tool"
)

func TestDownload_aes128_with_cryptSvc(t *testing.T) {
	plain := append([]byte{0x47}, bytes.Repeat([]byte("hello"), 1000)...)
	key := []byte("1234567890123456")
	enc, err := tool.AES128Encrypt(plain, key, nil)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/playlist.m3u8":
			_, _ = fmt.Fprintf(w, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-KEY:METHOD=AES-128,URI="key.bin"
#EXTINF:10.0,
seg0.ts
#EXT-X-ENDLIST
`)
		case "/key.bin":
			_, _ = w.Write(key)
		case "/seg0.ts":
			_, _ = w.Write(enc)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	reg, err := crypt.NewRegistry(crypt.RegistryOptions{ScriptsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	svc := crypt.NewService(reg)

	outDir := t.TempDir()
	d, err := NewTask(outDir, srv.URL+"/playlist.m3u8", "test", nil, svc)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.download(0); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(outDir, tsFolderName, tsFilename(0)))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plain) {
		t.Fatalf("decrypted segment: got %q want %q", got, plain)
	}
}

func TestDownload_unencrypted_skips_decrypt(t *testing.T) {
	raw := append([]byte{0x47}, bytes.Repeat([]byte("plain"), 1000)...)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/playlist.m3u8":
			_, _ = fmt.Fprintf(w, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:10.0,
seg0.ts
#EXT-X-ENDLIST
`)
		case "/seg0.ts":
			_, _ = w.Write(raw)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	reg, err := crypt.NewRegistry(crypt.RegistryOptions{ScriptsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	svc := crypt.NewService(reg)

	outDir := t.TempDir()
	d, err := NewTask(outDir, srv.URL+"/playlist.m3u8", "test", nil, svc)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.download(0); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(outDir, tsFolderName, tsFilename(0)))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(raw) {
		t.Fatalf("plain segment: got %q want %q", got, raw)
	}
}
