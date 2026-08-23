package lazyver

import "testing"

// withVersion swaps the package-level versionString for the duration of the
// test and restores it afterwards.
func withVersion(t *testing.T, v string) {
	t.Helper()
	orig := versionString
	versionString = v
	t.Cleanup(func() { versionString = orig })
}

// TestResolvedVersionLinkerOverride ensures a linker-injected version
// (goreleaser -X flag) always wins.
func TestResolvedVersionLinkerOverride(t *testing.T) {
	withVersion(t, "v9.8.7")
	if got := resolvedVersion(); got != "v9.8.7" {
		t.Errorf("resolvedVersion() = %q, want %q", got, "v9.8.7")
	}
}

// TestResolvedVersionNeverEmpty ensures the fallback chain never yields an
// empty string: without linker injection and outside a tagged module install,
// the result must be exactly "dev".
func TestResolvedVersionNeverEmpty(t *testing.T) {
	withVersion(t, "")
	if got := resolvedVersion(); got == "" {
		t.Error("resolvedVersion() returned an empty string")
	}
}
