package sem

import (
	"fmt"
	"slices"
	"strings"
	"time"

	yaml "github.com/audryus/lazyver/internal/ywriter"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func Run(path string) yaml.Version {
	r, err := git.PlainOpen(path)
	CheckIfError(err)
	major := 0
	minor := 0
	patch := 0

	v, err := yaml.Read(path)
	opts := &git.LogOptions{Order: git.LogOrderCommitterTime}

	if v != nil {
		if v.Kind != "sem" {
			panic(fmt.Sprintf("invalid kind: %s", v.Kind))
		}

		major = v.Major
		minor = v.Minor
		patch = v.Patch
		opts.Since = &v.Last
	}

	cIter, err := r.Log(opts)
	CheckIfError(err)

	messages := make([]string, 0)

	last := time.Now()
	i := 0
	err = cIter.ForEach(func(c *object.Commit) error {
		messages = append(messages, c.Message)
		if i == 0 {
			last = c.Author.When
			i++
		}
		return nil
	})

	CheckIfError(err)
	slices.Reverse[[]string](messages)

	for _, message := range messages {
		if isMajor(message) {
			major++
			minor = 0
			patch = 0
		}
		if isMinor(message) {
			minor++
			patch = 0
		}
		if isPatch(message) {
			patch++
		}
	}

	return yaml.Write(path, major, minor, patch, last.Add(10*time.Millisecond), "sem")
}

func CheckIfError(err error) {
	if err != nil {
		panic(err)
	}
}

var majorKeys = []string{"BREAKING CHANGE", "BREAKING CHANGES", "!"}
var minorKeys = []string{"feat", "chore", "build", "docs", "ci", "test", "style"}
var patchKeys = []string{"fix", "perf", "revert", "refactor"}

func isMajor(message string) bool {
	for _, key := range majorKeys {
		if strings.Contains(message, key) {
			return true
		}
	}
	return false
}

func isMinor(message string) bool {
	for _, key := range minorKeys {
		if strings.HasPrefix(message, key) {
			return true
		}
	}
	return false
}
func isPatch(message string) bool {
	for _, key := range patchKeys {
		if strings.HasPrefix(message, key) {
			return true
		}
	}
	return false
}
