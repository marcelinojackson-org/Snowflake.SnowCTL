package connectioncmd

import (
	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/output"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

func newListConnectionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured connections",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListConnections(cmd)
		},
	}
	return cmd
}

type connectionView struct {
	Name        string `json:"name"`
	IsCurrent   bool   `json:"isCurrent"`
	IsDefault   bool   `json:"isDefault"`
	Account     string `json:"account,omitempty"`
	AccountURL  string `json:"accountUrl,omitempty"`
	User        string `json:"user,omitempty"`
	Role        string `json:"role,omitempty"`
	Warehouse   string `json:"warehouse,omitempty"`
	Database    string `json:"database,omitempty"`
	Schema      string `json:"schema,omitempty"`
	Description string `json:"description,omitempty"`
	AuthMethod  string `json:"authMethod,omitempty"`
}

func runListConnections(cmd *cobra.Command) error {
	rt, err := runtime.RequireRuntime(cmd.Context())
	if err != nil {
		return err
	}

	contexts := rt.Config.SortedContexts()
	views := make([]connectionView, 0, len(contexts))
	for _, ctx := range contexts {
		if ctx == nil {
			continue
		}
		views = append(views, connectionView{
			Name:        ctx.Name,
			IsCurrent:   ctx.Name == rt.Config.CurrentContext,
			IsDefault:   ctx.Name == rt.Config.DefaultContext,
			Account:     ctx.Account,
			AccountURL:  ctx.AccountURL,
			User:        ctx.User,
			Role:        ctx.Role,
			Warehouse:   ctx.Warehouse,
			Database:    ctx.Database,
			Schema:      ctx.Schema,
			Description: ctx.Description,
			AuthMethod:  ctx.AuthMethod,
		})
	}

	return output.Print(cmd, views)
}
