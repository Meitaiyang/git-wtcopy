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

func newLinkedWorktreeWithManifest(t *testing.T, manifestContent string) (mainDir, linkedDir string) {
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
	if err := os.WriteFile(filepath.Join(mainDir, ".wtcopy"), []byte(manifestContent), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, mainDir, "add", "README.md", ".wtcopy")
	runGit(t, mainDir, "commit", "-q", "-m", "init")
	runGit(t, mainDir, "worktree", "add", "-q", "-b", "feature", linkedDir)
	return mainDir, linkedDir
}

func TestRun_CopyEndToEnd(t *testing.T) {
	// Arrange: a main worktree with a committed manifest and a gitignored
	// .env file, plus a linked worktree (missing .env) to copy into.
	mainDir, linkedDir := newLinkedWorktreeWithManifest(t, ".env\n")
	if err := os.WriteFile(filepath.Join(mainDir, ".env"), []byte("SECRET=1"), 0o644); err != nil {
		t.Fatal(err)
	}

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

func TestRun_CopyGlobEndToEnd(t *testing.T) {
	// Arrange: a main worktree with two files matched by a committed .env*
	// manifest entry, plus a linked worktree missing both files.
	mainDir, linkedDir := newLinkedWorktreeWithManifest(t, ".env*\n")
	if err := os.WriteFile(filepath.Join(mainDir, ".env"), []byte("BASE=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, ".env.local"), []byte("LOCAL=1"), 0o644); err != nil {
		t.Fatal(err)
	}

	chdir(t, linkedDir)

	// Act: run the default copy command from the linked worktree.
	var stdout, stderr bytes.Buffer
	code := Run(nil, &stdout, &stderr)

	// Assert: both matched files are copied with their source contents.
	if code != 0 {
		t.Fatalf("Run() = %d, stderr = %s", code, stderr.String())
	}
	for path, want := range map[string]string{
		".env":       "BASE=1",
		".env.local": "LOCAL=1",
	} {
		got, err := os.ReadFile(filepath.Join(linkedDir, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(got) != want {
			t.Fatalf("%s content = %q, want %q", path, got, want)
		}
		if !strings.Contains(stdout.String(), path) {
			t.Fatalf("stdout = %q, want copied path %s", stdout.String(), path)
		}
	}
}

func TestRun_StatusIsDryRun(t *testing.T) {
	// Arrange: a main worktree with a committed manifest and a gitignored
	// .env file, plus a linked worktree (missing .env) to run status from.
	mainDir, linkedDir := newLinkedWorktreeWithManifest(t, ".env\n")
	if err := os.WriteFile(filepath.Join(mainDir, ".env"), []byte("SECRET=1"), 0o644); err != nil {
		t.Fatal(err)
	}

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

func TestRun_EmptyAndUnmatchedManifests(t *testing.T) {
	cases := []struct {
		name            string
		args            []string
		manifestContent string
		wantStdout      string
	}{
		{
			name:            "copy empty manifest",
			manifestContent: "# no entries\n",
			wantStdout:      "git-wtcopy: manifest is empty; nothing to do.\n",
		},
		{
			name:            "status empty manifest",
			args:            []string{"status"},
			manifestContent: "# no entries\n",
			wantStdout:      "git-wtcopy: manifest is empty; nothing to do.\n",
		},
		{
			name:            "copy unmatched pattern",
			manifestContent: "missing/*.env\n",
			wantStdout:      "git-wtcopy: nothing to copy.\n",
		},
		{
			name:            "status unmatched pattern",
			args:            []string{"status"},
			manifestContent: "missing/*.env\n",
			wantStdout:      "git-wtcopy: nothing to copy.\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange: a linked worktree whose manifest is empty or matches nothing.
			_, linkedDir := newLinkedWorktreeWithManifest(t, tc.manifestContent)
			chdir(t, linkedDir)

			// Act: run copy or status from the linked worktree.
			var stdout, stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)

			// Assert: the shared prepare path reports the precise no-op reason.
			if code != 0 {
				t.Fatalf("Run(%v) = %d, stderr = %s", tc.args, code, stderr.String())
			}
			if stdout.String() != tc.wantStdout {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tc.wantStdout)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
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

	// Assert: it exits cleanly and creates a manifest documenting glob usage.
	if code != 0 {
		t.Fatalf("Run(init) = %d, stderr = %s", code, stderr.String())
	}
	content, err := os.ReadFile(filepath.Join(dir, ".wtcopy"))
	if err != nil {
		t.Fatalf("read .wtcopy: %v", err)
	}
	for _, want := range []string{".env*", "packages/*/.env", "Recursive ** patterns are not supported"} {
		if !strings.Contains(string(content), want) {
			t.Fatalf(".wtcopy = %q, want it to contain %q", content, want)
		}
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
