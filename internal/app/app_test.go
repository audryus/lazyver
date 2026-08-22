package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"codeberg.org/audryus/lazyver/internal/statefile"
)

// gitAvailable skips the suite when the git binary is missing.
func gitAvailable(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
}

// newRepo creates a temp git repository with a test identity configured.
func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "init", "-b", "main")
	run(t, dir, "config", "user.email", "test@lazyver.local")
	run(t, dir, "config", "user.name", "lazyver test")
	return dir
}

// run executes a git command inside dir, failing the test on error. It
// always disables real git hooks: the hook installed by app.Run points to
// os.Executable(), which during tests is the test binary itself — letting
// git invoke it would recursively re-run the whole suite.
func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	hooksPath := filepath.Join(t.TempDir(), "no-hooks") // empty dir = no hooks run
	full := append([]string{"-c", "core.hooksPath=" + hooksPath}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

// commit writes a file and commits it with the given message.
func commit(t *testing.T, dir, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte(message), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "add", "file.txt")
	run(t, dir, "commit", "-m", message)
}

// loadState is a test helper that must succeed.
func loadState(t *testing.T, dir string) *statefile.State {
	t.Helper()
	state, err := statefile.Load(dir)
	if err != nil || state == nil {
		t.Fatalf("state missing or unreadable: %v", err)
	}
	return state
}

// TestRunInitializesFromFullHistory covers the one-time full scan: three
// commits (feat, fix, fix) must produce v0.1.2 and install the hook.
func TestRunInitializesFromFullHistory(t *testing.T) {
	gitAvailable(t)
	repo := newRepo(t)
	commit(t, repo, "feat: add core")
	commit(t, repo, "fix: patch core")
	commit(t, repo, "fix: patch again")

	version, err := Run(statefile.KindSemver, repo)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if version != "v0.1.2" {
		t.Errorf("version = %q, want v0.1.2", version)
	}
	if state := loadState(t, repo); state.Pending {
		t.Error("pending should be false after initialization")
	}
	if !hookInstalled(t, repo) {
		t.Error("commit-msg hook not installed")
	}
}

// hookInstalled reads the commit-msg hook to confirm installation.
func hookInstalled(t *testing.T, repo string) bool {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, ".git", "hooks", "commit-msg"))
	return err == nil && strings.Contains(string(data), "lazyver")
}

// TestRunIncrementalOnlyNewCommits ensures a second Run does not recount the
// whole history: only commits after the stored hash are applied.
func TestRunIncrementalOnlyNewCommits(t *testing.T) {
	gitAvailable(t)
	repo := newRepo(t)
	commit(t, repo, "feat: one")
	if _, err := Run(statefile.KindSemver, repo); err != nil {
		t.Fatal(err)
	}

	commit(t, repo, "fix: two")
	commit(t, repo, "chore: three")
	version, err := Run(statefile.KindSemver, repo)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if version != "v0.2.0" {
		t.Errorf("version = %q, want v0.2.0", version)
	}
}

// TestHookBumpThenRunNoDoubleCount is the critical end-to-end scenario:
// the hook bumps and stages the version file, the commit is created, and a
// later Run must NOT count that commit again.
func TestHookBumpThenRunNoDoubleCount(t *testing.T) {
	gitAvailable(t)
	repo := newRepo(t)
	commit(t, repo, "feat: base")
	if _, err := Run(statefile.KindSemver, repo); err != nil { // v0.1.0
		t.Fatal(err)
	}

	// Simulate the commit-msg hook: message file written by git.
	msgFile := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	if err := os.WriteFile(msgFile, []byte("feat: hook driven change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := HandleHookMessage(repo, msgFile); err != nil {
		t.Fatalf("HandleHookMessage() error = %v", err)
	}

	// The hook must have staged the version file.
	status, _ := exec.Command("git", "-C", repo, "status", "--porcelain").CombinedOutput()
	if !strings.Contains(string(status), ".lazyver.yaml") {
		t.Fatalf("version file not staged by hook: %s", status)
	}

	// Finish the commit like git would after a successful commit-msg hook.
	run(t, repo, "commit", "-m", "feat: hook driven change")

	state := loadState(t, repo)
	if state.Version != "v0.2.0" {
		t.Fatalf("after hook version = %q, want v0.2.0", state.Version)
	}
	if !state.Pending {
		t.Fatal("hook should mark the state pending")
	}

	// Reconciliation run: the pending commit must be skipped, not double-counted.
	version, err := Run(statefile.KindSemver, repo)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if version != "v0.2.0" {
		t.Errorf("version after reconcile = %q, want v0.2.0 (no double bump)", version)
	}
	if state := loadState(t, repo); state.Pending {
		t.Error("pending flag should be cleared after reconciliation")
	}
}

// TestLazyModeEndToEnd validates initialization, hook bump and
// reconciliation for the lazy (commit-count) mode.
func TestLazyModeEndToEnd(t *testing.T) {
	gitAvailable(t)
	repo := newRepo(t)
	commit(t, repo, "first")
	commit(t, repo, "second")
	commit(t, repo, "third")
	commit(t, repo, "fourth")
	commit(t, repo, "fifth")

	version, err := Run(statefile.KindLazy, repo)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if version != "v0.0.5" {
		t.Errorf("version = %q, want v0.0.5", version)
	}

	msgFile := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	os.WriteFile(msgFile, []byte("sixth\n"), 0o644)
	if err := HandleHookMessage(repo, msgFile); err != nil {
		t.Fatalf("HandleHookMessage() error = %v", err)
	}
	run(t, repo, "commit", "-m", "sixth")

	if version, _ = Run(statefile.KindLazy, repo); version != "v0.0.6" {
		t.Errorf("version = %q, want v0.0.6", version)
	}
}

// TestRunKindMismatch refuses to operate on a repository initialized with a
// different mode.
func TestRunKindMismatch(t *testing.T) {
	gitAvailable(t)
	repo := newRepo(t)
	commit(t, repo, "feat: one")
	if _, err := Run(statefile.KindSemver, repo); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(statefile.KindLazy, repo); err == nil {
		t.Error("Run with different kind should fail")
	}
}

// TestRunOutsideRepository fails fast on non-git directories.
func TestRunOutsideRepository(t *testing.T) {
	if _, err := Run(statefile.KindSemver, t.TempDir()); err == nil {
		t.Error("Run outside a repository should fail")
	}
}
