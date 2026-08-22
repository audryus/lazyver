// Package app orchestrates lazyver's workflows on top of the lower-level
// packages: git access, message classification, state persistence and hook
// management.
//
// Two entry points exist:
//
//   - Run: invoked by the CLI commands "lazyver semver" and "lazyver lazy".
//     It initializes the control file when missing (reading the full commit
//     history once) and afterwards only inspects commits newer than the
//     stored hash. It also (re)installs the commit-msg hook on every run.
//
//   - HandleHookMessage: invoked by the commit-msg hook itself. It applies a
//     single bump based on the message being committed, marks the state as
//     pending and stages the version file so it travels with the commit.
package app

import (
	"fmt"
	"os"

	"codeberg.org/audryus/lazyver/internal/calculator"
	"codeberg.org/audryus/lazyver/internal/gitrepo"
	"codeberg.org/audryus/lazyver/internal/hookmgr"
	"codeberg.org/audryus/lazyver/internal/statefile"
)

// Run synchronizes the version file in dir using the given versioning kind,
// installs the commit-msg hook and returns the current version string
// (e.g. "v1.2.3").
//
// Behaviour:
//
//   - When no ".lazyver.yaml" exists, the full repository history is read
//     once and the version is computed from scratch (initialization).
//   - When the file exists, only commits after the stored hash are examined.
//     If the previous operation was a hook-driven bump (pending flag), the
//     very first of those commits is skipped because it was already counted.
//   - The hook is installed/refreshed unconditionally so the binary path is
//     always up to date.
func Run(kind, dir string) (string, error) {
	kind, err := calculator.NormalizeKind(kind)
	if err != nil {
		return "", err
	}
	if !gitrepo.IsRepository(dir) {
		return "", fmt.Errorf("%q is not inside a git working tree", dir)
	}

	state, err := statefile.Load(dir)
	if err != nil {
		return "", err
	}

	if state == nil {
		state, err = initialize(dir, kind)
		if err != nil {
			return "", err
		}
	} else {
		if state.Kind != "" && state.Kind != kind {
			return "", fmt.Errorf("repository initialized with kind %q; refusing to run with %q (delete %s to reset)",
				state.Kind, kind, statefile.FileName)
		}
		if err := increment(dir, state); err != nil {
			return "", err
		}
	}

	if err := installHook(dir); err != nil {
		return "", err
	}
	if err := statefile.Save(dir, state); err != nil {
		return "", err
	}
	return state.Version, nil
}

// HandleHookMessage is the entry point used by the commit-msg hook. It reads
// the commit message from messageFile (the path git passes as $1), applies
// exactly one version bump for it, marks the state pending and stages the
// version file so it is included in the ongoing commit.
//
// If the repository was never initialized, initialization happens first so
// that even a brand-new repository gets a correct baseline version.
func HandleHookMessage(dir, messageFile string) error {
	data, err := os.ReadFile(messageFile)
	if err != nil {
		return fmt.Errorf("read commit message file: %w", err)
	}
	message := string(data)

	state, err := statefile.Load(dir)
	if err != nil {
		return err
	}
	if state == nil {
		if state, err = initialize(dir, statefile.KindSemver); err != nil {
			return err
		}
	} else if !gitrepo.IsRepository(dir) {
		return fmt.Errorf("%q is not inside a git working tree", dir)
	}

	bumped := false
	switch state.Kind {
	case statefile.KindLazy:
		calculator.ApplyLazy(state, 1)
		bumped = true
	default:
		bumped = calculator.ApplySemVerSingle(state, message)
	}

	if bumped {
		// The commit being created does not have a hash yet; flagging
		// "pending" tells the next Run to skip exactly one commit (this
		// one) when reconciling, avoiding a double bump.
		state.Pending = true
	}
	if err := statefile.Save(dir, state); err != nil {
		return err
	}
	return gitrepo.StageFile(dir, statefile.FileName)
}

// initialize performs the one-time full-history scan: every commit message
// since the beginning of the repository is replayed through the chosen
// strategy and the resulting state is anchored at the current HEAD.
func initialize(dir, kind string) (*statefile.State, error) {
	headHash, err := gitrepo.HeadHash(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve HEAD (does the branch have any commits?): %w", err)
	}

	commits, err := gitrepo.CommitsBetween(dir, "")
	if err != nil {
		return nil, err
	}

	state := &statefile.State{Kind: kind}
	messages := make([]string, 0, len(commits))
	for _, c := range commits {
		messages = append(messages, c.Message)
	}

	switch kind {
	case statefile.KindLazy:
		calculator.ApplyLazy(state, len(commits))
	default:
		calculator.ApplySemVer(state, messages)
	}

	state.LastHash = headHash
	state.Pending = false
	state.Version = state.Format()
	return state, nil
}

// increment brings an existing state up to date by inspecting only the
// commits created after state.LastHash. It honors the pending flag set by
// the hook so a hook-bumped commit is never counted twice.
func increment(dir string, state *statefile.State) error {
	commits, err := gitrepo.CommitsBetween(dir, state.LastHash)
	if err != nil {
		return err
	}

	if state.Pending && len(commits) > 0 {
		commits = commits[1:]
		state.Pending = false
	}

	switch state.Kind {
	case statefile.KindLazy:
		calculator.ApplyLazy(state, len(commits))
	default:
		messages := make([]string, 0, len(commits))
		for _, c := range commits {
			messages = append(messages, c.Message)
		}
		calculator.ApplySemVer(state, messages)
	}

	headHash, err := gitrepo.HeadHash(dir)
	if err != nil {
		return fmt.Errorf("resolve HEAD: %w", err)
	}
	state.LastHash = headHash
	return nil
}

// installHook delegates to hookmgr, resolving the running binary path so
// the generated script keeps working even after `go install` upgrades move
// the binary around.
func installHook(dir string) error {
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve lazyver executable path: %w", err)
	}
	return hookmgr.Install(dir, binaryPath)
}
