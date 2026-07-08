package crypt

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ScriptsDir      string        `yaml:"scripts_dir"`
	ExternalTimeout time.Duration `yaml:"external_timeout"`
	Rules           []Rule        `yaml:"rules"`
}

type Rule struct {
	Match  MatchSpec `yaml:"match"`
	Script string    `yaml:"script"`
}

type MatchSpec struct {
	Host   string `yaml:"host"`
	Method string `yaml:"method"`
	URL    string `yaml:"url"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.ExternalTimeout == 0 {
		cfg.ExternalTimeout = 30 * time.Second
	}
	return &cfg, nil
}
