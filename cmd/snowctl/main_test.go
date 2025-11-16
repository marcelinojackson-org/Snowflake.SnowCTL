package main

import (
	"os"
	"testing"
)

func TestMainExecutesHelp(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	t.Cleanup(func() { os.Unsetenv("HOME") })

	oldArgs := os.Args
	os.Args = []string{"snowctl", "--help"}
	defer func() { os.Args = oldArgs }()

	main()
}
