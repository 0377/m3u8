package tool

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPConfig holds optional HTTP client settings for M3U8 and segment requests.
type HTTPConfig struct {
	Headers     map[string]string
	Cookie      string
	Proxy       string
	InsecureTLS bool
}

// ValidateProxyURL checks that proxy is a supported http(s) proxy URL.
func ValidateProxyURL(proxy string) error {
	if proxy == "" {
		return nil
	}
	u, err := url.Parse(proxy)
	if err != nil {
		return fmt.Errorf("invalid proxy URL %q: %w", proxy, err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return fmt.Errorf("unsupported proxy scheme %q (use http or https)", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid proxy URL %q: missing host", proxy)
	}
	return nil
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

func (c *HTTPConfig) client() (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if c.InsecureTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	if c.Proxy != "" {
		proxyURL, err := url.Parse(c.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
	}, nil
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

// HTTPConfigFrom builds an HTTPConfig, or nil when all options are empty.
func HTTPConfigFrom(headers map[string]string, cookie, proxy string, insecureTLS bool) (*HTTPConfig, error) {
	if err := ValidateProxyURL(proxy); err != nil {
		return nil, err
	}
	if len(headers) == 0 && cookie == "" && proxy == "" && !insecureTLS {
		return nil, nil
	}
	return &HTTPConfig{
		Headers:     headers,
		Cookie:      cookie,
		Proxy:       proxy,
		InsecureTLS: insecureTLS,
	}, nil
}

// Get performs an HTTP GET with optional client configuration.
func Get(url string, cfg *HTTPConfig) (io.ReadCloser, error) {
	var client *http.Client
	if cfg == nil {
		client = defaultClient()
	} else {
		var err error
		client, err = cfg.client()
		if err != nil {
			return nil, err
		}
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
