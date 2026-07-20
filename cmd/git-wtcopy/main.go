// Command git-wtcopy copies gitignored files (like .env) from a
// repository's main worktree into a linked worktree, so that files
// required to run the project don't have to be recreated by hand every
// time "git worktree add" is used.
package main

import (
	"os"

	"github.com/meitaiyang/git-wtcopy/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
