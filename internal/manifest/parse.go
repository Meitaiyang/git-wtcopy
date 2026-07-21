package manifest

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// Load reads and parses the manifest file at path.
func Load(filePath string) ([]Entry, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	entries, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filePath, err)
	}
	return entries, nil
}

// Parse reads manifest entries from r. Blank lines and lines starting with
// "#" are ignored. Every other line is a single repository-root-relative
// path or glob pattern; absolute paths and paths containing ".." segments
// are rejected since they could escape the worktree they're being copied into.
func Parse(r io.Reader) ([]Entry, error) {
	var entries []Entry

	scanner := bufio.NewScanner(r)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if err := validatePath(line); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}

		entries = append(entries, Entry{Path: line})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func validatePath(p string) error {
	if path.IsAbs(p) {
		return fmt.Errorf("absolute paths are not allowed: %q", p)
	}
	for _, segment := range strings.Split(p, "/") {
		if segment == ".." {
			return fmt.Errorf("path traversal (\"..\") is not allowed: %q", p)
		}
	}
	if strings.Contains(p, "**") {
		return fmt.Errorf("\"**\" is not supported yet: %q", p)
	}
	if isPattern(p) {
		if _, err := path.Match(p, ""); err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", p, err)
		}
	}
	return nil
}
