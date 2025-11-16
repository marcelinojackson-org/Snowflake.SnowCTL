package sqlcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

func prepareSQLRuntime(t *testing.T) *runtime.Runtime {
	t.Helper()
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	t.Cleanup(func() { os.Unsetenv("HOME") })

	cfg := config.DefaultConfig()
	cfg.SetContext("primary", &config.Context{Account: "acct", AuthMethod: "password"})
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	rt, err := runtime.NewRuntime("", "json")
	if err != nil {
		t.Fatalf("runtime: %v", err)
	}
	return rt
}

func TestSQLCommandOutputsJSON(t *testing.T) {
	rt := prepareSQLRuntime(t)

	orig := runQueryFn
	runQueryFn = func(ctx context.Context, info *config.Context, secret, stmt string) ([]map[string]any, error) {
		if stmt != "select 1" {
			t.Fatalf("expected statement select 1, got %s", stmt)
		}
		return []map[string]any{{"COL1": float64(1)}}, nil
	}
	defer func() { runQueryFn = orig }()

	os.Setenv("SNOWFLAKE_PASSWORD", "secret")
	t.Cleanup(func() { os.Unsetenv("SNOWFLAKE_PASSWORD") })

	cmd := NewSQLCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetContext(runtime.WithRuntime(context.Background(), rt))
	cmd.SetArgs([]string{"--query", "select 1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var payload struct {
		Connection string           `json:"connection"`
		Statement  string           `json:"statement"`
		Rows       []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Connection != "primary" || payload.Statement != "select 1" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if len(payload.Rows) != 1 {
		t.Fatalf("expected 1 row")
	}
}
