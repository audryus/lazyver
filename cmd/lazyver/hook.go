package lazyver

import (
	"github.com/audryus/lazyver/internal/app"
	"github.com/spf13/cobra"
)

// hookCmd is a hidden command invoked by the installed commit-msg git hook.
// Users are not expected to call it manually.
var hookCmd = &cobra.Command{
	Use:    "hook <message-file>",
	Short:  "Internal: invoked by the installed commit-msg git hook",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.HandleHookMessage(pathFlag, args[0])
	},
}

// hookPostCmd is a hidden command invoked by the installed post-commit git
// hook. It amends HEAD so the bumped version file joins the commit.
var hookPostCmd = &cobra.Command{
	Use:    "hook-post",
	Short:  "Internal: invoked by the installed post-commit git hook",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.HandlePostCommit(pathFlag)
	},
}

func init() {
	for _, c := range []*cobra.Command{hookCmd, hookPostCmd} {
		c.Flags().StringVar(&pathFlag, "path", "./", "Path to the repository (directory containing .git)")
	}
	rootCmd.AddCommand(hookCmd, hookPostCmd)
}
