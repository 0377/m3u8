package tool

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestGetRange(t *testing.T) {
	payload := []byte("0123456789abcdef")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ra := r.Header.Get("Range")
		if !strings.HasPrefix(ra, "bytes=") {
			http.Error(w, "missing range", http.StatusBadRequest)
			return
		}
		parts := strings.Split(strings.TrimPrefix(ra, "bytes="), "-")
		if len(parts) != 2 {
			http.Error(w, "bad range", http.StatusBadRequest)
			return
		}
		start, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			http.Error(w, "bad start", http.StatusBadRequest)
			return
		}
		end, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			http.Error(w, "bad end", http.StatusBadRequest)
			return
		}
		if end >= uint64(len(payload)) {
			end = uint64(len(payload)) - 1
		}
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(payload[start : end+1])
	}))
	defer srv.Close()

	body, err := GetRange(srv.URL, 2, 4, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "2345" {
		t.Fatalf("got %q, want %q", got, "2345")
	}
}

func TestGetRange_rejects200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("full-body"))
	}))
	defer srv.Close()

	_, err := GetRange(srv.URL, 0, 4, nil)
	if err == nil {
		t.Fatal("expected error when server returns 200 instead of 206")
	}
}
