package api

import (
	"errors"
	"time"
)

var (
	ErrTooManyTasks = errors.New("too many running tasks")
	ErrTaskNotFound = errors.New("task not found")
)

const Version = "1.3.0"

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusExpired   TaskStatus = "expired"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type ParseRequest struct {
	URL   string `json:"url"`
	Proxy string `json:"proxy,omitempty"`
}

type SegmentInfo struct {
	Index    int     `json:"index"`
	URI      string  `json:"uri"`
	Duration float32 `json:"duration"`
}

type ParseResponse struct {
	URL            string        `json:"url"`
	PlaylistType   string        `json:"playlist_type"`
	Version        int8          `json:"version"`
	TargetDuration float64       `json:"target_duration"`
	SegmentCount   int           `json:"segment_count"`
	TotalDuration  float64       `json:"total_duration"`
	Encrypted      bool          `json:"encrypted"`
	Segments       []SegmentInfo `json:"segments"`
}

type CreateTaskRequest struct {
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	Concurrency int    `json:"concurrency"`
	ToMP4       *bool  `json:"to_mp4"`
	Proxy       string `json:"proxy,omitempty"`
}

type CreateTaskResponse struct {
	TaskID    string     `json:"task_id"`
	Status    TaskStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

type OutputInfo struct {
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
	Format    string `json:"format"`
}

type TaskResponse struct {
	TaskID       string      `json:"task_id"`
	Status       TaskStatus  `json:"status"`
	Progress     float64     `json:"progress,omitempty"`
	Message      string      `json:"message,omitempty"`
	SegmentTotal int         `json:"segment_total,omitempty"`
	SegmentDone  int         `json:"segment_done,omitempty"`
	Error        string      `json:"error,omitempty"`
	Output       *OutputInfo `json:"output,omitempty"`
	DownloadURL  string      `json:"download_url,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	CompletedAt  *time.Time  `json:"completed_at,omitempty"`
	ExpiresAt    *time.Time  `json:"expires_at,omitempty"`
}

type TaskRecord struct {
	TaskID       string      `json:"task_id"`
	URL          string      `json:"url"`
	Filename     string      `json:"filename"`
	Concurrency  int         `json:"concurrency"`
	ToMP4        bool        `json:"to_mp4"`
	Proxy        string      `json:"proxy,omitempty"`
	Status       TaskStatus  `json:"status"`
	Progress     float64     `json:"progress"`
	Message      string      `json:"message"`
	SegmentTotal int         `json:"segment_total"`
	SegmentDone  int         `json:"segment_done"`
	Error        string      `json:"error"`
	Output       *OutputInfo `json:"output"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	CompletedAt  *time.Time  `json:"completed_at"`
	ExpiresAt    *time.Time  `json:"expires_at"`
	Cancelled    bool        `json:"cancelled"`
}
