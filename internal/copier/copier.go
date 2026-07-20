// Package copier copies the files and directories listed in a manifest
// from one worktree root to another.
package copier

import (
	"os"
	"path/filepath"

	"github.com/meitaiyang/git-wtcopy/internal/manifest"
)

// Action describes what happened (or would happen, in a dry run) to a
// single manifest entry.
type Action int

const (
	// ActionCopied means the entry was copied to the destination.
	ActionCopied Action = iota
	// ActionWouldCopy means the entry would be copied (dry run only).
	ActionWouldCopy
	// ActionSkippedExists means the destination already existed and Force
	// was not set.
	ActionSkippedExists
	// ActionMissingSource means the entry did not exist under the source
	// root, so there was nothing to copy.
	ActionMissingSource
	// ActionError means an unexpected filesystem error occurred; see
	// Result.Err for details.
	ActionError
)

func (a Action) String() string {
	switch a {
	case ActionCopied:
		return "copied"
	case ActionWouldCopy:
		return "would copy"
	case ActionSkippedExists:
		return "skipped (already exists)"
	case ActionMissingSource:
		return "missing in source"
	case ActionError:
		return "error"
	default:
		return "unknown"
	}
}

// Result is the outcome of processing a single manifest entry.
type Result struct {
	Entry  manifest.Entry
	Action Action
	Err    error
}

// Options controls how Run treats existing destination files.
type Options struct {
	// Force overwrites files/directories that already exist at the
	// destination. Without it, existing destinations are left untouched.
	Force bool
	// DryRun reports what would happen without touching the filesystem.
	DryRun bool
}

// Run copies every entry from sourceRoot to destRoot, honoring Options, and
// reports what happened to each entry. It never returns an error itself;
// per-entry failures are reported in each Result so that one bad entry
// doesn't abort the rest of the batch.
func Run(sourceRoot, destRoot string, entries []manifest.Entry, opts Options) []Result {
	results := make([]Result, 0, len(entries))
	for _, e := range entries {
		results = append(results, runOne(sourceRoot, destRoot, e, opts))
	}
	return results
}

func runOne(sourceRoot, destRoot string, e manifest.Entry, opts Options) Result {
	src := filepath.Join(sourceRoot, filepath.FromSlash(e.Path))
	dst := filepath.Join(destRoot, filepath.FromSlash(e.Path))

	srcInfo, err := os.Lstat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{Entry: e, Action: ActionMissingSource}
		}
		return Result{Entry: e, Action: ActionError, Err: err}
	}

	if _, err := os.Lstat(dst); err == nil {
		if !opts.Force {
			return Result{Entry: e, Action: ActionSkippedExists}
		}
	} else if !os.IsNotExist(err) {
		return Result{Entry: e, Action: ActionError, Err: err}
	}

	if opts.DryRun {
		return Result{Entry: e, Action: ActionWouldCopy}
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return Result{Entry: e, Action: ActionError, Err: err}
	}

	if opts.Force {
		if err := os.RemoveAll(dst); err != nil {
			return Result{Entry: e, Action: ActionError, Err: err}
		}
	}

	if srcInfo.IsDir() {
		err = copyDir(src, dst)
	} else {
		err = copyFile(src, dst, srcInfo)
	}
	if err != nil {
		return Result{Entry: e, Action: ActionError, Err: err}
	}
	return Result{Entry: e, Action: ActionCopied}
}
