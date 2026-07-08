package crypt

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type RegistryOptions struct {
	ScriptsDir      string
	ScriptsDirAbs   string
	ConfigPath      string
	CLIScript       string
	Config          *Config
	ExternalTimeout time.Duration
}

type Registry struct {
	opts    RegistryOptions
	builtin *BuiltinDecryptor
	cache   map[string]Decryptor
	mu      sync.Mutex
}

func NewRegistry(opts RegistryOptions) (*Registry, error) {
	if opts.ScriptsDirAbs == "" && opts.ScriptsDir != "" {
		abs, err := filepath.Abs(opts.ScriptsDir)
		if err != nil {
			return nil, err
		}
		opts.ScriptsDirAbs = abs
	}
	if opts.ExternalTimeout == 0 {
		opts.ExternalTimeout = 30 * time.Second
	}
	return &Registry{
		opts:    opts,
		builtin: &BuiltinDecryptor{},
		cache:   make(map[string]Decryptor),
	}, nil
}

func (r *Registry) Resolve(ctx *Context) (Decryptor, error) {
	cacheKey, scriptPath, useBuiltin, err := r.resolveTarget(ctx)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if d, ok := r.cache[cacheKey]; ok {
		return d, nil
	}

	var d Decryptor
	if useBuiltin {
		d = r.builtin
	} else {
		d, err = r.loadScript(scriptPath)
		if err != nil {
			return nil, err
		}
	}
	r.cache[cacheKey] = d
	return d, nil
}

func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for _, d := range r.cache {
		if c, ok := d.(interface{ Close() error }); ok {
			if err := c.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	r.cache = make(map[string]Decryptor)
	return errors.Join(errs...)
}

func (r *Registry) resolveTarget(ctx *Context) (cacheKey, scriptPath string, useBuiltin bool, err error) {
	if r.opts.CLIScript != "" {
		resolved, e := resolveScriptPath(r.opts.CLIScript, r.opts.ScriptsDirAbs, r.opts.ConfigPath)
		if e != nil {
			return "", "", false, e
		}
		abs, e := filepath.Abs(resolved)
		if e != nil {
			return "", "", false, e
		}
		return abs, resolved, false, nil
	}
	if rule := r.matchConfigRule(ctx); rule != "" {
		resolved, err := resolveScriptPath(rule, r.opts.ScriptsDirAbs, r.opts.ConfigPath)
		if err != nil {
			return "", "", false, err
		}
		abs, e := filepath.Abs(resolved)
		if e != nil {
			return "", "", false, e
		}
		return abs, resolved, false, nil
	}
	if script := r.autoDiscover(ctx); script != "" {
		abs, e := filepath.Abs(script)
		if e != nil {
			return "", "", false, e
		}
		return abs, script, false, nil
	}
	if ctx.Method == "AES-128" || ctx.Method == "" || ctx.Method == "NONE" {
		return "builtin", "", true, nil
	}
	return "", "", false, fmt.Errorf(
		`unsupported encryption method %q, add script to scripts/ or use -decrypt-script`,
		ctx.Method,
	)
}

func (r *Registry) matchConfigRule(ctx *Context) string {
	if r.opts.Config == nil {
		return ""
	}
	u, _ := url.Parse(ctx.M3U8URL)
	host := ""
	if u != nil {
		host = u.Hostname()
	}
	for _, rule := range r.opts.Config.Rules {
		if rule.Match.Method != "" && rule.Match.Method != ctx.Method {
			continue
		}
		if rule.Match.Host != "" && !matchHost(rule.Match.Host, host) {
			continue
		}
		if rule.Match.URL != "" && !strings.Contains(ctx.M3U8URL, rule.Match.URL) {
			continue
		}
		return rule.Script
	}
	return ""
}

func matchHost(pattern, host string) bool {
	if pattern == host {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(host, suffix)
	}
	return false
}

func (r *Registry) autoDiscover(ctx *Context) string {
	if r.opts.ScriptsDirAbs == "" {
		return ""
	}
	u, _ := url.Parse(ctx.M3U8URL)
	host := ""
	if u != nil {
		host = u.Hostname()
	}
	candidates := []string{}
	if ctx.Method != "" {
		candidates = append(candidates,
			filepath.Join(r.opts.ScriptsDirAbs, ctx.Method+".star"),
			filepath.Join(r.opts.ScriptsDirAbs, ctx.Method+".py"),
		)
	}
	if host != "" {
		candidates = append(candidates,
			filepath.Join(r.opts.ScriptsDirAbs, host+".star"),
			filepath.Join(r.opts.ScriptsDirAbs, host+".py"),
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func (r *Registry) loadScript(path string) (Decryptor, error) {
	resolved, err := resolveScriptPath(path, r.opts.ScriptsDirAbs, r.opts.ConfigPath)
	if err != nil {
		return nil, err
	}
	if err := validateScriptPath(resolved, r.opts); err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(resolved))
	switch ext {
	case ".star":
		return newStarlarkDecryptor(resolved)
	default:
		return newExternalDecryptor(resolved, r.opts.ExternalTimeout)
	}
}

func resolveScriptPath(path, scriptsDirAbs, configPath string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty script path")
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	candidates := []string{}
	if configPath != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(configPath), path))
	}
	if scriptsDirAbs != "" {
		candidates = append(candidates, filepath.Join(scriptsDirAbs, path))
	}
	candidates = append(candidates, path)
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, err := filepath.Abs(c)
			if err != nil {
				return "", err
			}
			return abs, nil
		}
	}
	return filepath.Abs(path)
}

func validateScriptPath(path string, opts RegistryOptions) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if opts.CLIScript != "" {
		cliAbs, err := filepath.Abs(opts.CLIScript)
		if err == nil && abs == cliAbs {
			return nil
		}
		resolvedCLI, err := resolveScriptPath(opts.CLIScript, opts.ScriptsDirAbs, opts.ConfigPath)
		if err == nil {
			cliAbs, err = filepath.Abs(resolvedCLI)
			if err == nil && abs == cliAbs {
				return nil
			}
		}
	}
	if opts.ScriptsDirAbs != "" {
		rel, err := filepath.Rel(opts.ScriptsDirAbs, abs)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return nil
		}
	}
	if opts.ConfigPath != "" {
		configDir := filepath.Dir(opts.ConfigPath)
		rel, err := filepath.Rel(configDir, abs)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return nil
		}
	}
	return fmt.Errorf("script path %q must be under scripts_dir, config directory, or specified via -decrypt-script", path)
}
