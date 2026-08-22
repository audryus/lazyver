// Package hookmgr installs and manages the lazyver commit-msg hook inside a
// git repository. The hook is what makes lazyver "zero effort": the user
// just commits normally, and the version file is bumped, written and staged
// automatically before the commit is finalized.
package hookmgr

import (
	"fmt"
	"os"
	"path/filepath"
)

// Marker identifies hooks written by lazyver so that reinstalling never
// clobbers a foreign hook silently.
const Marker = "# installed by lazyver"

// HookName is the git hook being used. commit-msg (not pre-commit) is used
// because the commit message must already exist for the bump rules to be
// applied — pre-commit runs before the message is written.
const HookName = "commit-msg"

// hookTemplate is the shell script written to .git/hooks/commit-msg. It
// delegates to the lazyver binary that installed it, forwarding the message
// file path provided by git.
var hookTemplate = `#!/bin/sh
%s
exec "%s" hook "$@"
`

// Install writes the commit-msg hook into the repository's hooks directory.
//
// Parameters:
//   - repoPath:  path to the repository working tree (hook goes into its
//     .git/hooks subdirectory).
//   - binaryPath: absolute path of the lazyver executable that should be
//     invoked by the hook (obtained via os.Executable() by callers).
//
// The hook is overwritten on every install so that updates to the binary
// path are picked up, and is always marked executable (0755).
func Install(repoPath, binaryPath string) error {
	hooksDir := filepath.Join(repoPath, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("create hooks dir: %w", err)
	}
	script := fmt.Sprintf(hookTemplate, Marker, binaryPath)
	hookPath := filepath.Join(hooksDir, HookName)
	if err := os.WriteFile(hookPath, []byte(script), 0o755); err != nil {
		return fmt.Errorf("write %s hook: %w", HookName, err)
	}
	return nil
}

// IsInstalled reports whether a lazyver-managed commit-msg hook already
// exists in the repository at repoPath.
func IsInstalled(repoPath string) bool {
	data, err := os.ReadFile(filepath.Join(repoPath, ".git", "hooks", HookName))
	return err == nil && string(data) != ""
}
