package showcmd

import "github.com/spf13/cobra"

// NewShowCmd groups informational commands such as `show account`.
func NewShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display information about the current Snowflake environment",
	}

	cmd.AddCommand(newAccountCmd())
	return cmd
}
