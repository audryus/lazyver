package lazyver

import (
	"codeberg.org/audryus/lazyver/internal/app"
	"github.com/spf13/cobra"
)

// hookCmd is a hidden command invoked by the commit-msg git hook that
// lazyver installs. Users are not expected to call it manually.
var hookCmd = &cobra.Command{
	Use:    "hook <message-file>",
	Short:  "Internal: invoked by the installed commit-msg git hook",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.HandleHookMessage(pathFlag, args[0])
	},
}

func init() {
	hookCmd.Flags().StringVar(&pathFlag, "path", "./", "Path to the repository (directory containing .git)")
	rootCmd.AddCommand(hookCmd)
}
