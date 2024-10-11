package lazy

import (
	"fmt"
	"strconv"
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

	//ref, err := r.Head()
	CheckIfError(err)

	//hash := ref.Hash()
	cCount := 0

	v, err := yaml.Read(path)
	opts := &git.LogOptions{Order: git.LogOrderCommitterTime}

	if v != nil {
		if v.Kind != "lazy" {
			panic(fmt.Sprintf("invalid kind: %s", v.Kind))
		}

		major = v.Major
		minor = v.Minor
		patch = v.Patch

		cCount += major * 100
		cCount += minor * 10
		cCount += patch

		opts.Since = &v.Last
	}

	cIter, err := r.Log(opts)
	CheckIfError(err)

	last := time.Now()
	i := 0
	err = cIter.ForEach(func(c *object.Commit) error {
		cCount++
		if i == 0 {
			last = c.Author.When
			i++
		}
		return nil
	})

	CheckIfError(err)

	chars := strings.Split(fmt.Sprintf("%03d", cCount), "")
	major, _ = strconv.Atoi(chars[0])
	minor, _ = strconv.Atoi(chars[1])
	patch, _ = strconv.Atoi(chars[2])

	return yaml.Write(path, major, minor, patch, last.Add(10*time.Millisecond), "lazy")
}

func CheckIfError(err error) {
	if err != nil {
		panic(err)
	}
}
