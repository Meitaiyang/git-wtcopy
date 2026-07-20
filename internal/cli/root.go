package cli

import (
	"fmt"
	"io"
	"strings"
)

const version = "0.1.0"

const usage = `git-wtcopy copies files that are intentionally gitignored (like .env)
from the repository's main worktree into a linked worktree created with
"git worktree add". It finds the main worktree by reading the on-disk
repository layout directly (the .git file, gitdir and commondir), the
same way git resolves worktrees internally - never by calling the git CLI.

Usage:
  git wtcopy [--force] [--manifest path]        copy manifest entries (default)
  git wtcopy copy [--force] [--manifest path]    same as above, explicit
  git wtcopy status [--manifest path]            preview what would be copied
  git wtcopy init [--force] [--manifest path]    create a starter .wtcopy file
  git wtcopy help
  git wtcopy version
`

// Run dispatches to a subcommand and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return runCopy(args, stdout, stderr)
	}

	switch args[0] {
	case "copy":
		return runCopy(args[1:], stdout, stderr)
	case "status", "list":
		return runStatus(args[1:], stdout, stderr)
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintln(stdout, version)
		return 0
	default:
		if strings.HasPrefix(args[0], "-") {
			return runCopy(args, stdout, stderr)
		}
		fmt.Fprintf(stderr, "git-wtcopy: unknown command %q\n\n", args[0])
		fmt.Fprint(stderr, usage)
		return 2
	}
}
