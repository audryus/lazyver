package gitrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitAvailable reports whether the git binary can be executed at all; all
// integration tests skip gracefully when it is missing.
func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// newRepo creates a temporary directory initialized as a git repository
// with user identity configured (needed to create commits in CI sandboxes).
func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@lazyver.local")
	run("config", "user.name", "lazyver test")
	return dir
}

// commitFile writes content to a file inside the repo and commits it with
// the given message, returning the commit hash.
func commitFile(t *testing.T, dir, file, message string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, file), []byte(message), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return string(out)
	}
	run("add", file)
	run("commit", "-m", message)
	return gitOutput(t, dir, "rev-parse", "HEAD")
}

// gitOutput runs a git command and returns its trimmed stdout.
func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

func TestGitAvailableGuard(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git binary not available")
	}
}

// TestIsRepository distinguishes a real repository from a plain directory.
func TestIsRepository(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git binary not available")
	}
	repo := newRepo(t)
	if !IsRepository(repo) {
		t.Error("IsRepository(repo) = false, want true")
	}
	if IsRepository(t.TempDir()) {
		t.Error("IsRepository(empty dir) = true, want false")
	}
}

// TestHeadHashAndCommitsBetween validates full-history reading, ordering
// (oldest first), exclusive lower bound and message fidelity including
// multi-line bodies with BREAKING CHANGE footers.
func TestHeadHashAndCommitsBetween(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git binary not available")
	}
	repo := newRepo(t)
	h1 := commitFile(t, repo, "a.txt", "feat: first")
	h2 := commitFile(t, repo, "b.txt", "fix: second\n\nBREAKING CHANGE: yes")

	head, err := HeadHash(repo)
	if err != nil {
		t.Fatalf("HeadHash() error = %v", err)
	}
	if head != h2 {
		t.Errorf("HeadHash() = %q, want %q", head, h2)
	}

	all, err := CommitsBetween(repo, "")
	if err != nil {
		t.Fatalf("CommitsBetween(all) error = %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d commits, want 2", len(all))
	}
	if all[0].Hash != h1 || all[1].Hash != h2 {
		t.Error("commits are not in oldest-first order")
	}
	if !contains(all[1].Message, "BREAKING CHANGE") {
		t.Errorf("full body not captured: %q", all[1].Message)
	}

	sinceSecond, err := CommitsBetween(repo, h1)
	if err != nil {
		t.Fatalf("CommitsBetween(since h1) error = %v", err)
	}
	if len(sinceSecond) != 1 || sinceSecond[0].Hash != h2 {
		t.Errorf("exclusive range wrong: %+v", sinceSecond)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// TestCommitsBetweenEmptyRepo ensures an unborn branch yields no error and
// no commits.
func TestCommitsBetweenEmptyRepo(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git binary not available")
	}
	repo := newRepo(t)
	commits, err := CommitsBetween(repo, "")
	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if commits != nil {
		t.Errorf("commits = %+v, want nil", commits)
	}
}

// TestStageFile confirms that an ignored file is force-added to the index.
func TestStageFile(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git binary not available")
	}
	repo := newRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".lazyver.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("add", ".gitignore")
	run("commit", "-m", "chore: gitignore")
	if err := os.WriteFile(filepath.Join(repo, ".lazyver.yaml"), []byte("version: v0.0.1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := StageFile(repo, ".lazyver.yaml"); err != nil {
		t.Fatalf("StageFile() error = %v", err)
	}
	out, err := Run(repo, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(out, ".lazyver.yaml") {
		t.Errorf("ignored file not staged: status = %q", out)
	}
}
