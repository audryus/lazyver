// Command lazyver is the executable entry point of the lazyver CLI, a tool
// that generates and maintains a project version file from git commit
// history, with automatic per-commit bumps through a git hook.
package main

import "github.com/audryus/lazyver/cmd/lazyver"

func main() {
	lazyver.Execute()
}
