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

	// dirty tracks whether the state materially changed and therefore must
	// be persisted. Pure bookkeeping (zeroing pendingCount, advancing
	// lastHash with no new commits to apply) is deferred to the next hook
	// invocation, so a manual run never leaves the working tree dirty.
	dirty := false

	if state == nil {
		state, err = initialize(dir, kind)
		if err != nil {
			return "", err
		}
		dirty = true
	} else {
		if state.Kind != "" && state.Kind != kind {
			return "", fmt.Errorf("repository initialized with kind %q; refusing to run with %q (delete %s to reset)",
				state.Kind, kind, statefile.FileName)
		}
		applied, err := increment(dir, state)
		if err != nil {
			return "", err
		}
		dirty = applied > 0
	}

	if err := installHook(dir); err != nil {
		return "", err
	}
	if dirty {
		if err := statefile.Save(dir, state); err != nil {
			return "", err
		}
	}
	return state.Version, nil
}

// HandleHookMessage is the entry point used by the commit-msg hook. It reads
// the commit message from messageFile (the path git passes as $1), applies
// exactly one version bump for it and marks the state pending. The bumped
// version file is staged right away, but git has already snapshotted the
// tree by the time commit-msg runs, so the staged copy only reaches the
// commit through the post-commit amend (HandlePostCommit) — verified
// empirically: ls-tree HEAD shows no version file until the amend lands.
//
// If the repository was never initialized, initialization happens first so
// that even the very first commit of a brand-new repository gets a correct
// baseline version (an unborn branch is tolerated).
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
		// The commit being created does not have a hash yet; counting it
		// as pending tells the post-commit hook to anchor the state at the
		// rewritten HEAD once this commit exists (and, defensively, to fold
		// the version file in when it missed the commit).
		state.PendingCount++
	}
	if err := statefile.Save(dir, state); err != nil {
		return err
	}
	return gitrepo.StageFile(dir, statefile.FileName)
}

// HandlePostCommit is the entry point used by the post-commit hook. When a
// bump is pending (set by the commit-msg phase), it amends HEAD so the
// version file becomes part of the commit that triggered it.
//
// It deliberately does NOT rewrite the state file afterwards: the file staged
// before the amend already carries the correct bump, and saving different
// content would either dirty the working tree or require another amend whose
// own post-commit run would see stale state again. Reconciliation of
// lastHash/pendingCount happens on the next Run.
//
// Safety rules:
//   - nothing happens when no bump is pending;
//   - nothing happens when HEAD already contains a byte-identical version
//     file. This is what bounds the cycle: the ordinary staging during
//     commit-msg usually puts the file in the commit directly, and when the
//     defensive amend does run, its own post-commit invocation finds the
//     file already folded in and returns — no infinite amend recursion;
//   - the amend is skipped when HEAD is already reachable from a remote,
//     i.e. the commit was pushed — rewriting public history is never done.
func HandlePostCommit(dir string) error {
	state, err := statefile.Load(dir)
	if err != nil || state == nil || state.PendingCount == 0 {
		return err
	}
	// Never rewrite commits that are already public.
	if gitrepo.IsPublished(dir) {
		return nil
	}
	// Break the amend/post-commit recursion: once HEAD carries the very same
	// version file, either the index staging during commit-msg already
	// included it or a previous post-commit run folded it in — there is
	// nothing left to do.
	if gitrepo.HeadContainsFile(dir, statefile.FileName) {
		return nil
	}
	if err := gitrepo.StageFile(dir, statefile.FileName); err != nil {
		return err
	}
	if err := gitrepo.AmendHead(dir); err != nil {
		return fmt.Errorf("amend commit to include %s: %w", statefile.FileName, err)
	}
	return nil
}

// initialize performs the one-time full-history scan: every commit message
// since the beginning of the repository is replayed through the chosen
// strategy and the resulting state is anchored at the current HEAD.
func initialize(dir, kind string) (*statefile.State, error) {
	// An unborn branch (brand-new repository, e.g. during the commit-msg
	// phase of its very first commit) cannot resolve HEAD yet. Anchoring is
	// then deferred: lastHash stays empty and is recorded on the next
	// reconciliation instead of failing the whole operation.
	headHash, _ := gitrepo.HeadHash(dir)

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
	state.PendingCount = 0
	state.Version = state.Format()
	return state, nil
}

// increment brings an existing state up to date by inspecting only the
// commits created after state.LastHash. The PendingCount skip is a legacy
// reconciliation path: since HandlePostCommit now anchors lastHash right
// after every hook-driven commit, pendingCount is normally zero here and
// nothing needs to be skipped.
//
// It returns how many NEW commits were actually applied (zero means the
// state was already up to date and callers may skip persisting it).
func increment(dir string, state *statefile.State) (int, error) {
	commits, err := gitrepo.CommitsBetween(dir, state.LastHash)
	if err != nil {
		return 0, err
	}

	// Legacy states (written before post-commit anchoring existed, or when
	// the amend was skipped for a published HEAD) may still carry pending
	// commits: they are always the most recent ones, so skip exactly
	// PendingCount of them from the tail.
	skip := state.PendingCount
	if skip > len(commits) {
		skip = len(commits)
	}
	commits = commits[:len(commits)-skip]
	// Bookkeeping is updated in memory even when nothing is applied; it
	// only reaches disk if the caller decides the change is material.
	defer func() { state.PendingCount = 0 }()

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
		return 0, fmt.Errorf("resolve HEAD: %w", err)
	}
	state.LastHash = headHash
	return len(commits), nil
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
