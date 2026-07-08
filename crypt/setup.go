package crypt

import (
	"path/filepath"
	"time"
)

// ServiceOptions configures crypt.Service construction for CLI and API.
type ServiceOptions struct {
	DecryptScript  string
	DecryptConfig  string
	ScriptsDir     string
	ProviderParams ProviderParams
}

// BuildService loads decrypt.yaml (if present) and constructs a crypt.Service.
func BuildService(m3u8URL string, opts ServiceOptions) (processedURL string, svc *Service, err error) {
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
		return "", nil, err
	}
	if cfg != nil && cfg.ScriptsDir != "" {
		scriptsDir = cfg.ScriptsDir
	}

	scriptsAbs, err := filepath.Abs(scriptsDir)
	if err != nil {
		return "", nil, err
	}
	configAbs, _ := filepath.Abs(decryptConfig)

	timeout := 30 * time.Second
	if cfg != nil && cfg.ExternalTimeout > 0 {
		timeout = cfg.ExternalTimeout
	}

	processedURL = m3u8URL
	activeID := ""
	if providerIntegration.PreprocessURL != nil {
		processedURL = providerIntegration.PreprocessURL(m3u8URL, opts.ProviderParams)
	}
	if providerIntegration.DetectFromURL != nil {
		activeID = providerIntegration.DetectFromURL(processedURL)
	}
	if providerIntegration.ValidateParams != nil {
		if err := providerIntegration.ValidateParams(activeID, opts.ProviderParams); err != nil {
			return "", nil, err
		}
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
		return "", nil, err
	}
	svc = NewService(reg, ServiceProviderOptions{
		ActiveID: activeID,
		Params:   opts.ProviderParams,
	})
	return processedURL, svc, nil
}
