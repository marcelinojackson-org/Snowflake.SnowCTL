package output

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

func newOutputCmd(t *testing.T, format string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	rt := &runtime.Runtime{OutputFormat: format}
	cmd.SetContext(runtime.WithRuntime(context.Background(), rt))
	return cmd, buf
}

func TestPrintJSON(t *testing.T) {
	cmd, buf := newOutputCmd(t, "json")

	data := struct {
		Foo string `json:"foo"`
		Num int    `json:"num"`
	}{
		Foo: "bar",
		Num: 42,
	}

	if err := Print(cmd, data); err != nil {
		t.Fatalf("Print json: %v", err)
	}

	expected := "{\n  \"foo\": \"bar\",\n  \"num\": 42\n}\n"
	if buf.String() != expected {
		t.Fatalf("unexpected json output:\n%s", buf.String())
	}
}

func TestPrintYAML(t *testing.T) {
	cmd, buf := newOutputCmd(t, "yaml")

	data := struct {
		Foo string `yaml:"foo"`
		Num int    `yaml:"num"`
	}{
		Foo: "bar",
		Num: 42,
	}

	if err := Print(cmd, data); err != nil {
		t.Fatalf("Print yaml: %v", err)
	}

	expected := "foo: bar\nnum: 42\n\n"
	if buf.String() != expected {
		t.Fatalf("unexpected yaml output:\n%s", buf.String())
	}
}

func TestPrintCSVWithMetadata(t *testing.T) {
	cmd, buf := newOutputCmd(t, "csv")

	data := map[string]any{
		"connection": "prod",
		"count":      2,
		"rows": []map[string]any{
			{"name": "alpha", "value": 1},
			{"name": "beta"},
		},
	}

	if err := Print(cmd, data); err != nil {
		t.Fatalf("Print csv: %v", err)
	}

	output := buf.String()
	parts := strings.SplitN(output, "\n\n", 2)
	if len(parts) != 2 {
		t.Fatalf("expected metadata and csv sections, got %q", output)
	}

	meta := map[string]any{}
	if err := json.Unmarshal([]byte(parts[0]), &meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if meta["connection"] != "prod" || meta["count"].(float64) != 2 {
		t.Fatalf("unexpected metadata: %#v", meta)
	}

	reader := csv.NewReader(strings.NewReader(parts[1]))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	expected := [][]string{
		{"name", "value"},
		{"alpha", "1"},
		{"beta", ""},
	}
	if len(records) != len(expected) {
		t.Fatalf("expected %d rows, got %d", len(expected), len(records))
	}
	for i := range expected {
		if strings.Join(records[i], ",") != strings.Join(expected[i], ",") {
			t.Fatalf("row %d = %v, want %v", i, records[i], expected[i])
		}
	}
}

func TestPrintTSVWithoutMetadata(t *testing.T) {
	cmd, buf := newOutputCmd(t, "tsv")

	rows := []map[string]any{
		{"value": 1, "name": "alpha"},
	}

	if err := Print(cmd, rows); err != nil {
		t.Fatalf("Print tsv: %v", err)
	}

	expected := "name\tvalue\nalpha\t1\n"
	if buf.String() != expected {
		t.Fatalf("unexpected tsv output %q", buf.String())
	}
}

func TestPrintUnsupportedFormat(t *testing.T) {
	cmd, _ := newOutputCmd(t, "xml")

	err := Print(cmd, map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestPrintCSVWithMetadataProvider(t *testing.T) {
	cmd, buf := newOutputCmd(t, "csv")

	data := providerPayload{
		Statement: "select 1",
		Rows: []map[string]any{
			{"col": 1},
		},
	}

	if err := Print(cmd, data); err != nil {
		t.Fatalf("Print csv metadata provider: %v", err)
	}

	parts := strings.SplitN(buf.String(), "\n\n", 2)
	if len(parts) != 2 {
		t.Fatalf("expected metadata + rows, got %q", buf.String())
	}
	if !strings.Contains(parts[0], "select 1") {
		t.Fatalf("metadata missing statement: %s", parts[0])
	}
	if !strings.Contains(parts[1], "col") {
		t.Fatalf("rows missing content: %s", parts[1])
	}
}

type providerPayload struct {
	Statement string
	Rows      []map[string]any
}

func (p providerPayload) OutputMetadata() (interface{}, interface{}) {
	return providerMeta{Statement: p.Statement}, p.Rows
}

type providerMeta struct {
	Statement string `json:"statement"`
}
