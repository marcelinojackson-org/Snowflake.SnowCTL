package cmd

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

func prepareRootRuntime(t *testing.T) *runtime.Runtime {
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

func TestPrintRootHelpIncludesSections(t *testing.T) {
	rt := prepareRootRuntime(t)
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetContext(runtime.WithRuntime(context.Background(), rt))

	printRootHelp(cmd)
	output := buf.String()
	if !strings.Contains(output, "Usage:") {
		t.Fatalf("expected Usage section, got %s", output)
	}
	if !strings.Contains(output, "Commands:") {
		t.Fatalf("expected Commands section")
	}
}
