package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCompletionCmd(root *cobra.Command) *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `To load completions:

Bash:
  $ source <(snowctl completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ snowctl completion bash > /etc/bash_completion.d/snowctl
  # macOS:
  $ snowctl completion bash > /usr/local/etc/bash_completion.d/snowctl

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  $ snowctl completion zsh > "${fpath[1]}/_snowctl"

Fish:
  $ snowctl completion fish | source
  $ snowctl completion fish > ~/.config/fish/completions/snowctl.fish

PowerShell:
  PS> snowctl completion powershell | Out-String | Invoke-Expression
  PS> snowctl completion powershell > snowctl.ps1
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return root.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return root.GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}

	return completionCmd
}
