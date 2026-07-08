package tool

import (
	"io"
	"testing"
)

func TestGet(t *testing.T) {
	body, err := Get("https://raw.githubusercontent.com/0377/m3u8/master/README.md")
	if err != nil {
		t.Skipf("skip network test: %v", err)
	}
	defer body.Close()
	_, err = io.ReadAll(body)
	if err != nil {
		t.Error(err)
	}
}
