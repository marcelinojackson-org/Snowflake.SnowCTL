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
		meta, primary := splitMetadata(data)
		records, err := normalizeRecords(primary)
		if err != nil {
			return err
		}
		if err := writeMetadata(out, meta); err != nil {
			return err
		}
		return writeSeparated(out, records, ',')
	case "tsv":
		meta, primary := splitMetadata(data)
		records, err := normalizeRecords(primary)
		if err != nil {
			return err
		}
		if err := writeMetadata(out, meta); err != nil {
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
	var anyData interface{}
	if err := json.Unmarshal(raw, &anyData); err != nil {
		return nil, err
	}
	return flattenAny(anyData)
}

func splitMetadata(data interface{}) (map[string]any, interface{}) {
	switch v := data.(type) {
	case map[string]any:
		return extractMetadata(v)
	default:
		return nil, data
	}
}

func extractMetadata(m map[string]any) (map[string]any, interface{}) {
	if rows, ok := m["rows"]; ok {
		meta := copyMap(m)
		delete(meta, "rows")
		if len(meta) == 0 {
			meta = nil
		}
		return meta, rows
	}
	return nil, m
}

func writeMetadata(out io.Writer, meta map[string]any) error {
	if meta == nil || len(meta) == 0 {
		return nil
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(meta); err != nil {
		return err
	}
	if _, err := out.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

func flattenAny(value interface{}) ([]map[string]any, error) {
	switch v := value.(type) {
	case []interface{}:
		res := make([]map[string]any, 0, len(v))
		for _, item := range v {
			m, err := toMap(item)
			if err != nil {
				return nil, err
			}
			res = append(res, m)
		}
		return res, nil
	case map[string]interface{}:
		if rows, ok := v["rows"].([]interface{}); ok {
			base := copyMap(v)
			delete(base, "rows")
			res := make([]map[string]any, 0, len(rows)+1)
			if len(base) > 0 {
				res = append(res, base)
			}
			for _, row := range rows {
				m, err := toMap(row)
				if err != nil {
					return nil, err
				}
				res = append(res, m)
			}
			return res, nil
		}
		m, err := toMap(v)
		if err != nil {
			return nil, err
		}
		return []map[string]any{m}, nil
	default:
		m, err := toMap(v)
		if err != nil {
			return nil, fmt.Errorf("data must be an object or array of objects for csv/tsv output")
		}
		return []map[string]any{m}, nil
	}
}

func toMap(value interface{}) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	switch v := value.(type) {
	case map[string]interface{}:
		return v, nil
	default:
		return map[string]any{"value": v}, nil
	}
}

func copyMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
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
