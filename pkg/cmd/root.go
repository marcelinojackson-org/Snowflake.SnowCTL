package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/build"
	connectioncmd "github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/cmd/connection"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

var (
	connectionOverride string
	outputFormat       string
)

// NewRootCmd constructs the root snowctl command with global flags and subcommands.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "snowctl",
		Short:         "snowctl is a Snowflake utility CLI",
		Long:          `snowctl is a Snowflake utility for managing configuration, governance, and operational workflows.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			rt, err := runtime.NewRuntime(connectionOverride, outputFormat)
			if err != nil {
				return err
			}
			cmd.SetContext(runtime.WithRuntime(cmd.Context(), rt))
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	rootCmd.Version = build.Version
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.PersistentFlags().StringVarP(&connectionOverride, "connection", "c", "", "Snowflake connection to use (overrides the current connection)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "Output format. Supported: json")

	rootCmd.AddCommand(
		connectioncmd.NewConnectionCmd(),
	)

	rootCmd.AddCommand(newCompletionCmd(rootCmd))
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		if c == rootCmd {
			printRootHelp(c)
			return
		}
		defaultHelp(c, args)
	})

	return rootCmd
}

// Execute runs the root snowctl command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printRootHelp(cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Snowflake DevOps CLI (%s)\n\n", build.Version)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintf(out, "  %s <command> [flags]\n\n", cmd.CommandPath())

	fmt.Fprintln(out, "Commands:")
	var commands []string
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.Name() == "help" {
			continue
		}
		commands = append(commands, fmt.Sprintf("%-12s %s", c.Name(), c.Short))
	}
	sort.Strings(commands)
	for _, line := range commands {
		fmt.Fprintf(out, "  %s\n", line)
	}
	fmt.Fprintln(out, "")

	fmt.Fprintln(out, "Flags:")
	flags := []string{
		"-h, --help        Show help",
		"--version         Show version",
		"-c, --connection  Use a connection",
		"-o, --output      Output format",
	}
	for _, f := range flags {
		fmt.Fprintf(out, "  %s\n", f)
	}
}
