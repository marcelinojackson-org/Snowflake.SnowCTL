package versioncmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/build"
)

// NewVersionCmd prints CLI build metadata similar to docker/kubectl version.
func NewVersionCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print snowctl version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Print(cmd.OutOrStdout(), output)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output format: short|json")
	return cmd
}

// Print renders version info to the provided writer.
func Print(out io.Writer, output string) error {
	switch output {
	case "json":
		payload := map[string]string{
			"version": build.Version,
			"commit":  build.Commit,
			"date":    build.Date,
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	case "short":
		_, err := fmt.Fprintf(out, "%s\n", build.Version)
		return err
	default:
		_, err := fmt.Fprintf(out, "%s\n", build.Version)
		return err
	}
}
