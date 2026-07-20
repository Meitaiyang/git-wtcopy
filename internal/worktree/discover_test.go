package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runGit is a test-only helper. Using the git binary to build realistic
// fixtures is fine — it's the code under test (this package) that must
// never shell out to git.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func newRepoWithWorktree(t *testing.T) (mainDir, linkedDir string) {
	t.Helper()
	base := t.TempDir()
	mainDir = filepath.Join(base, "main")
	linkedDir = filepath.Join(base, "feature")

	if err := os.Mkdir(mainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, mainDir, "init", "-q")
	if err := os.WriteFile(filepath.Join(mainDir, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, mainDir, "add", "README.md")
	runGit(t, mainDir, "commit", "-q", "-m", "init")
	runGit(t, mainDir, "worktree", "add", "-q", "-b", "feature", linkedDir)

	return mainDir, linkedDir
}

func TestDiscover_MainWorktree(t *testing.T) {
	// Arrange: a repository with a linked worktree, discovering from the
	// main worktree's own root.
	mainDir, _ := newRepoWithWorktree(t)

	// Act: discover from the main worktree.
	l, err := Discover(mainDir)

	// Assert: it is identified as the main worktree, and MainWorktreeRoot
	// resolves back to itself.
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !l.IsMainWorktree {
		t.Fatal("expected IsMainWorktree = true")
	}
	if l.WorktreeRoot != mainDir {
		t.Fatalf("WorktreeRoot = %q, want %q", l.WorktreeRoot, mainDir)
	}
	root, err := l.MainWorktreeRoot()
	if err != nil {
		t.Fatalf("MainWorktreeRoot: %v", err)
	}
	if root != mainDir {
		t.Fatalf("MainWorktreeRoot() = %q, want %q", root, mainDir)
	}
}

func TestDiscover_LinkedWorktree(t *testing.T) {
	// Arrange: a repository with a linked worktree, discovering from the
	// linked worktree's root.
	mainDir, linkedDir := newRepoWithWorktree(t)

	// Act: discover from the linked worktree.
	l, err := Discover(linkedDir)

	// Assert: it is identified as a linked (non-main) worktree, and
	// MainWorktreeRoot resolves to the main worktree.
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if l.IsMainWorktree {
		t.Fatal("expected IsMainWorktree = false")
	}
	if l.WorktreeRoot != linkedDir {
		t.Fatalf("WorktreeRoot = %q, want %q", l.WorktreeRoot, linkedDir)
	}
	root, err := l.MainWorktreeRoot()
	if err != nil {
		t.Fatalf("MainWorktreeRoot: %v", err)
	}
	if root != mainDir {
		t.Fatalf("MainWorktreeRoot() = %q, want %q", root, mainDir)
	}
}

func TestDiscover_NestedSubdirectory(t *testing.T) {
	// Arrange: a nested subdirectory inside the linked worktree, with no
	// .git entry of its own.
	_, linkedDir := newRepoWithWorktree(t)
	nested := filepath.Join(linkedDir, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	// Act: discover from the nested subdirectory.
	l, err := Discover(nested)

	// Assert: it walks upward and resolves to the linked worktree's root.
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if l.WorktreeRoot != linkedDir {
		t.Fatalf("WorktreeRoot = %q, want %q", l.WorktreeRoot, linkedDir)
	}
}

func TestDiscover_NotARepository(t *testing.T) {
	// Arrange: an empty directory with no .git entry anywhere above it.
	dir := t.TempDir()

	// Act: discover from that directory.
	_, err := Discover(dir)

	// Assert: it reports ErrNotARepository.
	if err != ErrNotARepository {
		t.Fatalf("err = %v, want ErrNotARepository", err)
	}
}

func TestDiscover_Submodule_NotALinkedWorktree(t *testing.T) {
	// Arrange: a superproject with a submodule checked out — its gitlink
	// points at a module directory with no commondir file, unlike a real
	// linked worktree.
	base := t.TempDir()
	subDir := filepath.Join(base, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, subDir, "init", "-q")
	if err := os.WriteFile(filepath.Join(subDir, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, subDir, "add", "f")
	runGit(t, subDir, "commit", "-q", "-m", "init")

	superDir := filepath.Join(base, "super")
	if err := os.Mkdir(superDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, superDir, "init", "-q")
	runGit(t, superDir, "-c", "protocol.file.allow=always", "submodule", "add", "-q", subDir, "subdir")

	// Act: discover from the submodule's checkout directory.
	_, err := Discover(filepath.Join(superDir, "subdir"))

	// Assert: it reports ErrNotALinkedWorktree, not a false-positive worktree.
	if err != ErrNotALinkedWorktree {
		t.Fatalf("err = %v, want ErrNotALinkedWorktree", err)
	}
}
