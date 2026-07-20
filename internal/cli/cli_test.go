package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}

func TestRun_CopyEndToEnd(t *testing.T) {
	// Arrange: a main worktree with a committed manifest and a gitignored
	// .env file, plus a linked worktree (missing .env) to copy into.
	base := t.TempDir()
	mainDir := filepath.Join(base, "main")
	linkedDir := filepath.Join(base, "feature")

	if err := os.Mkdir(mainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, mainDir, "init", "-q")
	if err := os.WriteFile(filepath.Join(mainDir, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, ".wtcopy"), []byte(".env\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, ".env"), []byte("SECRET=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, mainDir, "add", "README.md", ".wtcopy")
	runGit(t, mainDir, "commit", "-q", "-m", "init")
	runGit(t, mainDir, "worktree", "add", "-q", "-b", "feature", linkedDir)

	chdir(t, linkedDir)

	// Act: run the default (copy) subcommand from the linked worktree.
	var stdout, stderr bytes.Buffer
	code := Run(nil, &stdout, &stderr)

	// Assert: it exits cleanly, reports the copy, and .env now exists in the
	// linked worktree with the source content.
	if code != 0 {
		t.Fatalf("Run() = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "copied") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	got, err := os.ReadFile(filepath.Join(linkedDir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "SECRET=1" {
		t.Fatalf(".env content = %q", got)
	}
}

func TestRun_StatusIsDryRun(t *testing.T) {
	// Arrange: a main worktree with a committed manifest and a gitignored
	// .env file, plus a linked worktree (missing .env) to run status from.
	base := t.TempDir()
	mainDir := filepath.Join(base, "main")
	linkedDir := filepath.Join(base, "feature")

	if err := os.Mkdir(mainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, mainDir, "init", "-q")
	if err := os.WriteFile(filepath.Join(mainDir, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, ".wtcopy"), []byte(".env\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, ".env"), []byte("SECRET=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, mainDir, "add", "README.md", ".wtcopy")
	runGit(t, mainDir, "commit", "-q", "-m", "init")
	runGit(t, mainDir, "worktree", "add", "-q", "-b", "feature", linkedDir)

	chdir(t, linkedDir)

	// Act: run the status subcommand from the linked worktree.
	var stdout, stderr bytes.Buffer
	code := Run([]string{"status"}, &stdout, &stderr)

	// Assert: it reports what it would copy, exits cleanly, and leaves the
	// linked worktree untouched.
	if code != 0 {
		t.Fatalf("Run(status) = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "would copy") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(linkedDir, ".env")); !os.IsNotExist(err) {
		t.Fatalf("status should not have created .env, err = %v", err)
	}
}

func TestRun_InitCreatesManifest(t *testing.T) {
	// Arrange: a fresh git repository with no .wtcopy manifest yet.
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	chdir(t, dir)

	// Act: run the init subcommand.
	var stdout, stderr bytes.Buffer
	code := Run([]string{"init"}, &stdout, &stderr)

	// Assert: it exits cleanly and creates a .wtcopy manifest.
	if code != 0 {
		t.Fatalf("Run(init) = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".wtcopy")); err != nil {
		t.Fatalf(".wtcopy not created: %v", err)
	}
}

func TestRun_CopyFromMainWorktreeIsNoop(t *testing.T) {
	// Arrange: a fresh git repository — the current directory is itself the
	// main worktree, so there is nothing to copy from.
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	chdir(t, dir)

	// Act: run the default (copy) subcommand from the main worktree.
	var stdout, stderr bytes.Buffer
	code := Run(nil, &stdout, &stderr)

	// Assert: it exits cleanly and reports that there's nothing to do.
	if code != 0 {
		t.Fatalf("Run() = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "main worktree") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
