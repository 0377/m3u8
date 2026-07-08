package tool

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPConfig holds optional HTTP client settings for M3U8 and segment requests.
type HTTPConfig struct {
	Headers     map[string]string
	Cookie      string
	InsecureTLS bool
}

// ParseHeaders parses "Key: Value" lines into a header map.
func ParseHeaders(lines []string) (map[string]string, error) {
	headers := make(map[string]string, len(lines))
	for _, line := range lines {
		idx := strings.Index(line, ":")
		if idx <= 0 {
			return nil, fmt.Errorf("invalid header %q, expected \"Key: Value\"", line)
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if key == "" {
			return nil, fmt.Errorf("invalid header %q, empty key", line)
		}
		headers[key] = value
	}
	return headers, nil
}

func (c *HTTPConfig) client() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if c.InsecureTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
	}
}

func defaultClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

func (c *HTTPConfig) applyRequest(req *http.Request) {
	if c == nil {
		return
	}
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}
	if c.Cookie != "" {
		req.Header.Set("Cookie", c.Cookie)
	}
}

// Get performs an HTTP GET with optional client configuration.
func Get(url string, cfg *HTTPConfig) (io.ReadCloser, error) {
	var client *http.Client
	if cfg == nil {
		client = defaultClient()
	} else {
		client = cfg.client()
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	cfg.applyRequest(req)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("http error: status code %d", resp.StatusCode)
	}
	return resp.Body, nil
}
