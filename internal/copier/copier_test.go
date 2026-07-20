package copier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/meitaiyang/git-wtcopy/internal/manifest"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRun_CopiesFile(t *testing.T) {
	// Arrange: a source file and an empty destination.
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, ".env"), "SECRET=1")

	// Act: copy the manifest entry.
	results := Run(src, dst, []manifest.Entry{{Path: ".env"}}, Options{})

	// Assert: the entry is reported copied and the content lands at dst.
	if len(results) != 1 || results[0].Action != ActionCopied || results[0].Err != nil {
		t.Fatalf("results = %+v", results)
	}
	if got := readFile(t, filepath.Join(dst, ".env")); got != "SECRET=1" {
		t.Fatalf("copied content = %q", got)
	}
}

func TestRun_CopiesDirectoryRecursively(t *testing.T) {
	// Arrange: a source directory tree and an empty destination.
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, ".venv", "lib", "pkg.py"), "print(1)")

	// Act: copy the directory manifest entry.
	results := Run(src, dst, []manifest.Entry{{Path: ".venv"}}, Options{})

	// Assert: the entry is reported copied and its contents are recreated
	// under dst.
	if len(results) != 1 || results[0].Action != ActionCopied || results[0].Err != nil {
		t.Fatalf("results = %+v", results)
	}
	if got := readFile(t, filepath.Join(dst, ".venv", "lib", "pkg.py")); got != "print(1)" {
		t.Fatalf("copied content = %q", got)
	}
}

func TestRun_SkipsExistingDestinationByDefault(t *testing.T) {
	// Arrange: source and destination both already have the file, with
	// different content.
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, ".env"), "NEW")
	writeFile(t, filepath.Join(dst, ".env"), "LOCAL")

	// Act: copy without Force.
	results := Run(src, dst, []manifest.Entry{{Path: ".env"}}, Options{})

	// Assert: the entry is reported skipped and the destination is untouched.
	if len(results) != 1 || results[0].Action != ActionSkippedExists {
		t.Fatalf("results = %+v", results)
	}
	if got := readFile(t, filepath.Join(dst, ".env")); got != "LOCAL" {
		t.Fatalf("destination was modified: %q", got)
	}
}

func TestRun_ForceOverwritesExisting(t *testing.T) {
	// Arrange: source and destination both already have the file, with
	// different content.
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, ".env"), "NEW")
	writeFile(t, filepath.Join(dst, ".env"), "LOCAL")

	// Act: copy with Force.
	results := Run(src, dst, []manifest.Entry{{Path: ".env"}}, Options{Force: true})

	// Assert: the entry is reported copied and the destination now has the
	// source content.
	if len(results) != 1 || results[0].Action != ActionCopied {
		t.Fatalf("results = %+v", results)
	}
	if got := readFile(t, filepath.Join(dst, ".env")); got != "NEW" {
		t.Fatalf("destination = %q, want NEW", got)
	}
}

func TestRun_DryRunTouchesNothing(t *testing.T) {
	// Arrange: a source file and an empty destination.
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, ".env"), "NEW")

	// Act: copy with DryRun.
	results := Run(src, dst, []manifest.Entry{{Path: ".env"}}, Options{DryRun: true})

	// Assert: the entry is reported as would-copy but nothing is written.
	if len(results) != 1 || results[0].Action != ActionWouldCopy {
		t.Fatalf("results = %+v", results)
	}
	if _, err := os.Stat(filepath.Join(dst, ".env")); !os.IsNotExist(err) {
		t.Fatalf("dry run created a file, err = %v", err)
	}
}

func TestRun_MissingSource(t *testing.T) {
	// Arrange: neither src nor dst has the manifest entry.
	src := t.TempDir()
	dst := t.TempDir()

	// Act: copy the manifest entry.
	results := Run(src, dst, []manifest.Entry{{Path: ".env"}}, Options{})

	// Assert: the entry is reported as missing source.
	if len(results) != 1 || results[0].Action != ActionMissingSource {
		t.Fatalf("results = %+v", results)
	}
}
