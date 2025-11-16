package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	t.Cleanup(func() { os.Unsetenv("HOME") })

	cfg := DefaultConfig()
	cfg.SetContext("test", &Context{Account: "acct", User: "user", AuthMethod: "password"})

	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.CurrentContext != "test" {
		t.Fatalf("expected current context 'test', got %s", loaded.CurrentContext)
	}
	if _, ok := loaded.Contexts["test"]; !ok {
		t.Fatalf("expected context 'test' present")
	}
}

func TestMigrationFromJSON(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	t.Cleanup(func() { os.Unsetenv("HOME") })

	legacy := map[string]any{
		"currentContext": "old",
		"defaultContext": "old",
		"contexts": map[string]map[string]string{
			"old": {"account": "acct", "user": "user"},
		},
	}
	legacyPath := filepath.Join(dir, ".snowctl", "config.json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, _ := json.Marshal(legacy)
	if err := os.WriteFile(legacyPath, data, 0o600); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.CurrentContext != "old" {
		t.Fatalf("expected migrated current context 'old'")
	}
	cfgPath, _ := path()
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected new config file, got err: %v", err)
	}
}
