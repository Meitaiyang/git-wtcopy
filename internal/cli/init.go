package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const manifestTemplate = `# .wtcopy - files/directories to copy from the main worktree into this one.
#
# One repository-root-relative path or glob pattern per line. Supported glob
# metacharacters are *, ?, and [...]. Recursive ** patterns are not supported.
# Lines starting with "#" and blank lines are ignored. Directories are copied
# recursively.
#
# Commit this file to git (do not gitignore it) so it is present in every
# worktree created with "git worktree add".
#
# Examples:
# .env*
# packages/*/.env
# .venv
`

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	force := fs.Bool("force", false, "overwrite an existing manifest file")
	manifestFlag := fs.String("manifest", "", "path to create the manifest file at (default: .wtcopy at the worktree root)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	layout, err := discoverRepo()
	if err != nil {
		fmt.Fprintf(stderr, "git-wtcopy: %v\n", err)
		return 1
	}

	mpath := resolveManifestPath(layout, *manifestFlag)
	if _, err := os.Stat(mpath); err == nil && !*force {
		fmt.Fprintf(stderr, "git-wtcopy: %s already exists (use --force to overwrite)\n", mpath)
		return 1
	}

	if err := os.WriteFile(mpath, []byte(manifestTemplate), 0o644); err != nil {
		fmt.Fprintf(stderr, "git-wtcopy: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "git-wtcopy: created %s\n", mpath)
	fmt.Fprintln(stdout, "Remember to `git add` this file so it is tracked and shows up in every worktree.")
	return 0
}
