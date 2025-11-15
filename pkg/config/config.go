package config

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

// Context stores the Snowflake connection profile.
type Context struct {
    Name        string `json:"-"`
    Account     string `json:"account,omitempty"`
    AccountURL  string `json:"accountUrl,omitempty"`
    User        string `json:"user,omitempty"`
    Role        string `json:"role,omitempty"`
    Warehouse   string `json:"warehouse,omitempty"`
    Database    string `json:"database,omitempty"`
    Schema      string `json:"schema,omitempty"`
    Description string `json:"description,omitempty"`
    AuthMethod  string `json:"authMethod,omitempty"`
}

// Config describes the snowctl configuration schema.
type Config struct {
    CurrentContext string              `json:"currentContext,omitempty"`
    DefaultContext string              `json:"defaultContext,omitempty"`
    Contexts       map[string]*Context `json:"contexts,omitempty"`
}

// DefaultConfig returns an initialized configuration.
func DefaultConfig() *Config {
    return &Config{Contexts: make(map[string]*Context)}
}

// Load reads configuration from disk or returns defaults when files are missing.
func Load() (*Config, error) {
    cfg := DefaultConfig()

    cfgPath, err := path()
    if err != nil {
        return nil, err
    }

    data, err := os.ReadFile(cfgPath)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return cfg, nil
        }
        return nil, fmt.Errorf("read config: %w", err)
    }

    if err := json.Unmarshal(data, cfg); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }

    cfg.ensureNames()
    return cfg, nil
}

// Save writes the configuration atomically to disk.
func Save(cfg *Config) error {
    if cfg == nil {
        return fmt.Errorf("nil config")
    }
    cfg.ensureNames()

    cfgPath, err := path()
    if err != nil {
        return err
    }

    dir := filepath.Dir(cfgPath)
    if err := os.MkdirAll(dir, 0o700); err != nil {
        return fmt.Errorf("create config dir: %w", err)
    }

    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal config: %w", err)
    }

    tmpFile := cfgPath + ".tmp"
    if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
        return fmt.Errorf("write config temp file: %w", err)
    }
    if err := os.Rename(tmpFile, cfgPath); err != nil {
        return fmt.Errorf("persist config: %w", err)
    }
    return nil
}

// GetContext returns the named context if present.
func (c *Config) GetContext(name string) (*Context, bool) {
    if c == nil {
        return nil, false
    }
    ctx, ok := c.Contexts[name]
    if ok && ctx != nil {
        ctx.Name = name
    }
    return ctx, ok
}

// SetContext creates or updates a context.
func (c *Config) SetContext(name string, ctx *Context) {
    if c.Contexts == nil {
        c.Contexts = make(map[string]*Context)
    }
    copied := *ctx
    copied.Name = name
    c.Contexts[name] = &copied
    if c.CurrentContext == "" {
        c.CurrentContext = name
    }
    if c.DefaultContext == "" {
        c.DefaultContext = name
    }
}

// DeleteContext removes a context from the configuration.
func (c *Config) DeleteContext(name string) {
    if c.Contexts == nil {
        return
    }
    delete(c.Contexts, name)
    if c.CurrentContext == name {
        c.CurrentContext = ""
    }
    if c.DefaultContext == name {
        c.DefaultContext = ""
    }
    if c.CurrentContext == "" && c.DefaultContext != "" {
        if _, ok := c.Contexts[c.DefaultContext]; ok {
            c.CurrentContext = c.DefaultContext
        }
    }
    if c.CurrentContext == "" {
        for k := range c.Contexts {
            c.CurrentContext = k
            break
        }
    }
    if c.DefaultContext == "" {
        for k := range c.Contexts {
            c.DefaultContext = k
            break
        }
    }
}

// SortedContexts returns contexts sorted by name for deterministic output.
func (c *Config) SortedContexts() []*Context {
    names := c.ContextNames()
    res := make([]*Context, 0, len(names))
    for _, name := range names {
        if ctx, ok := c.GetContext(name); ok {
            res = append(res, ctx)
        }
    }
    return res
}

// ContextNames returns sorted context names.
func (c *Config) ContextNames() []string {
    names := make([]string, 0, len(c.Contexts))
    for name := range c.Contexts {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}

func (c *Config) ensureNames() {
    if c.Contexts == nil {
        c.Contexts = make(map[string]*Context)
        return
    }
    for name, ctx := range c.Contexts {
        if ctx != nil {
            ctx.Name = name
        }
    }
}

// ValidateConnectionName ensures connection names can be safely stored on disk.
func ValidateConnectionName(name string) error {
    trimmed := strings.TrimSpace(name)
    if trimmed == "" {
        return fmt.Errorf("connection name cannot be empty")
    }
    if strings.ContainsAny(trimmed, "/\\") {
        return fmt.Errorf("connection name %q cannot contain path separators", name)
    }
    return nil
}

// ValidateContextName is kept for backwards compatibility.
func ValidateContextName(name string) error {
    return ValidateConnectionName(name)
}

func configDir() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("determine home directory: %w", err)
    }
    return filepath.Join(home, ".snowctl"), nil
}

func path() (string, error) {
    dir, err := configDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, "config.json"), nil
}

// Path exposes the absolute configuration path for reference in help output.
func Path() (string, error) {
    return path()
}
