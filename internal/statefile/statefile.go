// Package statefile handles reading and writing the lazyver control file
// (".lazyver.yaml") that lives at the root of the versioned repository.
//
// The file stores the current version, the mode ("kind") that produced it,
// the hash of the last commit already accounted for and a "pending" flag.
// Thanks to lastHash + pending, lazyver never needs to re-read the whole
// repository history after initialization; it only inspects commits created
// after the last processed one.
package statefile

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// FileName is the name of the control file created inside the target
// repository. It is intentionally dot-prefixed to keep the workspace clean.
const FileName = ".lazyver.yaml"

// Kinds are the two supported versioning modes.
const (
	KindSemver = "semver" // Bumps driven by conventional-commit types.
	KindLazy   = "lazy"   // Bumps driven purely by the number of commits.
)

// State is the on-disk representation of lazyver's knowledge about a
// repository.
type State struct {
	Version  string    `yaml:"version"`  // Human-readable version, e.g. "v1.2.3".
	Major    int       `yaml:"major"`    // Major component of the version.
	Minor    int       `yaml:"minor"`    // Minor component of the version.
	Patch    int       `yaml:"patch"`    // Patch component of the version.
	Kind     string    `yaml:"kind"`     // Versioning mode: "semver" or "lazy".
	LastHash string    `yaml:"lastHash"` // Hash of the last commit already counted.
	Pending  bool      `yaml:"pending"`  // True when the hook already bumped for the commit right after LastHash.
	Last     time.Time `yaml:"last"`     // Timestamp of the moment the file was last written.
}

// Load reads the state file from dir. It returns:
//   - (state, nil) when the file exists and parses correctly;
//   - (nil, nil) when the file does not exist yet (fresh repository);
//   - (nil, err) when the file exists but cannot be read or parsed.
func Load(dir string) (*State, error) {
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", FileName, err)
	}
	state := new(State)
	if err := yaml.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("parse %s: %w", FileName, err)
	}
	return state, nil
}

// Save serializes state to disk as ".lazyver.yaml" inside dir, stamping the
// current time into the "last" field. The file is written with permissions
// 0644 so any tooling can read it afterwards.
func Save(dir string, state *State) error {
	state.Last = time.Now()
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", FileName, err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", FileName, err)
	}
	return nil
}

// Remove deletes the state file from dir, ignoring the "not exists" error.
func Remove(dir string) error {
	err := os.Remove(filepath.Join(dir, FileName))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Format renders the current components as the canonical lazyver string,
// e.g. "v1.2.3".
func (s *State) Format() string {
	return fmt.Sprintf("v%d.%d.%d", s.Major, s.Minor, s.Patch)
}
