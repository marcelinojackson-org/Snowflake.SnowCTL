package connectioncmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

func newSetDefaultConnectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-default NAME",
		Short: "Set the default connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetDefaultConnection(cmd, args[0])
		},
	}
	return cmd
}

func runSetDefaultConnection(cmd *cobra.Command, name string) error {
	rt, err := runtime.RequireRuntime(cmd.Context())
	if err != nil {
		return err
	}
	if _, ok := rt.Config.GetContext(name); !ok {
		return fmt.Errorf("connection %q not found", name)
	}
	rt.Config.DefaultContext = name
	if rt.Config.CurrentContext == "" {
		rt.Config.CurrentContext = name
	}
	if err := config.Save(rt.Config); err != nil {
		return err
	}

	payload := map[string]any{
		"default": name,
	}
	if rt.Config.CurrentContext == name {
		payload["current"] = name
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
