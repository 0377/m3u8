package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0377/m3u8/api"
	_ "github.com/0377/m3u8/api/task"
	"github.com/0377/m3u8/api/testutil"
)

func newTestAPIServer(t *testing.T, auth bool) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	srv, err := api.NewServer(api.ServerConfig{
		Port:            0,
		DataDir:         dir,
		AuthEnabled:     auth,
		APIKey:          "test-key",
		CORSOrigins:     []string{"*"},
		MaxTasks:        3,
		TaskTTL:         time.Hour,
		CleanupInterval: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(srv.Handler)
}

func TestServerHealth(t *testing.T) {
	srv := newTestAPIServer(t, false)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var health api.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	if health.Status != "ok" {
		t.Fatalf("want ok, got %s", health.Status)
	}
}

func TestServerParseAndDownload(t *testing.T) {
	m3u8Srv := testutil.NewM3U8Server(2)
	defer m3u8Srv.Close()

	apiSrv := newTestAPIServer(t, false)
	defer apiSrv.Close()

	m3u8URL := m3u8Srv.URL + "/index.m3u8"

	parseBody, err := json.Marshal(api.ParseRequest{URL: m3u8URL})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(apiSrv.URL+"/api/v1/parse", "application/json", bytes.NewReader(parseBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("parse want 200, got %d: %s", resp.StatusCode, body)
	}

	var parseResp api.ParseResponse
	if err := json.NewDecoder(resp.Body).Decode(&parseResp); err != nil {
		t.Fatal(err)
	}
	if parseResp.SegmentCount != 2 {
		t.Fatalf("want 2 segments, got %d", parseResp.SegmentCount)
	}

	toMP4 := false
	createBody, err := json.Marshal(api.CreateTaskRequest{
		URL:         m3u8URL,
		Filename:    "test",
		Concurrency: 2,
		ToMP4:       &toMP4,
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.Post(apiSrv.URL+"/api/v1/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create task want 202, got %d: %s", resp.StatusCode, body)
	}

	var createResp api.CreateTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatal(err)
	}
	taskID := createResp.TaskID
	if taskID == "" {
		t.Fatal("empty task_id")
	}

	deadline := time.Now().Add(30 * time.Second)
	var task api.TaskResponse
	for time.Now().Before(deadline) {
		resp, err = http.Get(apiSrv.URL + "/api/v1/tasks/" + taskID)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			resp.Body.Close()
			t.Fatal(err)
		}
		resp.Body.Close()
		if task.Status == api.TaskStatusCompleted {
			break
		}
		if task.Status == api.TaskStatusFailed {
			t.Fatalf("task failed: %s", task.Error)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if task.Status != api.TaskStatusCompleted {
		t.Fatalf("task not completed, status=%s", task.Status)
	}

	resp, err = http.Get(apiSrv.URL + "/api/v1/tasks/" + taskID + "/download")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("download want 200, got %d: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("empty download body")
	}
}

func TestServerAuth(t *testing.T) {
	srv := newTestAPIServer(t, true)
	defer srv.Close()

	resp, err := http.Post(
		srv.URL+"/api/v1/parse",
		"application/json",
		bytes.NewReader([]byte(`{"url":"http://example.com"}`)),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
}
