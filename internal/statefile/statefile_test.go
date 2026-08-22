package statefile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadMissing verifies that Load returns nil state (and no error) when
// the control file does not exist yet.
func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()
	state, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if state != nil {
		t.Fatalf("Load() = %+v, want nil for missing file", state)
	}
}

// TestSaveAndLoadRoundTrip verifies that a saved state survives a load with
// all fields intact and that the version field matches the components.
func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := &State{
		Version: "v1.2.3", Major: 1, Minor: 2, Patch: 3,
		Kind: KindSemver, LastHash: "abc123", Pending: true,
	}
	if err := Save(dir, in); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	out, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if out == nil || out.Version != "v1.2.3" || out.Major != 1 || out.Minor != 2 ||
		out.Patch != 3 || out.Kind != KindSemver || out.LastHash != "abc123" || !out.Pending {
		t.Fatalf("round trip mismatch: got %+v", out)
	}
	if out.Last.IsZero() {
		t.Error("Save() should stamp the last timestamp")
	}
}

// TestFileNameIsDotPrefixed ensures the control file lives at the repo root
// under its documented name.
func TestFileNameIsDotPrefixed(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, &State{Version: "v0.0.1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, FileName)); err != nil {
		t.Fatalf("expected %s in dir: %v", FileName, err)
	}
}

// TestSaveYAMLLayout documents the on-disk YAML shape so accidental schema
// changes are caught.
func TestSaveYAMLLayout(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, &State{Version: "v1.2.3", Major: 1, Minor: 2, Patch: 3, Kind: "lazy"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, FileName))
	text := string(data)
	for _, key := range []string{"version:", "major:", "minor:", "patch:", "kind:", "lastHash:", "pending:", "last:"} {
		if !strings.Contains(text, key) {
			t.Errorf("yaml output missing key %q:\n%s", key, text)
		}
	}
}

// TestRemove deletes an existing file and tolerates a missing one.
func TestRemove(t *testing.T) {
	dir := t.TempDir()
	if err := Remove(dir); err != nil { // missing file must not error
		t.Fatalf("Remove() on missing file error = %v", err)
	}
	if err := Save(dir, &State{}); err != nil {
		t.Fatal(err)
	}
	if err := Remove(dir); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, FileName)); !os.IsNotExist(err) {
		t.Error("Remove() did not delete the state file")
	}
}

// TestFormat checks the canonical version rendering.
func TestFormat(t *testing.T) {
	s := &State{Major: 12, Minor: 3, Patch: 44}
	if got := s.Format(); got != "v12.3.44" {
		t.Errorf("Format() = %q, want v12.3.44", got)
	}
}
