package connectioncmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/output"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

func newRemoveConnectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove NAME",
		Aliases: []string{"rm", "delete"},
		Short:   "Delete a saved connection",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemoveConnection(cmd, args[0])
		},
	}
	return cmd
}

func runRemoveConnection(cmd *cobra.Command, name string) error {
	rt, err := runtime.RequireRuntime(cmd.Context())
	if err != nil {
		return err
	}
	if _, ok := rt.Config.GetContext(name); !ok {
		return fmt.Errorf("connection %q not found", name)
	}
	rt.Config.DeleteContext(name)
	if err := config.Save(rt.Config); err != nil {
		return err
	}
	return output.Print(cmd, map[string]string{
		"connection": name,
		"status":     "deleted",
	})
}
