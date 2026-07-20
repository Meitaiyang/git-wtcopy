package worktree

import "errors"

// ErrNotARepository is returned when no .git entry can be found by walking
// up from the start directory.
var ErrNotARepository = errors.New("not a git repository (no .git found in this or any parent directory)")

// ErrNotALinkedWorktree is returned when the discovered .git entry is a
// file but does not point at a linked-worktree admin directory (for
// example, it points at a submodule's module directory instead). Linked
// worktree admin directories always contain a "commondir" file; this is
// how git itself tells the two apart, and how we tell them apart too.
var ErrNotALinkedWorktree = errors.New("resolved .git file does not point at a linked worktree (missing commondir)")

// ErrBareCommonDir is returned when the shared repository directory is not
// a conventional ".git" directory (e.g. a bare repository), so there is no
// unambiguous main worktree checkout to copy files from.
var ErrBareCommonDir = errors.New("common git directory is not a standard \".git\" directory; cannot determine a main worktree checkout")
