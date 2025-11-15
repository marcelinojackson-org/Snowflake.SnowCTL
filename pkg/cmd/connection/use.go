package connectioncmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

func newUseConnectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "use NAME",
		Aliases: []string{"activate"},
		Short:   "Switch the current connection",
		Args:    cobra.ExactArgs(1),
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
	fmt.Fprintf(cmd.OutOrStdout(), "Now using connection %q (account=%s, role=%s).\n", name, ctx.Account, ctx.Role)
	return nil
}
