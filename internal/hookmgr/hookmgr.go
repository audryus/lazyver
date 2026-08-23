// Package hookmgr installs and manages the lazyver git hooks inside a
// repository. Two hooks work together to make versioning seamless:
//
//   - commit-msg: runs after the message is written but before the commit is
//     created; classifies it and bumps the version (marking the state as
//     pending). At this point git has already built the commit tree, so a
//     file added here would arrive too late.
//   - post-commit: if a bump happened, amends HEAD so the version file is
//     folded into the very commit that triggered it.
package hookmgr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Marker identifies hooks written by lazyver. Any existing hook file that
// does not contain this marker is considered foreign and is never replaced.
const Marker = "# installed by lazyver"

const (
	// CommitMsgHook classifies the message and bumps the version.
	CommitMsgHook = "commit-msg"
	// PostCommitHook folds the bumped version file into the commit via amend.
	PostCommitHook = "post-commit"
)

// hooks lists every git hook managed by lazyver and the lazyver subcommand
// each one delegates to.
var hooks = map[string]string{
	CommitMsgHook:  "hook",
	PostCommitHook: "hook-post",
}

// hookTemplate is the shell script written to .git/hooks/<name>. It
// delegates to the lazyver binary that installed it, forwarding any
// arguments provided by git.
var hookTemplate = `#!/bin/sh
%s
exec "%s" %s "$@"
`

// Install writes all managed hooks into the repository's hooks directory.
//
// Parameters:
//   - repoPath: path to the repository working tree (hooks go into its
//     .git/hooks subdirectory).
//   - binaryPath: absolute path of the lazyver executable that should be
//     invoked by the hooks (obtained via os.Executable() by callers).
//
// Hooks already managed by lazyver (identified by Marker) are overwritten on
// every install so updates to the binary path are picked up, and are always
// marked executable (0755). A pre-existing hook file that does NOT contain
// the marker is treated as foreign: it is left untouched and an error is
// returned, so lazyver never clobbers a user's own hook silently. Removing
// or integrating the foreign hook is up to the repository owner.
func Install(repoPath, binaryPath string) error {
	hooksDir := filepath.Join(repoPath, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("create hooks dir: %w", err)
	}
	for name, subcommand := range hooks {
		hookPath := filepath.Join(hooksDir, name)
		data, err := os.ReadFile(hookPath)
		switch {
		case err == nil && !strings.Contains(string(data), Marker):
			return fmt.Errorf(
				"refusing to overwrite %s hook %q: it was not installed by lazyver; remove it or integrate lazyver into it manually",
				name, hookPath)
		case err != nil && !os.IsNotExist(err):
			return fmt.Errorf("read %s hook: %w", name, err)
		}
		script := fmt.Sprintf(hookTemplate, Marker, binaryPath, subcommand)
		if err := os.WriteFile(hookPath, []byte(script), 0o755); err != nil {
			return fmt.Errorf("write %s hook: %w", name, err)
		}
	}
	return nil
}

// IsInstalled reports whether all lazyver-managed hooks exist in the
// repository at repoPath AND were written by lazyver (i.e. carry Marker).
func IsInstalled(repoPath string) bool {
	for name := range hooks {
		data, err := os.ReadFile(filepath.Join(repoPath, ".git", "hooks", name))
		if err != nil || !strings.Contains(string(data), Marker) {
			return false
		}
	}
	return true
}
