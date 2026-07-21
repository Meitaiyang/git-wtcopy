# git-wtcopy

Copies the files you don't want add into git but need to exist in every worktree, so you don't have to copy them by hand every time.

## The problem

Files like `.env` are gitignored on purpose — they hold local secrets and
shouldn't be committed. Some frameworks require `.env` to exist just to
boot. `git worktree add` checks out only tracked files, so every new
worktree is missing `.env` and won't start until you copy it over by hand.

## The fix

1. List the gitignored files your project needs in a tracked `.wtcopy`
   file at the repository root.
2. After `git worktree add`, run `git wtcopy` inside the new worktree.
3. Everything listed in `.wtcopy` is copied over from the main worktree,
   at the same relative path.

```
$ git worktree add -b feature ../feature
$ cd ../feature
$ git wtcopy
  copied    .env
  copied    .venv
```

## Install

```
go install github.com/meitaiyang/git-wtcopy/cmd/git-wtcopy@latest
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`. Because the binary is
named `git-wtcopy`, git itself picks it up as the subcommand `git wtcopy`
(the same mechanism used by `git-lfs` and any other `git-*` executable).

## Usage

```
git wtcopy init                       # create a starter .wtcopy file
git wtcopy status                     # preview what would be copied
git wtcopy                            # copy (default command)
git wtcopy --force                    # overwrite files that already exist
git wtcopy --manifest path/to/file    # use a manifest at a non-default path
```

Running `git wtcopy` from the main worktree itself is a no-op — there is
nothing to copy from. By default, files that already exist at the
destination are left alone; pass `--force` to overwrite them.

## `.wtcopy` file format

One repository-root-relative path or glob pattern per line. Blank lines and
lines starting with `#` are ignored. A listed or matched directory is copied
recursively.

```
# .wtcopy
.env*
packages/*/.env
.venv
```

Patterns support `*` (any sequence within one path segment), `?` (one
character), and `[...]` (a character class). Recursive `**` patterns are not
supported and are rejected when the manifest is read. Glob patterns that match
nothing are silently ignored. Paths without glob metacharacters retain the
original behavior, including reporting `missing in source` when the path does
not exist in the main worktree.

Unlike many shells, glob matching includes dotfiles. The repository-root `.git`
directory and paths below it are always removed from glob results; other
dotfiles such as `.env` match normally. A literal `.git` entry retains the
existing literal-path behavior.

Glob matches reached through an intermediate directory symlink that resolves
outside the main worktree are also silently removed. These safety filters apply
only to paths discovered by glob expansion. Literal entries are explicit paths
and retain the existing behavior, including when an intermediate symlink points
outside the main worktree.

There is no cross-platform escaping syntax for filenames that literally
contain `[`, `*`, or `?`; such entries are interpreted as patterns. Although
the underlying matcher supports backslash escaping on Unix-like systems,
Windows treats backslash as a path separator.

`.wtcopy` itself must be a **tracked** file (do not gitignore it) so that
it exists in every worktree right after checkout.

## How it finds the main worktree

git-wtcopy never shells out to the `git` CLI and never links a git
implementation library. It determines worktree topology the same way git
itself does internally, by reading the repository's on-disk layout:

1. Walk upward from the current directory looking for a `.git` entry.
2. If `.git` is a directory, the current directory *is* the main worktree
   — there is nothing to copy from.
3. If `.git` is a file, it contains `gitdir: <path>` pointing at the
   worktree's private admin directory
   (`<main>/.git/worktrees/<name>`). Reading that admin directory's
   `commondir` file gives the shared `.git` directory; its parent
   directory is the main worktree root. A directory is only treated as a
   linked worktree's admin directory if it contains a `commondir` file —
   this is what distinguishes a worktree gitlink from, say, a submodule
   gitlink, which points at a module directory with no `commondir`.

See `internal/worktree` for the implementation.

## Architecture

```
cmd/git-wtcopy/       entrypoint — wires internal/cli.Run to os.Args
internal/
  worktree/           on-disk worktree topology detection (no git CLI/lib)
  manifest/           .wtcopy parsing, validation, and read-only glob expansion
  copier/             copies manifest entries between two worktree roots
  cli/                subcommand dispatch and flag parsing (stdlib only)
```

Each `internal` package has a single responsibility and no dependency on
the others' internals: `worktree` and `manifest` are read-only and
side-effect-free beyond the files they explicitly read; `copier` is the
only package that writes to disk; `cli` wires them together and is the
only package that knows about subcommands and flags. There are no
third-party dependencies — `go.sum` is empty.
