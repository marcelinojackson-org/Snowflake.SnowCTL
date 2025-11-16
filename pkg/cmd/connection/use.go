package connectioncmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/output"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

func newUseConnectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "use NAME",
		Aliases: []string{"activate"},
		Short:   "Switch the current connection",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("accepts 1 argument (connection name). Example: 'snowctl connection use HRAnalystics_Connection'")
			}
			return nil
		},
		Example: `# Switch to a specific connection by name
snowctl connection use HRAnalystics_Connection`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUseConnection(cmd, args[0])
		},
	}
	return cmd
}

func runUseConnection(cmd *cobra.Command, name string) error {
	rt, err := runtime.RequireRuntime(cmd.Context())
	if err != nil {
		return err
	}
	ctx, ok := rt.Config.GetContext(name)
	if !ok {
		return fmt.Errorf("connection %q not found", name)
	}
	rt.Config.CurrentContext = name
	if err := config.Save(rt.Config); err != nil {
		return err
	}
	return output.Print(cmd, map[string]string{
		"connection": name,
		"account":    ctx.Account,
		"role":       ctx.Role,
	})
}

func init() {}
