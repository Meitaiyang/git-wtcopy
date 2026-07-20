// Package manifest reads and validates the ".wtcopy" file: a list of
// repository-root-relative paths (files or directories) that should be
// copied from the main worktree into a newly created linked worktree.
package manifest

// DefaultFilename is the manifest file git-wtcopy looks for at the
// repository root when no explicit path is given.
const DefaultFilename = ".wtcopy"

// Entry is a single path listed in a .wtcopy file.
type Entry struct {
	// Path is the repository-root-relative path exactly as written in the
	// manifest, using "/" as the separator regardless of host OS.
	Path string
}
