// Package commitmsg inspects git commit messages and classifies the kind of
// version change they describe. It understands the conventional-commit style
// used by lazyver:
//
//	major: "feat!: ...", "fix!: ..." (bang right after the type) or a footer
//	       line starting with "BREAKING CHANGE:" / "BREAKING CHANGES:".
//	minor: messages whose first word is one of feat, chore, build, docs,
//	       ci, test, style.
//	patch: messages whose first word is one of fix, perf, revert, refactor.
//
// The type prefix is matched loosely (a simple prefix check), so messages
// such as "feat: x" and "feature: x" are both treated as minor changes.
package commitmsg

import (
	"regexp"
	"strings"
)

// Level describes how strongly a commit message affects the version.
type Level int

const (
	// LevelNone means the message does not bump the version at all.
	LevelNone Level = iota
	// LevelPatch bumps the patch component (x.y.z -> x.y.z+1).
	LevelPatch
	// LevelMinor bumps the minor component and resets the patch (x.y.z -> x.y+1.0).
	LevelMinor
	// LevelMajor bumps the major component and resets minor and patch (x.y.z -> x+1.0.0).
	LevelMajor
)

// breakingChangeFooter matches a footer line such as
// "BREAKING CHANGE: the API changed" anywhere in the message body.
var breakingChangeFooter = regexp.MustCompile(`(?mi)^BREAKING CHANGES?:`)

// bangType matches an optional conventional type with scope immediately
// followed by a bang and a colon, e.g. "feat!:", "fix(scope)!:" or
// "fix(api)!."-style breaking changes. The bang must come right after the
// type/scope, which avoids false positives from exclamation marks elsewhere.
var bangType = regexp.MustCompile(`^[a-zA-Z]+(\([^)]*\))?!:`)

// minorTypes are the commit types that produce a minor version bump.
var minorTypes = []string{"feat", "chore", "build", "docs", "ci", "test", "style"}

// patchTypes are the commit types that produce a patch version bump.
var patchTypes = []string{"fix", "perf", "revert", "refactor"}

// Classify inspects a raw commit message and returns the highest version
// level it describes. Evaluation order matters: major indicators (bang after
// the type or a breaking-change footer) always win over minor and patch
// types, and minor types win over patch types.
//
// Parameters:
//   - message: raw git commit message, including subject, body and footers.
//
// Returns one of LevelMajor, LevelMinor, LevelPatch or LevelNone.
func Classify(message string) Level {
	trimmed := strings.TrimSpace(message)

	// A bang directly after the conventional type means a breaking change.
	if bangType.MatchString(trimmed) {
		return LevelMajor
	}
	// A "BREAKING CHANGE:" / "BREAKING CHANGES:" footer also means major.
	if breakingChangeFooter.MatchString(trimmed) {
		return LevelMajor
	}

	lower := strings.ToLower(trimmed)
	if hasAnyPrefix(lower, minorTypes) {
		return LevelMinor
	}
	if hasAnyPrefix(lower, patchTypes) {
		return LevelPatch
	}
	return LevelNone
}

// hasAnyPrefix reports whether s starts with any of the given prefixes. The
// check is intentionally loose (plain prefix match) so lazy users are not
// punished for writing "feature:" instead of "feat:".
func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
