package tool

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseHeaders(t *testing.T) {
	t.Parallel()

	headers, err := ParseHeaders([]string{
		"User-Agent: m3u8-cli",
		"Referer: https://example.com/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if headers["User-Agent"] != "m3u8-cli" {
		t.Fatalf("unexpected User-Agent: %q", headers["User-Agent"])
	}
	if headers["Referer"] != "https://example.com/" {
		t.Fatalf("unexpected Referer: %q", headers["Referer"])
	}

	if _, err := ParseHeaders([]string{"invalid"}); err == nil {
		t.Fatal("expected error for invalid header")
	}
}

func TestGetWithHTTPConfig(t *testing.T) {
	t.Parallel()

	var gotUA, gotCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotCookie = r.Header.Get("Cookie")
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	body, err := Get(server.URL, &HTTPConfig{
		Headers: map[string]string{"User-Agent": "test-agent"},
		Cookie:  "sid=1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = body.Close()

	if gotUA != "test-agent" {
		t.Fatalf("unexpected User-Agent: %q", gotUA)
	}
	if gotCookie != "sid=1" {
		t.Fatalf("unexpected Cookie: %q", gotCookie)
	}
}

func TestValidateProxyURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		proxy string
		ok    bool
	}{
		{"", true},
		{"http://127.0.0.1:7890", true},
		{"https://proxy.example.com:8080", true},
		{"socks5://127.0.0.1:1080", false},
		{"not-a-url", false},
		{"http://", false},
	}
	for _, tc := range cases {
		err := ValidateProxyURL(tc.proxy)
		if tc.ok && err != nil {
			t.Fatalf("proxy %q: unexpected error: %v", tc.proxy, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("proxy %q: expected error", tc.proxy)
		}
	}
}

func TestHTTPConfigFromProxy(t *testing.T) {
	t.Parallel()

	cfg, err := HTTPConfigFrom(nil, "", "http://127.0.0.1:7890", false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil || cfg.Proxy != "http://127.0.0.1:7890" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	client, err := cfg.client()
	if err != nil {
		t.Fatal(err)
	}
	if client.Transport == nil {
		t.Fatal("expected transport")
	}

	empty, err := HTTPConfigFrom(nil, "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if empty != nil {
		t.Fatalf("expected nil config, got %#v", empty)
	}
}
