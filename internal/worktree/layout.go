// Package worktree determines git worktree topology by reading the on-disk
// repository layout directly (the ".git" file/directory, "gitdir" and
// "commondir" admin files) rather than by shelling out to the git CLI or
// linking a git implementation library. This mirrors how git itself
// resolves worktrees internally.
package worktree

import "path/filepath"

// Layout describes the on-disk topology of the repository as seen from a
// single worktree.
type Layout struct {
	// WorktreeRoot is the working directory root of the worktree that was
	// discovered (the directory that directly contains the ".git" entry).
	WorktreeRoot string

	// GitDir is the git directory used by this worktree: the private
	// per-worktree admin directory (e.g. ".git/worktrees/<name>") for a
	// linked worktree, or the real ".git" directory for the main worktree.
	GitDir string

	// CommonDir is the shared repository directory (objects/, refs/,
	// config, ...) used by every worktree of this repository.
	CommonDir string

	// IsMainWorktree reports whether WorktreeRoot is the repository's
	// original checkout, i.e. GitDir is a real ".git" directory rather
	// than a linked worktree's admin directory.
	IsMainWorktree bool
}

// MainWorktreeRoot returns the working directory root of the repository's
// main worktree (the original checkout that linked worktrees were created
// from). It is derived from CommonDir on the assumption that the main
// worktree's git directory is a conventional ".git" directory directly
// inside the main worktree root — true for every repository created the
// normal way.
func (l *Layout) MainWorktreeRoot() (string, error) {
	if l.IsMainWorktree {
		return l.WorktreeRoot, nil
	}
	if filepath.Base(l.CommonDir) != ".git" {
		return "", ErrBareCommonDir
	}
	return filepath.Dir(l.CommonDir), nil
}
