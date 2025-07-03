package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for pb.

To load completions:

Bash:
  $ source <(pb completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ pb completion bash > /etc/bash_completion.d/pb
  # macOS:
  $ pb completion bash > $(brew --prefix)/etc/bash_completion.d/pb

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ pb completion zsh > "${fpath[1]}/_pb"

  # You might need to start a new shell for this setup to take effect.

Fish:
  $ pb completion fish | source

  # To load completions for each session, execute once:
  $ pb completion fish > ~/.config/fish/completions/pb.fish

PowerShell:
  PS> pb completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> pb completion powershell > pb.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
