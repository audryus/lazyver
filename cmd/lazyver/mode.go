package lazyver

import (
	"fmt"

	"codeberg.org/audryus/lazyver/internal/app"
	"codeberg.org/audryus/lazyver/internal/statefile"
	"github.com/spf13/cobra"
)

// pathFlag is shared by the semver and lazy commands and holds the target
// repository directory.
var pathFlag string

// printFlag controls whether the computed version is written to stdout.
var printFlag bool

// newModeCmd builds a cobra command for one of the versioning modes
// ("semver" or "lazy"), reusing the shared flags and run logic. Both modes
// initialize the repository when no .lazyver.yaml exists, update it
// afterwards and (re)install the commit-msg hook.
func newModeCmd(kind, alias, short string) *cobra.Command {
	return &cobra.Command{
		Use:     kind,
		Aliases: []string{alias},
		Short:   short,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			version, err := app.Run(kind, pathFlag)
			if err != nil {
				return err
			}
			if printFlag {
				fmt.Println(version)
			}
			return nil
		},
	}
}

// semverCmd bumps the version following conventional-commit message types.
var semverCmd *cobra.Command

// lazyCmd bumps the version based purely on the number of commits.
var lazyCmd *cobra.Command

func init() {
	semverCmd = newModeCmd(statefile.KindSemver, "s",
		"Version from conventional commit types (feat/fix/...)")
	semverCmd.Long = `Bump (or initialize) the version using conventional commit types.

Rules applied to each commit message not yet accounted for:
  major  "feat!:", "fix!:", any "<type>!:" or a footer "BREAKING CHANGE:"
  minor  feat, chore, build, docs, ci, test, style
  patch  fix, perf, revert, refactor
  none   anything else (e.g. "merge", "wip") — no bump

On the first run the whole history is scanned once and .lazyver.yaml is
created. Later runs only read commits newer than the stored hash. The
commit-msg hook is (re)installed so future commits bump the version
automatically.

Examples:
  lazyver semver                     # update version in the current repo
  lazyver semver -o                  # also print the version, e.g. v1.2.3
  lazyver semver --path ../other     # operate on another repository`

	lazyCmd = newModeCmd(statefile.KindLazy, "l",
		"Version from the total commit count (lazy mode)")
	lazyCmd.Long = `Bump (or initialize) the version using the total commit count.

The flat commit count is split across the three version digits:
  hundreds -> major, tens -> minor, units -> patch
    5 commits -> v0.0.5     42 commits -> v0.4.2    339 commits -> v3.3.9

On the first run the whole history is counted once and .lazyver.yaml is
created. Later runs only count new commits. The commit-msg hook is
(re)installed so future commits bump the version automatically.

Examples:
  lazyver lazy                       # update version in the current repo
  lazyver lazy -o                    # also print the version, e.g. v0.4.2
  lazyver lazy --path ../other       # operate on another repository`

	for _, c := range []*cobra.Command{semverCmd, lazyCmd} {
		c.Flags().StringVar(&pathFlag, "path", "./", "Path to the repository (directory containing .git)")
		c.Flags().BoolVarP(&printFlag, "output", "o", false, "Print the resulting version to stdout")
	}

	rootCmd.AddCommand(semverCmd)
	rootCmd.AddCommand(lazyCmd)
	rootCmd.Version = versionString
}
