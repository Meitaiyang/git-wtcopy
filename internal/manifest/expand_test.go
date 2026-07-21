package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeSourceFile(t *testing.T, root, path string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(path), 0o644); err != nil {
		t.Fatal(err)
	}
}

func entryPaths(entries []Entry) []string {
	paths := make([]string, len(entries))
	for i, entry := range entries {
		paths[i] = entry.Path
	}
	return paths
}

func TestExpand_LiteralPathsPassThrough(t *testing.T) {
	// Arrange: one existing literal path and one missing literal path.
	root := t.TempDir()
	writeSourceFile(t, root, ".env")
	entries := []Entry{{Path: ".env"}, {Path: ".venv"}}

	// Act: expand the entries against the source root.
	got, err := Expand(root, entries)

	// Assert: both literals remain untouched and in their original order.
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	want := []string{".env", ".venv"}
	if paths := entryPaths(got); !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}

func TestExpand_PatternMatchesInStableOrder(t *testing.T) {
	// Arrange: source files whose names all match .env*.
	root := t.TempDir()
	writeSourceFile(t, root, ".env.test")
	writeSourceFile(t, root, ".env")
	writeSourceFile(t, root, ".env.local")

	// Act: expand the glob pattern.
	got, err := Expand(root, []Entry{{Path: ".env*"}})

	// Assert: filepath.Glob's lexical ordering is preserved.
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	want := []string{".env", ".env.local", ".env.test"}
	if paths := entryPaths(got); !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}

func TestExpand_PatternMatchesAcrossDirectories(t *testing.T) {
	// Arrange: package-local environment files in multiple directories.
	root := t.TempDir()
	writeSourceFile(t, root, "packages/b/.env")
	writeSourceFile(t, root, "packages/a/.env")

	// Act: expand a pattern with a wildcard directory segment.
	got, err := Expand(root, []Entry{{Path: "packages/*/.env"}})

	// Assert: each concrete repository-relative path is returned in order.
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	want := []string{"packages/a/.env", "packages/b/.env"}
	if paths := entryPaths(got); !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}

func TestExpand_ZeroMatchesReturnsNoEntries(t *testing.T) {
	// Arrange: an empty source root and a pattern with no possible match.
	root := t.TempDir()

	// Act: expand the unmatched pattern.
	got, err := Expand(root, []Entry{{Path: "*.env"}})

	// Assert: the result is empty and no error is reported.
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("entries = %+v, want none", got)
	}
}

func TestExpand_OverlappingEntriesAreDeduplicated(t *testing.T) {
	// Arrange: overlapping patterns followed by a duplicate literal entry.
	root := t.TempDir()
	writeSourceFile(t, root, ".env")
	writeSourceFile(t, root, ".env.local")
	entries := []Entry{{Path: ".env*"}, {Path: "*.local"}, {Path: ".env"}}

	// Act: expand all entries as one manifest.
	got, err := Expand(root, entries)

	// Assert: each concrete path appears once at its first position.
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	want := []string{".env", ".env.local"}
	if paths := entryPaths(got); !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}

func TestExpand_GitDirectoryIsFiltered(t *testing.T) {
	// Arrange: a source root containing both a .git directory and a dotfile.
	root := t.TempDir()
	writeSourceFile(t, root, ".git/config")
	writeSourceFile(t, root, ".env")

	// Act: expand a pattern that matches every root entry, including dotfiles.
	got, err := Expand(root, []Entry{{Path: "*"}})

	// Assert: .git is removed while other dotfiles remain eligible.
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	want := []string{".env"}
	if paths := entryPaths(got); !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}

func TestExpand_PatternCanMatchDirectory(t *testing.T) {
	// Arrange: a directory tree whose root matches .ven*.
	root := t.TempDir()
	writeSourceFile(t, root, ".venv/lib/pkg.py")

	// Act: expand the directory pattern.
	got, err := Expand(root, []Entry{{Path: ".ven*"}})

	// Assert: the directory itself is returned for recursive copier handling.
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	want := []string{".venv"}
	if paths := entryPaths(got); !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}

func TestExpand_SkipsMatchThroughSymlinkOutsideSourceRoot(t *testing.T) {
	// Arrange: one real package and one package symlinked outside sourceRoot.
	root := t.TempDir()
	outside := t.TempDir()
	writeSourceFile(t, root, "packages/local/.env")
	writeSourceFile(t, outside, ".env")
	if err := os.Symlink(outside, filepath.Join(root, "packages", "external")); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	// Act: expand through the package directory wildcard.
	got, err := Expand(root, []Entry{{Path: "packages/*/.env"}})

	// Assert: the outside match is filtered and the in-root match remains.
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	want := []string{"packages/local/.env"}
	if paths := entryPaths(got); !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}

func TestExpand_LiteralPathThroughExternalSymlinkPassesThrough(t *testing.T) {
	// Arrange: a literal entry whose intermediate directory is a symlink to
	// another filesystem tree.
	root := t.TempDir()
	outside := t.TempDir()
	writeSourceFile(t, outside, ".env")
	if err := os.MkdirAll(filepath.Join(root, "packages"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "packages", "external")); err != nil {
		t.Skipf("create symlink: %v", err)
	}
	entries := []Entry{{Path: "packages/external/.env"}}

	// Act: expand the explicit literal entry.
	got, err := Expand(root, entries)

	// Assert: literal paths preserve their legacy pass-through behavior.
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	want := []string{"packages/external/.env"}
	if paths := entryPaths(got); !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}

func TestExpand_DirectTraversalPatternIsFiltered(t *testing.T) {
	// Arrange: a sibling tree matched only by a traversal pattern that Parse
	// would reject, simulating a direct caller constructing Entry values.
	root := t.TempDir()
	outside := t.TempDir()
	writeSourceFile(t, outside, "outside.env")
	entries := []Entry{{Path: "../*/outside.env"}}

	// Act: expand the unparsed pattern directly.
	got, err := Expand(root, entries)

	// Assert: the lexical root check prevents the outside match from escaping.
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("entries = %+v, want none", got)
	}
}

func TestExpand_InvalidPatternReturnsContextualError(t *testing.T) {
	// Arrange: an entry containing an unterminated character class.
	root := t.TempDir()
	entries := []Entry{{Path: "["}}

	// Act: call Expand directly without parsing the entry first.
	_, err := Expand(root, entries)

	// Assert: filepath's syntax error is retained with the pattern context.
	if !errors.Is(err, filepath.ErrBadPattern) {
		t.Fatalf("err = %v, want filepath.ErrBadPattern", err)
	}
	if !strings.Contains(err.Error(), `expand glob pattern "["`) {
		t.Fatalf("err = %q, want pattern context", err)
	}
}
