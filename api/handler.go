package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/0377/m3u8/crypt"
	"github.com/0377/m3u8/parse"
	"github.com/go-chi/chi/v5"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB

type TaskManager interface {
	Create(req *CreateTaskRequest, maxRetry int) (*TaskRecord, error)
	Get(taskID string) (*TaskRecord, error)
	List(status string, limit, offset int) ([]*TaskRecord, error)
	Cancel(taskID string) error
	ToResponse(rec *TaskRecord) TaskResponse
	TaskDir(taskID string) string
	CryptService() *crypt.Service
}

type Handler struct {
	manager TaskManager
}

func NewHandler(m TaskManager) *Handler {
	return &Handler{manager: m}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok", Version: Version})
}

func (h *Handler) Parse(w http.ResponseWriter, r *http.Request) {
	var req ParseRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "url 为必填项")
		return
	}
	result, err := parse.FromURL(req.URL, nil, h.manager.CryptService())
	if err != nil {
		writeError(w, http.StatusBadRequest, "PARSE_FAILED", err.Error())
		return
	}
	full := r.URL.Query().Get("full") == "true"
	resp := buildParseResponse(result, full)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "url 为必填项")
		return
	}
	rec, err := h.manager.Create(&req, 10)
	if errors.Is(err, ErrTooManyTasks) {
		writeError(w, http.StatusTooManyRequests, "TOO_MANY_TASKS", err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "PARSE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, CreateTaskResponse{
		TaskID: rec.TaskID, Status: rec.Status, CreatedAt: rec.CreatedAt,
	})
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	rec, err := h.manager.Get(taskID)
	if errors.Is(err, ErrTaskNotFound) {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.manager.ToResponse(rec))
}

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}
	tasks, err := h.manager.List(status, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	resp := make([]TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, h.manager.ToResponse(t))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) DownloadTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	rec, err := h.manager.Get(taskID)
	if errors.Is(err, ErrTaskNotFound) {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	}
	if rec.Status == TaskStatusExpired {
		writeError(w, http.StatusGone, "TASK_EXPIRED", "任务已过期")
		return
	}
	if rec.Status != TaskStatusCompleted || rec.Output == nil {
		writeError(w, http.StatusConflict, "TASK_NOT_READY", "任务尚未完成")
		return
	}
	filePath := filepath.Join(h.manager.TaskDir(taskID), rec.Output.Filename)
	http.ServeFile(w, r, filePath)
}

func (h *Handler) CancelTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if err := h.manager.Cancel(taskID); errors.Is(err, ErrTaskNotFound) {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	} else if err != nil {
		writeError(w, http.StatusConflict, "TASK_NOT_READY", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func buildParseResponse(result *parse.Result, full bool) ParseResponse {
	m3u8 := result.M3u8
	var totalDuration float64
	for _, seg := range m3u8.Segments {
		totalDuration += float64(seg.Duration)
	}

	limit := len(m3u8.Segments)
	if !full && limit > 5 {
		limit = 5
	}
	segments := make([]SegmentInfo, 0, limit)
	for i := 0; i < limit; i++ {
		seg := m3u8.Segments[i]
		segments = append(segments, SegmentInfo{
			Index:    i,
			URI:      seg.URI,
			Duration: seg.Duration,
		})
	}

	return ParseResponse{
		URL:            result.URL.String(),
		PlaylistType:   string(m3u8.PlaylistType),
		Version:        m3u8.Version,
		TargetDuration: m3u8.TargetDuration,
		SegmentCount:   len(m3u8.Segments),
		TotalDuration:  totalDuration,
		Encrypted:      len(m3u8.Keys) > 0,
		Segments:       segments,
	}
}
