package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type ServerConfig struct {
	Port            int
	DataDir         string
	AuthEnabled     bool
	APIKey          string
	CORSOrigins     []string
	MaxTasks        int
	TaskTTL         time.Duration
	CleanupInterval time.Duration
}

type Server struct {
	*http.Server
	Handler http.Handler
	manager TaskManager
}

type managerFactory func(ServerConfig) (TaskManager, error)

var createManager managerFactory

func RegisterManagerFactory(f managerFactory) {
	createManager = f
}

func NewServer(cfg ServerConfig) (*Server, error) {
	if createManager == nil {
		return nil, fmt.Errorf("manager factory not registered")
	}
	mgr, err := createManager(cfg)
	if err != nil {
		return nil, err
	}

	h := NewHandler(mgr)
	r := chi.NewRouter()
	r.Use(RequestLogger)
	r.Use(CORSMiddleware(cfg.CORSOrigins))

	r.Get("/api/v1/health", h.Health)

	r.Group(func(r chi.Router) {
		r.Use(APIKeyMiddleware(cfg.AuthEnabled, cfg.APIKey))
		r.Post("/api/v1/parse", h.Parse)
		r.Post("/api/v1/tasks", h.CreateTask)
		r.Get("/api/v1/tasks", h.ListTasks)
		r.Get("/api/v1/tasks/{taskID}", h.GetTask)
		r.Get("/api/v1/tasks/{taskID}/download", h.DownloadTask)
		r.Delete("/api/v1/tasks/{taskID}", h.CancelTask)
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return &Server{Server: srv, Handler: r, manager: mgr}, nil
}

// Shutdown gracefully stops the HTTP server and releases manager resources.
func (s *Server) Shutdown(ctx context.Context) error {
	err := s.Server.Shutdown(ctx)
	if shutdownErr := s.manager.Shutdown(ctx); shutdownErr != nil && err == nil {
		err = shutdownErr
	}
	return err
}
