package worktree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	gitEntryName     = ".git"
	commonDirEntry   = "commondir"
	gitdirLinePrefix = "gitdir:"
)

// Discover walks upward from startDir looking for a ".git" entry, the same
// way git itself locates the repository a command was invoked in, and
// resolves the worktree topology around it.
func Discover(startDir string) (*Layout, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, fmt.Errorf("resolve start directory: %w", err)
	}

	for {
		gitEntry := filepath.Join(dir, gitEntryName)
		info, err := os.Stat(gitEntry)
		if err == nil {
			return resolveLayout(dir, gitEntry, info)
		}
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat %s: %w", gitEntry, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, ErrNotARepository
		}
		dir = parent
	}
}

func resolveLayout(worktreeRoot, gitEntry string, info os.FileInfo) (*Layout, error) {
	if info.IsDir() {
		return &Layout{
			WorktreeRoot:   worktreeRoot,
			GitDir:         gitEntry,
			CommonDir:      gitEntry,
			IsMainWorktree: true,
		}, nil
	}

	adminDir, err := readGitFile(gitEntry)
	if err != nil {
		return nil, err
	}

	commonDirFile := filepath.Join(adminDir, commonDirEntry)
	raw, err := os.ReadFile(commonDirFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotALinkedWorktree
		}
		return nil, fmt.Errorf("read %s: %w", commonDirFile, err)
	}

	commonDir := strings.TrimSpace(string(raw))
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(adminDir, commonDir)
	}
	commonDir = filepath.Clean(commonDir)

	return &Layout{
		WorktreeRoot:   worktreeRoot,
		GitDir:         adminDir,
		CommonDir:      commonDir,
		IsMainWorktree: false,
	}, nil
}

// readGitFile reads a ".git" file (as opposed to directory) and returns the
// git directory path it points at, resolved relative to the directory the
// ".git" file lives in.
func readGitFile(gitFile string) (string, error) {
	raw, err := os.ReadFile(gitFile)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", gitFile, err)
	}

	line := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(line, gitdirLinePrefix) {
		return "", fmt.Errorf("%s: unrecognized .git file format", gitFile)
	}

	target := strings.TrimSpace(strings.TrimPrefix(line, gitdirLinePrefix))
	if target == "" {
		return "", fmt.Errorf("%s: empty gitdir path", gitFile)
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(gitFile), target)
	}
	return filepath.Clean(target), nil
}
