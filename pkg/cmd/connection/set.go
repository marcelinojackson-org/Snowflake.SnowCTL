package connectioncmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/output"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

const (
	authMethodPassword = "password"
	authMethodPAT      = "pat"
)

func newSetConnectionCmd() *cobra.Command {
	opts := &setConnectionOptions{}

	cmd := &cobra.Command{
		Use:   "set [NAME]",
		Short: "Create or update a Snowflake connection",
		Args:  cobra.MaximumNArgs(1),
		Long: `Interactively collect Snowflake connection details and persist them under ~/.snowctl/connections.
Secrets such as passwords or PATs are stored with each connection, so you don't have to manage environment variables per connection.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run(cmd, args)
		},
	}

	cmd.Flags().StringVar(&opts.account, "account", "", "Snowflake account locator (e.g. xy12345.us-east-1)")
	cmd.Flags().StringVar(&opts.accountURL, "account-url", "", "Snowflake account URL")
	cmd.Flags().StringVar(&opts.user, "user", "", "Snowflake username")
	cmd.Flags().StringVar(&opts.role, "role", "", "Default role to assume")
	cmd.Flags().StringVar(&opts.warehouse, "warehouse", "", "Default warehouse")
	cmd.Flags().StringVar(&opts.database, "database", "", "Default database")
	cmd.Flags().StringVar(&opts.schema, "schema", "", "Default schema")
	cmd.Flags().StringVar(&opts.description, "description", "", "Optional context description")
	cmd.Flags().StringVar(&opts.authMethod, "auth-method", "", "Authentication method to use (password or pat)")
	cmd.Flags().BoolVar(&opts.makeCurrent, "make-current", false, "Switch to this connection after saving")
	cmd.Flags().BoolVar(&opts.noPrompt, "no-prompt", false, "Disable interactive prompts; requires all flags to be set")
	cmd.Flags().StringVar(&opts.secret, "secret", "", "Secret credential (password or PAT) to store with the connection")

	return cmd
}

type setConnectionOptions struct {
	account     string
	accountURL  string
	user        string
	role        string
	warehouse   string
	database    string
	schema      string
	description string
	authMethod  string
	makeCurrent bool
	noPrompt    bool
	secret      string
}

func (o *setConnectionOptions) run(cmd *cobra.Command, args []string) error {
	input := cmd.InOrStdin()
	reader := bufio.NewReader(input)
	interactive := !o.noPrompt && isInteractive(input)
	if !interactive && !o.noPrompt {
		fmt.Fprintln(cmd.OutOrStdout(), "Input is not a TTY. Falling back to --no-prompt mode; please supply all values via flags.")
		o.noPrompt = true
		interactive = false
	}

	var providedName string
	if len(args) > 0 {
		providedName = args[0]
	}

	rt, err := runtime.RequireRuntime(cmd.Context())
	if err != nil {
		return err
	}

	ctx := &config.Context{}
	if providedName != "" {
		if existing, ok := rt.Config.GetContext(providedName); ok {
			*ctx = *existing
		}
	}

	envDefaults := map[string]string{
		"account":     os.Getenv("SNOWFLAKE_ACCOUNT"),
		"account-url": os.Getenv("SNOWFLAKE_ACCOUNT_URL"),
		"user":        os.Getenv("SNOWFLAKE_USER"),
		"role":        os.Getenv("SNOWFLAKE_ROLE"),
		"warehouse":   os.Getenv("SNOWFLAKE_WAREHOUSE"),
		"database":    os.Getenv("SNOWFLAKE_DATABASE"),
		"schema":      os.Getenv("SNOWFLAKE_SCHEMA"),
	}

	ctx.Account, err = o.valueOrPrompt(cmd, reader, "Account locator", ctx.Account, o.account, "account", envDefaults["account"], true, interactive)
	if err != nil {
		return err
	}
	ctx.AccountURL, err = o.valueOrPrompt(cmd, reader, "Account URL", ctx.AccountURL, o.accountURL, "account-url", envDefaults["account-url"], true, interactive)
	if err != nil {
		return err
	}
	ctx.User, err = o.valueOrPrompt(cmd, reader, "Username", ctx.User, o.user, "user", envDefaults["user"], true, interactive)
	if err != nil {
		return err
	}
	ctx.Role, err = o.valueOrPrompt(cmd, reader, "Default role", ctx.Role, o.role, "role", envDefaults["role"], true, interactive)
	if err != nil {
		return err
	}
	ctx.Warehouse, err = o.valueOrPrompt(cmd, reader, "Default warehouse", ctx.Warehouse, o.warehouse, "warehouse", envDefaults["warehouse"], true, interactive)
	if err != nil {
		return err
	}
	ctx.Database, err = o.valueOrPrompt(cmd, reader, "Default database", ctx.Database, o.database, "database", envDefaults["database"], true, interactive)
	if err != nil {
		return err
	}
	ctx.Schema, err = o.valueOrPrompt(cmd, reader, "Default schema", ctx.Schema, o.schema, "schema", envDefaults["schema"], true, interactive)
	if err != nil {
		return err
	}
	ctx.Description, err = o.valueOrPrompt(cmd, reader, "Description", ctx.Description, o.description, "description", "", false, interactive)
	if err != nil {
		return err
	}

	ctx.AuthMethod, err = o.resolveAuthMethod(cmd, reader, ctx.AuthMethod, interactive)
	if err != nil {
		return err
	}
	secret, err := o.resolveAuthSecret(cmd, ctx.AuthMethod, ctx.Secret, interactive)
	if err != nil {
		return err
	}
	ctx.Secret = secret

	name := providedName
	if name == "" {
		if !interactive {
			return fmt.Errorf("connection name required; pass NAME or run interactively")
		}
		val, err := promptString(cmd, reader, "Connection name", "", true)
		if err != nil {
			return err
		}
		name = val
	}

	if err := config.ValidateConnectionName(name); err != nil {
		return err
	}

	ctx.Name = name
	ts, err := testConnectionFn(cmd.Context(), ctx)
	if err != nil {
		return fmt.Errorf("connection validation failed: %w", err)
	}

	rt.Config.SetContext(name, ctx)
	if o.makeCurrent {
		rt.Config.CurrentContext = name
	}

	if err := config.Save(rt.Config); err != nil {
		return err
	}

	resp := map[string]any{
		"connection": name,
		"savedAt":    connectionsLocation(),
		"serverTime": ts,
		"activated":  o.makeCurrent,
	}
	return output.Print(cmd, resp)
}

func (o *setConnectionOptions) valueOrPrompt(cmd *cobra.Command, reader *bufio.Reader, label, current, flagValue, flagName, envValue string, required bool, interactive bool) (string, error) {
	value := strings.TrimSpace(current)
	if cmd.Flags().Changed(flagName) {
		value = strings.TrimSpace(flagValue)
	}
	if value == "" && envValue != "" {
		value = strings.TrimSpace(envValue)
	}
	if interactive {
		return promptString(cmd, reader, label, value, required)
	}
	if required && value == "" {
		return "", fmt.Errorf("%s is required; pass --%s or run interactively", label, flagName)
	}
	return value, nil
}

func (o *setConnectionOptions) resolveAuthMethod(cmd *cobra.Command, reader *bufio.Reader, current string, interactive bool) (string, error) {
	value := strings.TrimSpace(strings.ToLower(current))
	if cmd.Flags().Changed("auth-method") {
		value = strings.TrimSpace(strings.ToLower(o.authMethod))
	}
	if value == "" {
		value = authMethodPassword
	}

	if interactive {
		for {
			answer, err := promptString(cmd, reader, "Authentication method (password|pat)", value, true)
			if err != nil {
				return "", err
			}
			answer = strings.ToLower(strings.TrimSpace(answer))
			if answer == authMethodPassword || answer == authMethodPAT {
				value = answer
				break
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Invalid authentication method. Enter 'password' or 'pat'.")
		}
	}

	switch value {
	case authMethodPassword, authMethodPAT:
		return value, nil
	default:
		return "", fmt.Errorf("invalid auth method %q: must be password or pat", value)
	}
}

func (o *setConnectionOptions) resolveAuthSecret(cmd *cobra.Command, method, current string, interactive bool) (string, error) {
	if cmd.Flags().Changed("secret") {
		value := strings.TrimSpace(o.secret)
		if value == "" {
			return "", fmt.Errorf("--secret cannot be empty")
		}
		return value, nil
	}

	trimmedCurrent := strings.TrimSpace(current)
	if interactive {
		label := "Password"
		if method == authMethodPAT {
			label = "Personal access token"
		}
		allowEmpty := trimmedCurrent != ""
		value, err := promptSecret(cmd, label, allowEmpty)
		if err != nil {
			return "", err
		}
		if value == "" && allowEmpty {
			return trimmedCurrent, nil
		}
		if value == "" {
			return "", fmt.Errorf("%s is required", label)
		}
		return value, nil
	}

	if trimmedCurrent != "" {
		return trimmedCurrent, nil
	}

	return "", fmt.Errorf("secret is required; provide --secret when running non-interactively")
}

func promptString(cmd *cobra.Command, reader *bufio.Reader, label, defaultValue string, required bool) (string, error) {
	for {
		prompt := label
		if defaultValue != "" {
			prompt = fmt.Sprintf("%s [%s]", label, defaultValue)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s: ", prompt)
		text, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		if err != nil && errors.Is(err, io.EOF) && text == "" {
			text = defaultValue
		}
		value := strings.TrimSpace(text)
		if value == "" {
			if defaultValue != "" {
				return strings.TrimSpace(defaultValue), nil
			}
			if required {
				fmt.Fprintln(cmd.OutOrStdout(), "This field is required.")
				continue
			}
			return "", nil
		}
		return value, nil
	}
}

func promptSecret(cmd *cobra.Command, label string, allowEmpty bool) (string, error) {
	input, ok := cmd.InOrStdin().(*os.File)
	if !ok {
		return "", fmt.Errorf("cannot prompt for %s without a terminal", strings.ToLower(label))
	}
	for {
		fmt.Fprintf(cmd.OutOrStdout(), "%s: ", label)
		data, err := term.ReadPassword(int(input.Fd()))
		fmt.Fprintln(cmd.OutOrStdout())
		if err != nil {
			return "", err
		}
		value := strings.TrimSpace(string(data))
		if value == "" && !allowEmpty {
			fmt.Fprintln(cmd.OutOrStdout(), "This field is required.")
			continue
		}
		return value, nil
	}
}

func isInteractive(r io.Reader) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func connectionsLocation() string {
	path, err := config.Path()
	if err != nil {
		return "~/.snowctl/config.yaml"
	}
	return path
}
