package dl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	taskMetaFileName = ".m3u8-task.json"
	taskMetaVersion  = 1
)

type TaskMeta struct {
	Version      int    `json:"version"`
	URL          string `json:"url"`
	Filename     string `json:"filename"`
	SegmentCount int    `json:"segment_count"`
	CreatedAt    string `json:"created_at"`
}

func LoadTaskMeta(dir string) (*TaskMeta, error) {
	path := filepath.Join(dir, taskMetaFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read task meta: %w", err)
	}
	var meta TaskMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse task meta: %w", err)
	}
	return &meta, nil
}

func SaveTaskMeta(dir string, meta *TaskMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal task meta: %w", err)
	}
	path := filepath.Join(dir, taskMetaFileName)
	tmpPath := filepath.Join(dir, taskMetaFileName+".tmp")
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write task meta: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("write task meta: %w", err)
	}
	return nil
}

func RemoveTaskMeta(dir string) error {
	path := filepath.Join(dir, taskMetaFileName)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove task meta: %w", err)
	}
	return nil
}

func ValidateTaskMeta(existing, current *TaskMeta) error {
	if existing == nil {
		return nil
	}
	if current == nil {
		return fmt.Errorf("current task meta is nil")
	}
	if existing.URL != current.URL {
		return taskMetaMismatchError(existing, current)
	}
	if existing.Filename != current.Filename {
		return taskMetaMismatchError(existing, current)
	}
	if existing.SegmentCount != current.SegmentCount {
		return taskMetaMismatchError(existing, current)
	}
	return nil
}

func taskMetaMismatchError(existing, current *TaskMeta) error {
	return fmt.Errorf(
		"输出目录存在未完成的下载任务，但参数不一致\n"+
			"  已有任务: url=%s, filename=%s, segments=%d\n"+
			"  当前参数: url=%s, filename=%s, segments=%d\n"+
			"  请删除 .m3u8-task.json 和 ts/ 目录后重试，或更换输出目录",
		existing.URL, existing.Filename, existing.SegmentCount,
		current.URL, current.Filename, current.SegmentCount,
	)
}

func NewTaskMeta(url, filename string, segmentCount int, createdAt string) *TaskMeta {
	return &TaskMeta{
		Version:      taskMetaVersion,
		URL:          url,
		Filename:     filename,
		SegmentCount: segmentCount,
		CreatedAt:    createdAt,
	}
}
