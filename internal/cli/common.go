// Package cli implements the git-wtcopy command line interface using only
// the standard library (no third-party CLI framework): a small manual
// subcommand dispatcher plus one flag.FlagSet per subcommand.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/meitaiyang/git-wtcopy/internal/copier"
	"github.com/meitaiyang/git-wtcopy/internal/manifest"
	"github.com/meitaiyang/git-wtcopy/internal/worktree"
)

func discoverRepo() (*worktree.Layout, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("determine current directory: %w", err)
	}

	layout, err := worktree.Discover(cwd)
	if err != nil {
		if errors.Is(err, worktree.ErrNotARepository) {
			return nil, fmt.Errorf("not inside a git repository")
		}
		return nil, err
	}
	return layout, nil
}

func resolveManifestPath(layout *worktree.Layout, override string) string {
	if override == "" {
		return filepath.Join(layout.WorktreeRoot, manifest.DefaultFilename)
	}
	if filepath.IsAbs(override) {
		return override
	}
	return filepath.Join(layout.WorktreeRoot, override)
}

// prepare resolves everything copy/status need: the main worktree root to
// copy from, the current worktree root to copy into, and the manifest
// entries. When ok is false, an explanatory message has already been
// written to stdout or stderr and the caller should simply return
// exitCode.
func prepare(stdout, stderr io.Writer, manifestOverride string) (mainRoot, dstRoot string, entries []manifest.Entry, exitCode int, ok bool) {
	layout, err := discoverRepo()
	if err != nil {
		fmt.Fprintf(stderr, "git-wtcopy: %v\n", err)
		return "", "", nil, 1, false
	}
	if layout.IsMainWorktree {
		fmt.Fprintln(stdout, "git-wtcopy: already in the main worktree; nothing to do.")
		return "", "", nil, 0, false
	}

	mainRoot, err = layout.MainWorktreeRoot()
	if err != nil {
		fmt.Fprintf(stderr, "git-wtcopy: %v\n", err)
		return "", "", nil, 1, false
	}

	mpath := resolveManifestPath(layout, manifestOverride)
	entries, err = manifest.Load(mpath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stdout, "git-wtcopy: no manifest found at %s (run `git wtcopy init` to create one).\n", mpath)
			return "", "", nil, 0, false
		}
		fmt.Fprintf(stderr, "git-wtcopy: %v\n", err)
		return "", "", nil, 1, false
	}
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(stdout, "git-wtcopy: manifest is empty; nothing to do.")
		return "", "", nil, 0, false
	}

	entries, err = manifest.Expand(mainRoot, entries)
	if err != nil {
		fmt.Fprintf(stderr, "git-wtcopy: %v\n", err)
		return "", "", nil, 1, false
	}
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(stdout, "git-wtcopy: nothing to copy.")
		return "", "", nil, 0, false
	}

	return mainRoot, layout.WorktreeRoot, entries, 0, true
}

func report(stdout, stderr io.Writer, results []copier.Result) int {
	exit := 0
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(stderr, "  %-9s %s: %v\n", r.Action, r.Entry.Path, r.Err)
			exit = 1
			continue
		}
		fmt.Fprintf(stdout, "  %-9s %s\n", r.Action, r.Entry.Path)
	}
	return exit
}
