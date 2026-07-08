package crypt

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// maxExternalLineBytes is the maximum JSON line size for external script I/O.
// Segment responses include base64-encoded TS data and can exceed bufio.Scanner's 64KB default.
const maxExternalLineBytes = 16 << 20 // 16 MiB

type externalDecryptor struct {
	path    string
	timeout time.Duration
	procMu  sync.Mutex // protects process start/stop/restart
	callMu  sync.Mutex // serializes stdin/stdout request-response pairs
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	nextID  atomic.Int64
}

type extRequest struct {
	ID         int64  `json:"id"`
	Hook       string `json:"hook"`
	Method     string `json:"method,omitempty"`
	RawKey     string `json:"raw_key,omitempty"`
	Key        string `json:"key,omitempty"`
	IV         string `json:"iv,omitempty"`
	M3U8URL    string `json:"m3u8_url,omitempty"`
	Ciphertext string `json:"ciphertext,omitempty"`
	Index      int    `json:"index,omitempty"`
	URI        string `json:"uri,omitempty"`
}

type extResponse struct {
	ID    int64  `json:"id"`
	OK    bool   `json:"ok"`
	Key   string `json:"key,omitempty"`
	IV    string `json:"iv,omitempty"`
	Data  string `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func newExternalDecryptor(path string, timeout time.Duration) (Decryptor, error) {
	d := &externalDecryptor{path: path, timeout: timeout}
	if err := d.start(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *externalDecryptor) Name() string { return filepath.Base(d.path) }

func (d *externalDecryptor) Close() error {
	d.callMu.Lock()
	defer d.callMu.Unlock()
	d.procMu.Lock()
	defer d.procMu.Unlock()
	return d.stopLocked()
}

func (d *externalDecryptor) stopLocked() error {
	if d.stdin != nil {
		_ = d.stdin.Close()
		d.stdin = nil
	}
	if d.cmd != nil && d.cmd.Process != nil {
		_ = d.cmd.Process.Kill()
		_ = d.cmd.Wait()
	}
	d.cmd = nil
	d.stdout = nil
	return nil
}

func (d *externalDecryptor) start() error {
	d.procMu.Lock()
	defer d.procMu.Unlock()
	return d.startLocked()
}

func (d *externalDecryptor) startLocked() error {
	var cmd *exec.Cmd
	if strings.HasSuffix(strings.ToLower(d.path), ".py") {
		cmd = exec.Command("python3", d.path)
	} else {
		cmd = exec.Command(d.path)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start external script %s: %w", d.path, err)
	}
	d.cmd = cmd
	d.stdin = stdin
	d.stdout = bufio.NewReader(stdout)
	return nil
}

func (d *externalDecryptor) restart() error {
	d.callMu.Lock()
	defer d.callMu.Unlock()
	d.procMu.Lock()
	defer d.procMu.Unlock()
	_ = d.stopLocked()
	return d.startLocked()
}

func (d *externalDecryptor) call(req extRequest) (extResponse, error) {
	resp, err := d.doCall(req)
	if err != nil && isTimeoutErr(err) {
		if rerr := d.restart(); rerr == nil {
			return d.doCall(req)
		}
	}
	if err != nil && isDeadProcessErr(err) {
		_ = d.restart()
	}
	return resp, err
}

func isDeadProcessErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "stdout closed") || strings.Contains(msg, "broken pipe")
}

func isTimeoutErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "timeout")
}

func (d *externalDecryptor) doCall(req extRequest) (extResponse, error) {
	d.callMu.Lock()
	defer d.callMu.Unlock()

	if d.stdin == nil || d.stdout == nil {
		return extResponse{}, fmt.Errorf("external script stdout closed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()

	line, err := json.Marshal(req)
	if err != nil {
		return extResponse{}, err
	}
	if _, err := fmt.Fprintf(d.stdin, "%s\n", line); err != nil {
		return extResponse{}, err
	}

	type result struct {
		resp extResponse
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		respLine, err := readExternalLine(d.stdout, maxExternalLineBytes)
		if err != nil {
			if err == io.EOF {
				ch <- result{err: fmt.Errorf("external script stdout closed")}
				return
			}
			ch <- result{err: err}
			return
		}
		var resp extResponse
		if err := json.Unmarshal(respLine, &resp); err != nil {
			ch <- result{err: err}
			return
		}
		ch <- result{resp: resp}
	}()

	select {
	case <-ctx.Done():
		d.procMu.Lock()
		if d.cmd != nil && d.cmd.Process != nil {
			_ = d.cmd.Process.Kill()
		}
		_ = d.stopLocked()
		d.procMu.Unlock()
		return extResponse{}, fmt.Errorf("external script timeout after %s", d.timeout)
	case r := <-ch:
		if r.err != nil {
			return extResponse{}, r.err
		}
		if r.resp.ID != req.ID {
			return extResponse{}, fmt.Errorf("response id mismatch: got %d want %d", r.resp.ID, req.ID)
		}
		if !r.resp.OK {
			if r.resp.Error == "not implemented" {
				return r.resp, ErrNotImplemented
			}
			return r.resp, fmt.Errorf("external script error: %s", r.resp.Error)
		}
		return r.resp, nil
	}
}

func readExternalLine(r *bufio.Reader, maxBytes int) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	line = bytes.TrimSuffix(line, []byte{'\n'})
	line = bytes.TrimSuffix(line, []byte{'\r'})
	if len(line) > maxBytes {
		return nil, fmt.Errorf("external script response exceeds %d bytes", maxBytes)
	}
	if err == io.EOF && len(line) == 0 {
		return nil, io.EOF
	}
	return line, nil
}

func decodeExternalIV(fallback, encoded string) []byte {
	if encoded == "" {
		return []byte(fallback)
	}
	if iv, err := base64.StdEncoding.DecodeString(encoded); err == nil {
		return iv
	}
	return []byte(encoded)
}

func (d *externalDecryptor) ProcessKey(ctx *Context, rawKey []byte, meta *KeyMeta) ([]byte, []byte, error) {
	id := d.nextID.Add(1)
	iv := ""
	if meta != nil {
		iv = meta.IV
	}
	resp, err := d.call(extRequest{
		ID: id, Hook: "key", Method: ctx.Method,
		RawKey: base64.StdEncoding.EncodeToString(rawKey),
		IV: iv, M3U8URL: ctx.M3U8URL,
	})
	if err == ErrNotImplemented {
		return rawKey, []byte(iv), nil
	}
	if err != nil {
		return nil, nil, err
	}
	key, err := base64.StdEncoding.DecodeString(resp.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid key in response: %w", err)
	}
	return key, decodeExternalIV(iv, resp.IV), nil
}

func (d *externalDecryptor) DecryptSegment(ctx *Context, ciphertext, key, iv []byte) ([]byte, error) {
	id := d.nextID.Add(1)
	resp, err := d.call(extRequest{
		ID: id, Hook: "segment",
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Key:        base64.StdEncoding.EncodeToString(key),
		IV:         base64.StdEncoding.EncodeToString(iv), Index: ctx.SegmentIdx, URI: ctx.SegmentURI,
	})
	if err == ErrNotImplemented {
		return nil, ErrNotImplemented
	}
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(resp.Data)
}

func (d *externalDecryptor) DecryptFull(ctx *Context, ciphertext []byte) ([]byte, bool, error) {
	id := d.nextID.Add(1)
	resp, err := d.call(extRequest{
		ID: id, Hook: "full",
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Key:        base64.StdEncoding.EncodeToString(ctx.Key),
		IV:         base64.StdEncoding.EncodeToString(ctx.IV),
		Index:      ctx.SegmentIdx, URI: ctx.SegmentURI, Method: ctx.Method,
	})
	if err == ErrNotImplemented {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	out, err := base64.StdEncoding.DecodeString(resp.Data)
	return out, true, err
}
