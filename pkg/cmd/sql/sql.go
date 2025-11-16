package sqlcmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/output"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
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

	if strings.TrimSpace(ctx.Secret) == "" {
		return fmt.Errorf("connection %q has no stored credential. Re-run 'snowctl connection set %s' to store one.", ctx.Name, ctx.Name)
	}

	rows, err := runQueryFn(cmd.Context(), ctx, stmt)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	resp := queryResponse{
		Statement:  stmt,
		Connection: ctx.Name,
		Rows:       rows,
	}
	return output.Print(cmd, resp)
}

type queryResponse struct {
	Connection string           `json:"connection" yaml:"connection"`
	Statement  string           `json:"statement" yaml:"statement"`
	Rows       []map[string]any `json:"rows" yaml:"rows"`
}

func (r queryResponse) OutputMetadata() (interface{}, interface{}) {
	meta := responseMetadata{
		Connection: r.Connection,
		Statement:  r.Statement,
	}
	return meta, r.Rows
}

type responseMetadata struct {
	Connection string `json:"connection" yaml:"connection"`
	Statement  string `json:"statement" yaml:"statement"`
}
