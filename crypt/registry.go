package crypt

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RegistryOptions struct {
	ScriptsDir      string
	ScriptsDirAbs   string
	CLIScript       string
	Config          *Config
	ExternalTimeout time.Duration
}

type Registry struct {
	opts    RegistryOptions
	builtin *BuiltinDecryptor
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
	return &Registry{opts: opts, builtin: &BuiltinDecryptor{}}, nil
}

func (r *Registry) Resolve(ctx *Context) (Decryptor, error) {
	if r.opts.CLIScript != "" {
		return r.loadScript(r.opts.CLIScript)
	}
	if rule := r.matchConfigRule(ctx); rule != "" {
		return r.loadScript(rule)
	}
	if script := r.autoDiscover(ctx); script != "" {
		return r.loadScript(script)
	}
	if ctx.Method == "AES-128" || ctx.Method == "" || ctx.Method == "NONE" {
		return r.builtin, nil
	}
	return nil, fmt.Errorf(
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
	if err := validateScriptPath(path, r.opts.ScriptsDirAbs, r.opts.CLIScript); err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".star":
		return newStarlarkDecryptor(path)
	default:
		return newExternalDecryptor(path, r.opts.ExternalTimeout)
	}
}

func validateScriptPath(path, scriptsDirAbs, cliScript string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if cliScript != "" {
		cliAbs, err := filepath.Abs(cliScript)
		if err == nil && abs == cliAbs {
			return nil
		}
	}
	if scriptsDirAbs != "" {
		rel, err := filepath.Rel(scriptsDirAbs, abs)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return nil
		}
	}
	return fmt.Errorf("script path %q must be under scripts_dir or specified via -decrypt-script", path)
}
