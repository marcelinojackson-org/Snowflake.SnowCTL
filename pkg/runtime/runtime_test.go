package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
)

func setupConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	t.Cleanup(func() { os.Unsetenv("HOME") })

	cfg := config.DefaultConfig()
	cfg.SetContext("one", &config.Context{Account: "acct", User: "user", AuthMethod: "password"})
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	return filepath.Join(dir, ".snowctl", "config")
}

func TestNewRuntimeDefaultsToJson(t *testing.T) {
	setupConfig(t)
	rt, err := NewRuntime("", "")
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if rt.OutputFormat != "json" {
		t.Fatalf("expected json output, got %s", rt.OutputFormat)
	}
}

func TestNewRuntimeInvalidOutput(t *testing.T) {
	setupConfig(t)
	if _, err := NewRuntime("", "invalid"); err == nil {
		t.Fatalf("expected error for invalid output format")
	}
}

func TestRequireActiveContext(t *testing.T) {
	setupConfig(t)
	rt, err := NewRuntime("", "json")
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	ctx, err := RequireActiveContext(WithRuntime(context.Background(), rt))
	if err != nil {
		t.Fatalf("RequireActiveContext: %v", err)
	}
	if ctx.Name != "one" {
		t.Fatalf("expected context 'one', got %s", ctx.Name)
	}
}
