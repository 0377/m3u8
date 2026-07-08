package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/0377/m3u8/api"
	"github.com/google/uuid"
)

type Store struct {
	baseDir string
	mu      sync.RWMutex
}

func NewStore(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

func (s *Store) TaskDir(taskID string) string {
	return filepath.Join(s.baseDir, taskID)
}

func (s *Store) taskFile(taskID string) string {
	return filepath.Join(s.TaskDir(taskID), "task.json")
}

func (s *Store) Save(rec *api.TaskRecord) error {
	if _, err := uuid.Parse(rec.TaskID); err != nil {
		return fmt.Errorf("invalid task id: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.TaskDir(rec.TaskID), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.taskFile(rec.TaskID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.taskFile(rec.TaskID))
}

func (s *Store) Load(taskID string) (*api.TaskRecord, error) {
	if _, err := uuid.Parse(taskID); err != nil {
		return nil, fmt.Errorf("invalid task id: %w", err)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(s.taskFile(taskID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("task not found")
		}
		return nil, err
	}
	var rec api.TaskRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *Store) ListAll() ([]*api.TaskRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []*api.TaskRecord
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rec, err := s.loadUnlocked(e.Name())
		if err != nil {
			continue
		}
		result = append(result, rec)
	}
	return result, nil
}

func (s *Store) loadUnlocked(taskID string) (*api.TaskRecord, error) {
	data, err := os.ReadFile(s.taskFile(taskID))
	if err != nil {
		return nil, err
	}
	var rec api.TaskRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *Store) Delete(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.RemoveAll(s.TaskDir(taskID))
}
