package versioncmd

import (
    "bytes"
    "testing"

    "github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/build"
)

func TestVersionPrintDefault(t *testing.T) {
    out := &bytes.Buffer{}
    if err := Print(out, ""); err != nil {
        t.Fatalf("Print: %v", err)
    }
    if got := out.String(); got != build.Version+"\n" {
        t.Fatalf("unexpected output %q", got)
    }
}

func TestVersionPrintJSON(t *testing.T) {
    out := &bytes.Buffer{}
    if err := Print(out, "json"); err != nil {
        t.Fatalf("Print json: %v", err)
    }
    if !bytes.Contains(out.Bytes(), []byte("version")) {
        t.Fatalf("expected json output, got %s", out.String())
    }
}
