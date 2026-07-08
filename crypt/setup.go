package crypt

import (
	"path/filepath"
	"time"
)

// ServiceOptions configures crypt.Service construction for CLI and API.
type ServiceOptions struct {
	DecryptScript string
	DecryptConfig string
	ScriptsDir    string
}

// BuildService loads decrypt.yaml (if present) and constructs a crypt.Service.
func BuildService(opts ServiceOptions) (*Service, error) {
	decryptConfig := opts.DecryptConfig
	if decryptConfig == "" {
		decryptConfig = "decrypt.yaml"
	}
	scriptsDir := opts.ScriptsDir
	if scriptsDir == "" {
		scriptsDir = "scripts"
	}

	cfg, err := LoadConfig(decryptConfig)
	if err != nil {
		return nil, err
	}
	if cfg != nil && cfg.ScriptsDir != "" {
		scriptsDir = cfg.ScriptsDir
	}

	scriptsAbs, err := filepath.Abs(scriptsDir)
	if err != nil {
		return nil, err
	}
	configAbs, _ := filepath.Abs(decryptConfig)

	timeout := 30 * time.Second
	if cfg != nil && cfg.ExternalTimeout > 0 {
		timeout = cfg.ExternalTimeout
	}

	reg, err := NewRegistry(RegistryOptions{
		ScriptsDir:      scriptsDir,
		ScriptsDirAbs:   scriptsAbs,
		ConfigPath:      configAbs,
		CLIScript:       opts.DecryptScript,
		Config:          cfg,
		ExternalTimeout: timeout,
	})
	if err != nil {
		return nil, err
	}
	return NewService(reg), nil
}
