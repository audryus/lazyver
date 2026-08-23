package hookmgr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newFakeRepo creates a directory that mimics a git repository's .git/hooks
// layout without needing a real repository.
func newFakeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestInstall writes the hooks and checks content, executable bit and that
// a second install overwrites cleanly (refreshing the binary path).
func TestInstall(t *testing.T) {
	repo := newFakeRepo(t)

	if err := Install(repo, "/usr/local/bin/lazyver"); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	hookPath := filepath.Join(repo, ".git", "hooks", CommitMsgHook)
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("hook not written: %v", err)
	}
	script := string(data)
	for _, want := range []string{"#!/bin/sh", Marker, "/usr/local/bin/lazyver"} {
		if !strings.Contains(script, want) {
			t.Errorf("hook script missing %q:\n%s", want, script)
		}
	}

	// Both managed hooks must exist and delegate to the right subcommands.
	postPath := filepath.Join(repo, ".git", "hooks", PostCommitHook)
	postData, err := os.ReadFile(postPath)
	if err != nil {
		t.Fatalf("post-commit hook not written: %v", err)
	}
	if !strings.Contains(string(postData), "hook-post") {
		t.Errorf("post-commit hook does not call hook-post:\n%s", postData)
	}
	if !strings.Contains(script, `hook "$@"`) {
		t.Errorf("commit-msg hook does not forward args:\n%s", script)
	}

	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Error("hook is not executable")
	}

	// Reinstall with a different binary path must replace the old one.
	if err := Install(repo, "/opt/bin/lazyver"); err != nil {
		t.Fatalf("reinstall error = %v", err)
	}
	data, _ = os.ReadFile(hookPath)
	if strings.Contains(string(data), "/usr/local/bin/lazyver") {
		t.Error("old binary path still present after reinstall")
	}
}

// TestIsInstalled reports true only for an existing hook file.
func TestIsInstalled(t *testing.T) {
	repo := newFakeRepo(t)
	if IsInstalled(repo) {
		t.Error("IsInstalled before install = true, want false")
	}
	if err := Install(repo, "/bin/lazyver"); err != nil {
		t.Fatal(err)
	}
	if !IsInstalled(repo) {
		t.Error("IsInstalled after install = false, want true")
	}
	// Removing one of the two managed hooks must make IsInstalled fail.
	if err := os.Remove(filepath.Join(repo, ".git", "hooks", PostCommitHook)); err != nil {
		t.Fatal(err)
	}
	if IsInstalled(repo) {
		t.Error("IsInstalled with missing post-commit hook = true, want false")
	}
}

// TestInstallCreatesHooksDir ensures installation works even when .git/hooks
// does not exist yet.
func TestInstallCreatesHooksDir(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Install(repo, "/bin/lazyver"); err != nil {
		t.Fatalf("Install() without hooks dir error = %v", err)
	}
}

// TestInstallRefusesForeignHook ensures lazyver never silently replaces a
// pre-existing hook it did not install (no lazyver marker inside).
func TestInstallRefusesForeignHook(t *testing.T) {
	repo := newFakeRepo(t)
	foreign := "#!/bin/sh\necho my own hook\n"
	msgPath := filepath.Join(repo, ".git/hooks", CommitMsgHook)
	postPath := filepath.Join(repo, ".git/hooks", PostCommitHook)
	for _, p := range []string{msgPath, postPath} {
		if err := os.WriteFile(p, []byte(foreign), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	err := Install(repo, "/bin/lazyver")
	if err == nil {
		t.Fatal("Install() over a foreign hook = nil error, want refusal")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Errorf("error %q does not mention the refusal", err)
	}

	// The foreign content must remain untouched on both hooks.
	for _, p := range []string{msgPath, postPath} {
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(data) != foreign {
			t.Errorf("%s was modified:\n%s", p, data)
		}
	}

	// A mixed state is also refused: one managed hook plus one foreign hook.
	if werr := os.WriteFile(msgPath, []byte("#!/bin/sh\n"+Marker+"\nexec x\n"), 0o755); werr != nil {
		t.Fatal(werr)
	}
	if err := Install(repo, "/bin/lazyver"); err == nil {
		t.Error("Install() with one foreign hook = nil error, want refusal")
	}
}

// TestInstallReplacesOwnHook ensures hooks carrying the lazyver marker are
// freely overwritten (binary path refresh).
func TestInstallReplacesOwnHook(t *testing.T) {
	repo := newFakeRepo(t)
	if err := Install(repo, "/old/lazyver"); err != nil {
		t.Fatal(err)
	}
	if err := Install(repo, "/new/lazyver"); err != nil {
		t.Fatalf("Install() over own hook error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repo, ".git/hooks", CommitMsgHook))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "/new/lazyver") || strings.Contains(string(data), "/old/lazyver") {
		t.Errorf("managed hook not refreshed:\n%s", data)
	}
}

// TestIsInstalledRequiresMarker ensures a foreign hook file that merely has
// the managed name is NOT considered installed by lazyver.
func TestIsInstalledRequiresMarker(t *testing.T) {
	repo := newFakeRepo(t)
	path := filepath.Join(repo, ".git/hooks", CommitMsgHook)
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho foreign\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if IsInstalled(repo) {
		t.Error("IsInstalled with foreign commit-msg hook = true, want false")
	}
}
