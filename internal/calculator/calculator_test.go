package calculator

import (
	"testing"

	"github.com/audryus/lazyver/internal/statefile"
)

// TestApplySemVer verifies cumulative bumps across a sequence of messages,
// including resets after major and minor changes.
func TestApplySemVer(t *testing.T) {
	state := &statefile.State{Kind: statefile.KindSemver}
	messages := []string{
		"feat: a",      // 0.1.0
		"fix: b",       // 0.1.1
		"fix: c",       // 0.1.2
		"feat!: d",     // 1.0.0
		"docs: e",      // 1.1.0
		"no bump here", // 1.1.0
	}
	ApplySemVer(state, messages)
	if got, want := state.Version, "v1.1.0"; got != want {
		t.Errorf("version = %q, want %q", got, want)
	}
}

// TestApplySemVerSingle covers the hook path: exactly one bump per call and
// the boolean result signalling whether anything changed.
func TestApplySemVerSingle(t *testing.T) {
	tests := []struct {
		message   string
		wantBump  bool
		wantMajor int
		wantMinor int
		wantPatch int
	}{
		{"feat: x", true, 0, 1, 0},
		{"fix: x", true, 0, 0, 1},
		{"chore!: x", true, 1, 0, 0},
		{"random text", false, 0, 0, 0},
	}
	for _, tt := range tests {
		state := &statefile.State{}
		got := ApplySemVerSingle(state, tt.message)
		if got != tt.wantBump || state.Major != tt.wantMajor ||
			state.Minor != tt.wantMinor || state.Patch != tt.wantPatch {
			t.Errorf("ApplySemVerSingle(%q): bumped=%v state=%d.%d.%d, want bumped=%v %d.%d.%d",
				tt.message, got, state.Major, state.Minor, state.Patch,
				tt.wantBump, tt.wantMajor, tt.wantMinor, tt.wantPatch)
		}
	}
}

// TestApplyLazy documents the digit-splitting scheme with fresh states.
func TestApplyLazy(t *testing.T) {
	tests := []struct {
		count     int
		wantMajor int
		wantMinor int
		wantPatch int
	}{
		{0, 0, 0, 0},
		{5, 0, 0, 5},
		{42, 0, 4, 2},
		{339, 3, 3, 9},
		{1000, 10, 0, 0},
	}
	for _, tt := range tests {
		state := &statefile.State{}
		ApplyLazy(state, tt.count)
		if state.Major != tt.wantMajor || state.Minor != tt.wantMinor || state.Patch != tt.wantPatch {
			t.Errorf("count=%d: got %d.%d.%d, want %d.%d.%d",
				tt.count, state.Major, state.Minor, state.Patch,
				tt.wantMajor, tt.wantMinor, tt.wantPatch)
		}
	}
}

// TestApplyLazyIncremental ensures an existing flat count is preserved and
// extended rather than recomputed from scratch.
func TestApplyLazyIncremental(t *testing.T) {
	state := &statefile.State{Major: 3, Minor: 3, Patch: 9} // flat count 339
	ApplyLazy(state, 2)
	if state.Version != "v3.4.1" {
		t.Errorf("version = %q, want v3.4.1 (339 + 2 = 341)", state.Version)
	}
}

// TestNormalizeKind validates kind handling including the empty default and
// rejection of unknown values.
func TestNormalizeKind(t *testing.T) {
	if k, err := NormalizeKind(""); err != nil || k != "semver" {
		t.Errorf("empty kind -> (%q, %v), want semver, nil", k, err)
	}
	if k, err := NormalizeKind("lazy"); err != nil || k != "lazy" {
		t.Errorf("lazy -> (%q, %v)", k, err)
	}
	if k, err := NormalizeKind(" SEMVER "); err != nil || k != "semver" {
		t.Errorf("spaces trimmed -> (%q, %v)", k, err)
	}
	if _, err := NormalizeKind("crazy"); err == nil {
		t.Error("unknown kind should error")
	}
}
