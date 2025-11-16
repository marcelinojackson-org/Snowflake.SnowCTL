package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

// Print renders data according to the runtime output format.
func Print(cmd *cobra.Command, data interface{}) error {
	rt, err := runtime.RequireRuntime(cmd.Context())
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()

	switch rt.OutputFormat {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case "yaml":
		buf, err := yaml.Marshal(data)
		if err != nil {
			return err
		}
		if _, err := out.Write(buf); err != nil {
			return err
		}
		_, err = out.Write([]byte("\n"))
		return err
	case "csv":
		records, err := normalizeRecords(data)
		if err != nil {
			return err
		}
		return writeSeparated(out, records, ',')
	case "tsv":
		records, err := normalizeRecords(data)
		if err != nil {
			return err
		}
		return writeSeparated(out, records, '\t')
	default:
		return fmt.Errorf("unsupported output format %q", rt.OutputFormat)
	}
}

func normalizeRecords(data interface{}) ([]map[string]any, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var slice []map[string]any
	if err := json.Unmarshal(raw, &slice); err == nil {
		return slice, nil
	}
	var single map[string]any
	if err := json.Unmarshal(raw, &single); err == nil {
		return []map[string]any{single}, nil
	}
	var rowsWrapper struct {
		Rows []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal(raw, &rowsWrapper); err == nil && len(rowsWrapper.Rows) > 0 {
		return rowsWrapper.Rows, nil
	}
	return nil, fmt.Errorf("data must be an object or array of objects for csv/tsv output")
}

func writeSeparated(w io.Writer, records []map[string]any, sep rune) error {
	if len(records) == 0 {
		return nil
	}
	headers := collectHeaders(records)
	writer := csv.NewWriter(w)
	writer.Comma = sep
	if err := writer.Write(headers); err != nil {
		return err
	}
	row := make([]string, len(headers))
	for _, rec := range records {
		for i, h := range headers {
			if val, ok := rec[h]; ok && val != nil {
				row[i] = fmt.Sprintf("%v", val)
			} else {
				row[i] = ""
			}
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func collectHeaders(records []map[string]any) []string {
	set := map[string]struct{}{}
	for _, rec := range records {
		for k := range rec {
			set[k] = struct{}{}
		}
	}
	headers := make([]string, 0, len(set))
	for k := range set {
		headers = append(headers, k)
	}
	sort.Strings(headers)
	return headers
}
