package manifest

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Expand resolves every entry against sourceRoot, replacing glob patterns
// with the concrete paths they match. Entries with no glob metacharacters
// pass through untouched, including their existing symlink behavior. Safety
// filtering applies only to paths discovered by glob expansion.
func Expand(sourceRoot string, entries []Entry) ([]Entry, error) {
	expanded := make([]Entry, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	appendPath := func(p string) {
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		expanded = append(expanded, Entry{Path: p})
	}

	for _, entry := range entries {
		if !isPattern(entry.Path) {
			// Literal entries are explicit user choices. Keep their legacy
			// behavior, just as a literal .git entry is kept.
			appendPath(entry.Path)
			continue
		}

		matches, err := expandPattern(sourceRoot, entry.Path)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			appendPath(match)
		}
	}

	return expanded, nil
}

func isPattern(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

func expandPattern(sourceRoot, pattern string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(sourceRoot, filepath.FromSlash(pattern)))
	if err != nil {
		return nil, fmt.Errorf("expand glob pattern %q: %w", pattern, err)
	}
	if len(matches) == 0 {
		return nil, nil
	}

	resolvedRoot, err := filepath.EvalSymlinks(sourceRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve source root for glob pattern %q: %w", pattern, err)
	}

	expanded := make([]string, 0, len(matches))
	for _, match := range matches {
		rel, err := filepath.Rel(sourceRoot, match)
		if err != nil {
			return nil, fmt.Errorf("make glob match %q relative to source root: %w", match, err)
		}
		// Parse normally rejects traversal before Expand is called. Keep this
		// check for callers that construct Entry values directly.
		if pathEscapesRoot(rel) {
			continue
		}

		rel = filepath.ToSlash(rel)
		if rel == ".git" || strings.HasPrefix(rel, ".git/") {
			continue
		}

		resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(match))
		if err != nil {
			return nil, fmt.Errorf("resolve parent of glob match %q: %w", rel, err)
		}
		resolvedRel, err := filepath.Rel(resolvedRoot, resolvedParent)
		if err != nil {
			return nil, fmt.Errorf("check glob match %q against source root: %w", rel, err)
		}
		if pathEscapesRoot(resolvedRel) {
			continue
		}

		expanded = append(expanded, rel)
	}

	return expanded, nil
}

func pathEscapesRoot(rel string) bool {
	return filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
