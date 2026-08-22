package commitmsg

import "testing"

// TestClassify covers the full decision table of message classification:
// major indicators (bang after the type and BREAKING CHANGE footers), minor
// types, patch types, loose prefixes, casing and unrelated messages.
func TestClassify(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    Level
	}{
		{"empty message", "", LevelNone},
		{"unrelated subject", "updated some stuff", LevelNone},
		{"merge commit", "Merge branch 'main' into dev", LevelNone},

		// Major: bang must come right after the type.
		{"bang after feat", "feat!: drop support for node 16", LevelMajor},
		{"bang after fix with scope", "fix(api)!: return 404 instead of 500", LevelMajor},
		{"bang without colon is not major", "wow! great feature", LevelNone},
		{"exclamation elsewhere is not major", "fixed the bug!!!", LevelPatch},
		{"breaking change footer", "feat: new api\n\nBREAKING CHANGE: endpoint renamed", LevelMajor},
		{"breaking changes plural footer", "chore: cleanup\n\nBREAKING CHANGES: several", LevelMajor},
		{"breaking footer lowercase flag", "feat: new api\n\nbreaking change: endpoint renamed", LevelMajor},

		// Minor types (loose prefix match, case-insensitive).
		{"feat", "feat: add login page", LevelMinor},
		{"chore", "chore: update deps", LevelMinor},
		{"build", "build: bump go to 1.24", LevelMinor},
		{"docs", "docs: rewrite readme", LevelMinor},
		{"ci", "ci: run tests on windows too", LevelMinor},
		{"test", "test: cover auth service", LevelMinor},
		{"style", "style: format code", LevelMinor},
		{"loose prefix feature", "feature: fancy new thing", LevelMinor},
		{"uppercase feat", "FEAT: shouty commit", LevelMinor},

		// Patch types.
		{"fix", "fix: null pointer on logout", LevelPatch},
		{"perf", "perf: cache user lookups", LevelPatch},
		{"revert", "revert: bad commit abc123", LevelPatch},
		{"refactor", "refactor: extract service layer", LevelPatch},

		// Priority: major beats minor/patch indicators.
		{"major beats patch type", "fix!: critical breaking hotfix\n\nBREAKING CHANGE: config format changed", LevelMajor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.message); got != tt.want {
				t.Errorf("Classify(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}
