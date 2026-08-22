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

// TestInstall writes the hook and checks content, executable bit and that a
// second install overwrites cleanly (refreshing the binary path).
func TestInstall(t *testing.T) {
	repo := newFakeRepo(t)

	if err := Install(repo, "/usr/local/bin/lazyver"); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	hookPath := filepath.Join(repo, ".git", "hooks", HookName)
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("hook not written: %v", err)
	}
	script := string(data)
	for _, want := range []string{"#!/bin/sh", Marker, "/usr/local/bin/lazyver", `hook "$@"`} {
		if !strings.Contains(script, want) {
			t.Errorf("hook script missing %q:\n%s", want, script)
		}
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
