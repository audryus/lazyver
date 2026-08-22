// Package gitrepo provides a thin wrapper around the user's locally installed
// git binary. It intentionally avoids any third-party git library (such as
// go-git) so that lazyver behaves exactly like the git the user already has
// configured, including hooks, credentials and repository state.
package gitrepo

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Commit represents a single git commit with its full hash and raw commit
// message (subject plus body plus footers).
type Commit struct {
	Hash    string // Full 40-character SHA-1 (or SHA-256) hash of the commit.
	Message string // Raw commit message, including body and footers.
}

// recordSeparator and unitSeparator are control characters used to build a
// machine-parseable `git log` output. They never appear inside commit
// messages, which makes the parsing safe and unambiguous.
const (
	recordSeparator = "\x1e"
	unitSeparator   = "\x1f"
)

// Run executes the git binary with the given arguments, using dir as the
// working directory (equivalent to `git -C dir`). It returns the trimmed
// standard output of the command.
//
// Parameters:
//   - dir:  directory in which the git command is executed. It must exist.
//   - args: arguments passed verbatim to the git binary (e.g. "log", "HEAD").
//
// Returns the trimmed stdout on success, or an error describing the failed
// command together with git's standard error output.
func Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// IsRepository reports whether dir (or any of its parents) is inside a git
// working tree. It relies on `git rev-parse --is-inside-work-tree`, so bare
// repositories are not considered valid targets for lazyver.
func IsRepository(dir string) bool {
	out, err := Run(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// HeadHash returns the full hash of the current HEAD commit.
//
// It returns an error when HEAD cannot be resolved, which happens for
// instance on a repository that has no commits yet (unborn branch).
func HeadHash(dir string) (string, error) {
	return Run(dir, "rev-parse", "HEAD")
}

// CommitsBetween returns every commit reachable from HEAD, in chronological
// order (oldest first), excluding the commit identified by sinceHash and all
// of its ancestors.
//
// Parameters:
//   - dir:       repository working directory.
//   - sinceHash: exclusive lower bound. When empty, the entire history is
//     returned — this is the behaviour used when initializing lazyver in a
//     repository for the first time.
//
// Commits with no ancestors (empty repository) yield a nil slice without
// error. Any other git failure is returned verbatim.
func CommitsBetween(dir, sinceHash string) ([]Commit, error) {
	args := []string{
		"log",
		"--reverse",                              // oldest commit first, so bumps apply in order.
		"--pretty=format:%H" + unitSeparator + "%B" + recordSeparator, // hash + full message, record separated.
	}
	if sinceHash != "" {
		args = append(args, sinceHash+"..HEAD")
	}

	out, err := Run(dir, args...)
	if err != nil {
		// An unborn branch (repository without any commit) makes `git log`
		// fail; that is not an error for lazyver, just an empty history.
		if strings.Contains(err.Error(), "does not have any commits yet") ||
			strings.Contains(err.Error(), "unknown revision") {
			return nil, nil
		}
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	records := strings.Split(out, recordSeparator)
	commits := make([]Commit, 0, len(records))
	for _, record := range records {
		record = strings.Trim(record, "\n")
		if record == "" {
			continue
		}
		parts := strings.SplitN(record, unitSeparator, 2)
		if len(parts) != 2 {
			continue
		}
		commits = append(commits, Commit{
			Hash:    strings.TrimSpace(parts[0]),
			Message: strings.TrimSpace(parts[1]),
		})
	}
	return commits, nil
}

// StageFile adds the given file to the git index, forcing the addition even
// when the file is matched by .gitignore. This is required because the
// version file must travel inside the commit that triggered it, regardless
// of the project's ignore rules.
func StageFile(dir, file string) error {
	_, err := Run(dir, "add", "-f", file)
	return err
}
