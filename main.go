package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/audryus/lazyver/cmd/lazyver"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func main() {
	lazyver.Execute()
	r, err := git.PlainOpen("D:\\projs\\2dpoint.site")
	CheckIfError(err)

	ref, err := r.Head()
	CheckIfError(err)

	cIter, err := r.Log(&git.LogOptions{Order: git.LogOrderCommitterTime, From: ref.Hash()})
	CheckIfError(err)

	messages := make([]string, 0)

	var cCount int
	err = cIter.ForEach(func(c *object.Commit) error {
		cCount++
		messages = append(messages, c.Message)
		return nil
	})

	CheckIfError(err)
	slices.Reverse[[]string](messages)
	major := 0
	minor := 0
	patch := 0

	chars := strings.Split(fmt.Sprintf("%03d", cCount), "")
	major, _ = strconv.Atoi(chars[0])
	minor, _ = strconv.Atoi(chars[1])
	patch, _ = strconv.Atoi(chars[2])
	//fmt.Println("major:", major) // 3
	//fmt.Println("minor:", minor) // 3
	//fmt.Println("patch:", patch) // 9

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

	//fmt.Println("major:", major) // 3
	//fmt.Println("minor:", minor) // 3
	//fmt.Println("patch:", patch) // 9

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
