package sqlcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/snowflake"
)

func NewSQLCmd() *cobra.Command {
	opts := &sqlOptions{}
	cmd := &cobra.Command{
		Use:   "sql",
		Short: "Execute SQL against the active connection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run(cmd)
		},
	}
	cmd.Flags().StringVarP(&opts.statement, "query", "q", "", "SQL query to execute")
	cmd.Flags().StringVar(&opts.statement, "text", "", "SQL query to execute (alias of --query)")
	return cmd
}

type sqlOptions struct {
	statement string
}

func (o *sqlOptions) run(cmd *cobra.Command) error {
	stmt := strings.TrimSpace(o.statement)
	if stmt == "" {
		return fmt.Errorf("query is required. Use --query \"SELECT ...\"")
	}

	ctx, err := runtime.RequireActiveContext(cmd.Context())
	if err != nil {
		return err
	}

	envVar := secretEnvVar(ctx.AuthMethod)
	credential := strings.TrimSpace(os.Getenv(envVar))
	if credential == "" {
		return fmt.Errorf("%s is not set; export it before running SQL", envVar)
	}

	rows, err := snowflake.RunQuery(cmd.Context(), ctx, credential, stmt)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	payload := map[string]any{
		"connection": ctx.Name,
		"statement":  stmt,
		"rows":       rows,
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func secretEnvVar(method string) string {
	if method == "pat" {
		return "SNOWFLAKE_PAT"
	}
	return "SNOWFLAKE_PASSWORD"
}
