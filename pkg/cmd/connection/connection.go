package connectioncmd

import "github.com/spf13/cobra"

// NewConnectionCmd manages stored Snowflake connections.
func NewConnectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connection",
		Short: "Manage Snowflake connections",
		Long:  "Create, inspect, and activate Snowflake connections saved on this workstation.",
	}

	cmd.AddCommand(
		newSetConnectionCmd(),
		newListConnectionsCmd(),
		newUseConnectionCmd(),
		newRemoveConnectionCmd(),
		newSetDefaultConnectionCmd(),
		newTestConnectionCmd(),
	)

	return cmd
}
