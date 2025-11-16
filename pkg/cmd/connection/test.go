package connectioncmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

func newTestConnectionCmd() *cobra.Command {
	opts := &testOptions{}

	cmd := &cobra.Command{
		Use:   "test [NAME]",
		Short: "Test connectivity for a stored connection",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run(cmd, args)
		},
	}

	cmd.Flags().BoolVar(&opts.setCurrent, "set-current", false, "Set this connection as current after a successful test")
	return cmd
}

type testOptions struct {
	setCurrent bool
}

func (o *testOptions) run(cmd *cobra.Command, args []string) error {
	rt, err := runtime.RequireRuntime(cmd.Context())
	if err != nil {
		return err
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	var connection *config.Context

	if name == "" {
		contexts := rt.Config.SortedContexts()
		if len(contexts) == 0 {
			return fmt.Errorf("no connections configured. Use 'snowctl connection set' first")
		}
		if len(contexts) == 1 {
			connection = contexts[0]
			name = connection.Name
		} else {
			selected, err := promptConnectionSelection(cmd, contexts)
			if err != nil {
				return err
			}
			connection = selected
			name = connection.Name
		}
	} else {
		ctx, ok := rt.Config.GetContext(name)
		if !ok {
			return fmt.Errorf("connection %q not found", name)
		}
		connection = ctx
	}

	envVar := secretEnvVar(connection.AuthMethod)
	credential := strings.TrimSpace(os.Getenv(envVar))
	if credential == "" {
		return fmt.Errorf("%s is not set; export it before testing connection %q", envVar, name)
	}

	ts, err := testConnectionFn(cmd.Context(), connection, credential)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	result := map[string]any{
		"connection": name,
		"account":    connection.Account,
		"user":       connection.User,
		"authEnv":    envVar,
		"serverTime": ts,
		"currentSet": false,
	}

	if o.setCurrent {
		rt.Config.CurrentContext = name
		if err := config.Save(rt.Config); err != nil {
			return fmt.Errorf("failed to update current connection: %w", err)
		}
		result["currentSet"] = true
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func promptConnectionSelection(cmd *cobra.Command, contexts []*config.Context) (*config.Context, error) {
	fmt.Fprintln(cmd.OutOrStdout(), "Select a connection:")
	for i, ctx := range contexts {
		fmt.Fprintf(cmd.OutOrStdout(), "  %d) %s\n", i+1, ctx.Name)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Enter number or name [%s]: ", contexts[0].Name)

	reader := bufio.NewReader(cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	choice := strings.TrimSpace(input)
	if choice == "" {
		return contexts[0], nil
	}

	if idx, err := strconv.Atoi(choice); err == nil {
		if idx >= 1 && idx <= len(contexts) {
			return contexts[idx-1], nil
		}
		return nil, fmt.Errorf("selection %d out of range", idx)
	}

	for _, ctx := range contexts {
		if strings.EqualFold(ctx.Name, choice) {
			return ctx, nil
		}
	}

	return nil, fmt.Errorf("connection %q not found", choice)
}
