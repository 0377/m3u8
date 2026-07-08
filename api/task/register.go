package task

import "github.com/0377/m3u8/api"

func init() {
	api.RegisterManagerFactory(func(cfg api.ServerConfig) (api.TaskManager, error) {
		mgr, err := NewManager(Config{
			DataDir:  cfg.DataDir,
			MaxTasks: cfg.MaxTasks,
			TaskTTL:  cfg.TaskTTL,
		})
		if err != nil {
			return nil, err
		}
		mgr.StartWorkers(cfg.MaxTasks)
		if err := mgr.Recover(); err != nil {
			return nil, err
		}
		mgr.StartCleanup(cfg.CleanupInterval)
		return mgr, nil
	})
}
