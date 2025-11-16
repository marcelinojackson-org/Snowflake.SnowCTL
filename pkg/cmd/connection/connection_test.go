package connectioncmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

func prepareRuntime(t *testing.T, configure func(*config.Config)) *runtime.Runtime {
	t.Helper()
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	t.Cleanup(func() { os.Unsetenv("HOME") })

	cfg := config.DefaultConfig()
	if configure != nil {
		configure(cfg)
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	rt, err := runtime.NewRuntime("", "json")
	if err != nil {
		t.Fatalf("runtime: %v", err)
	}
	return rt
}

func newCmdWithRuntime(rt *runtime.Runtime) (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetContext(runtime.WithRuntime(context.Background(), rt))
	return cmd, buf
}

func TestRunUseConnectionOutputsJSON(t *testing.T) {
	rt := prepareRuntime(t, func(cfg *config.Config) {
		cfg.SetContext("primary", &config.Context{Account: "acct", Role: "role", AuthMethod: "password"})
	})
	cmd, buf := newCmdWithRuntime(rt)

	if err := runUseConnection(cmd, "primary"); err != nil {
		t.Fatalf("runUseConnection: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["connection"] != "primary" {
		t.Fatalf("expected connection primary, got %v", payload["connection"])
	}

	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if reloaded.CurrentContext != "primary" {
		t.Fatalf("config current not updated")
	}
}

func TestRunSetDefaultConnection(t *testing.T) {
	rt := prepareRuntime(t, func(cfg *config.Config) {
		cfg.SetContext("one", &config.Context{Account: "acct1", AuthMethod: "password"})
		cfg.SetContext("two", &config.Context{Account: "acct2", AuthMethod: "password"})
		cfg.CurrentContext = "one"
		cfg.DefaultContext = "one"
	})
	cmd, buf := newCmdWithRuntime(rt)

	if err := runSetDefaultConnection(cmd, "two"); err != nil {
		t.Fatalf("runSetDefaultConnection: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["default"] != "two" {
		t.Fatalf("expected default two, got %v", payload["default"])
	}

	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.DefaultContext != "two" {
		t.Fatalf("default context not updated")
	}
}

func TestRunListConnectionsJSON(t *testing.T) {
	rt := prepareRuntime(t, func(cfg *config.Config) {
		cfg.SetContext("one", &config.Context{Account: "acct1", AuthMethod: "password"})
		cfg.SetContext("two", &config.Context{Account: "acct2", AuthMethod: "password"})
		cfg.CurrentContext = "one"
		cfg.DefaultContext = "two"
	})
	cmd, buf := newCmdWithRuntime(rt)

	if err := runListConnections(cmd); err != nil {
		t.Fatalf("runListConnections: %v", err)
	}

	var payload []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(payload))
	}
	foundCurrent := false
	foundDefault := false
	for _, row := range payload {
		if row["name"] == "one" && row["isCurrent"].(bool) {
			foundCurrent = true
		}
		if row["name"] == "two" && row["isDefault"].(bool) {
			foundDefault = true
		}
	}
	if !foundCurrent || !foundDefault {
		t.Fatalf("current/default flags missing: %v", payload)
	}
}

func TestConnectionTestOutputsJSON(t *testing.T) {
	rt := prepareRuntime(t, func(cfg *config.Config) {
		cfg.SetContext("one", &config.Context{Account: "acct", AuthMethod: "password"})
		cfg.CurrentContext = "one"
	})

	orig := testConnectionFn
	testConnectionFn = func(ctx context.Context, info *config.Context, secret string) (string, error) {
		if secret != "secret" {
			t.Fatalf("expected secret credential, got %s", secret)
		}
		return "2025-01-01T00:00:00Z", nil
	}
	defer func() { testConnectionFn = orig }()

	os.Setenv("SNOWFLAKE_PASSWORD", "secret")
	t.Cleanup(func() { os.Unsetenv("SNOWFLAKE_PASSWORD") })

	cmd, buf := newCmdWithRuntime(rt)
	opts := &testOptions{setCurrent: true}
	if err := opts.run(cmd, []string{"one"}); err != nil {
		t.Fatalf("test run: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["currentSet"] != true {
		t.Fatalf("expected currentSet true, got %v", payload["currentSet"])
	}
}
