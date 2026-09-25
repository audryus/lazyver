// Package calculator implements the two versioning strategies supported by
// lazyver. Both strategies mutate a statefile.State in place based on new
// commits that have not been accounted for yet.
//
// Semver strategy: bumps are driven by conventional-commit message types.
// Lazy strategy: the total number of commits is spread across the three
// version digits (hundreds -> major, tens -> minor, units -> patch), which
// is intentionally simple — the mode is called "lazy" for a reason.
package calculator

import (
	"fmt"
	"strings"

	"github.com/audryus/lazyver/internal/commitmsg"
	"github.com/audryus/lazyver/internal/statefile"
)

// ApplySemVer updates state by replaying messages (oldest first) through the
// semver rules. Every major message resets minor and patch; every minor
// message resets patch; every patch message increments the patch only.
//
// Parameters:
//   - state:    current persisted state; mutated in place.
//   - messages: commit messages not yet accounted for, oldest first.
func ApplySemVer(state *statefile.State, messages []string) {
	for _, message := range messages {
		switch commitmsg.Classify(message) {
		case commitmsg.LevelMajor:
			state.Major++
			state.Minor = 0
			state.Patch = 0
		case commitmsg.LevelMinor:
			state.Minor++
			state.Patch = 0
		case commitmsg.LevelPatch:
			state.Patch++
		case commitmsg.LevelNone:
			// Messages with no recognizable type do not bump anything.
		}
	}
	state.Version = state.Format()
}

// ApplySemVerSingle applies exactly one semver bump to state, derived from a
// single message. It is used by the commit-msg hook, where only the message
// being committed is available.
//
// Returns true when the message produced an actual bump, false otherwise.
func ApplySemVerSingle(state *statefile.State, message string) bool {
	level := commitmsg.Classify(message)
	switch level {
	case commitmsg.LevelMajor:
		state.Major++
		state.Minor = 0
		state.Patch = 0
	case commitmsg.LevelMinor:
		state.Minor++
		state.Patch = 0
	case commitmsg.LevelPatch:
		state.Patch++
	case commitmsg.LevelNone:
		return false
	}
	state.Version = state.Format()
	return true
}

// ApplyLazy updates state using the lazy counting scheme: the total number
// of commits is encoded across the three digits of the version. The stored
// major/minor/patch are first converted back into a flat count, incremented
// by newCount, and then split again digit-wise.
//
// Examples (with zero previous commits):
//
//	  5 commits -> v0.0.5
//	 42 commits -> v0.4.2
//	339 commits -> v3.3.9
//
// Parameters:
//   - state:    current persisted state; mutated in place.
//   - newCount: how many commits have happened since the last run.
func ApplyLazy(state *statefile.State, newCount int) {
	total := state.Major*100 + state.Minor*10 + state.Patch + newCount
	state.Patch = total % 10
	state.Minor = (total / 10) % 10
	state.Major = total / 100
	state.Version = state.Format()
}

// NormalizeKind validates that kind is one of the supported modes.
// An empty kind defaults to semver. Unknown kinds return an error listing
// the valid options.
func NormalizeKind(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", statefile.KindSemver:
		return statefile.KindSemver, nil
	case statefile.KindLazy:
		return statefile.KindLazy, nil
	default:
		return "", fmt.Errorf("unknown kind %q: expected %q or %q",
			kind, statefile.KindSemver, statefile.KindLazy)
	}
}
