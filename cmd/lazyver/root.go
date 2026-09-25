// Package lazyver implements the lazyver command-line interface. It exposes
// two versioning modes (semver and lazy) plus a hidden "hook" subcommand
// used exclusively by the installed commit-msg git hook.
package lazyver

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// versionString is the value shown by `lazyver --version`. It defaults to
// "dev" for plain source builds and is overridden at link time by goreleaser
// via `-X github.com/audryus/lazyver/cmd/lazyver.versionString=<tag>`.
var versionString = "dev"

// resolvedVersion returns the most accurate version string available, in
// order of preference:
//
//  1. the linker-injected versionString set by goreleaser at release time;
//  2. the module version embedded in the build info when the binary was
//     installed through `go install github.com/audryus/lazyver@<tag>`;
//  3. the fallback "dev", used for local source builds and tests.
func resolvedVersion() string {
	if versionString != "" && versionString != "dev" {
		return versionString
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		switch v := info.Main.Version; v {
		case "", "(devel)":
			// Not installed from a tagged module revision.
		default:
			return v
		}
	}
	return "dev"
}

// rootCmd is the entry point of the CLI. Running bare "lazyver" prints the
// full help so users can discover both modes.
var rootCmd = &cobra.Command{
	Use:   "lazyver",
	Short: "Automatic version generation for lazy people",
	Long: `lazyver — automatic version generation for lazy people.

lazyver derives your project's version from your git commit history, so you
never have to think about versions again: just commit and the version file
(.lazyver.yaml) is bumped, written and staged automatically before the
commit is finalized (via a commit-msg hook that lazyver installs for you).

Two versioning modes are available (pick ONE per repository):

  lazyver semver   Conventional-commit driven bumps:
                     feat:, chore:, build:, docs:, ci:, test:, style:  -> minor
                     fix:, perf:, revert:, refactor:                   -> patch
                     feat!:/fix!:/... or "BREAKING CHANGE:" footer     -> major

  lazyver lazy     Pure commit-count driven: the total number of commits
                   is spread over the version digits.
                     5 commits -> v0.0.5    42 commits -> v0.4.2
                   339 commits -> v3.3.9

How it works:
  1. First run in a repository reads the whole history once and writes
     .lazyver.yaml with the computed version and the last commit hash.
  2. Every later run (and every commit, through the hook) only inspects the
     commits newer than that hash — fast, even on huge repositories.
  3. The version file is committed together with your change, so the
     version travels with the code.

Examples:
  # Initialize/update a repository using conventional commits and print v1.2.3
  lazyver semver --path ./my-project -o

  # Same, but using the lazy commit-count mode
  lazyver lazy -o

  # Show the full help of a mode
  lazyver semver --help

Note: a repository cannot mix modes. Delete .lazyver.yaml to switch.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// Execute runs the root command and terminates the process with a non-zero
// status when the CLI fails, printing the error to stderr.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "lazyver: %v\n", err)
		os.Exit(1)
	}
}
