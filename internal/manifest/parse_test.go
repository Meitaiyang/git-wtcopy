package manifest

import (
	"strings"
	"testing"
)

func TestParse_ValidEntries(t *testing.T) {
	// Arrange: a manifest with comments, blank lines, and leading whitespace
	// mixed in among valid relative-path entries.
	input := `
# top-level comment
.env

  .env.local
.venv
config/local.json
`

	// Act: parse the manifest.
	entries, err := Parse(strings.NewReader(input))

	// Assert: comments and blank lines are dropped, and the remaining paths
	// come back trimmed and in order.
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []string{".env", ".env.local", ".venv", "config/local.json"}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(entries), len(want), entries)
	}
	for i, w := range want {
		if entries[i].Path != w {
			t.Errorf("entries[%d].Path = %q, want %q", i, entries[i].Path, w)
		}
	}
}

func TestParse_RejectsAbsolutePath(t *testing.T) {
	// Arrange: a manifest containing an absolute path.
	input := "/etc/passwd\n"

	// Act: parse the manifest.
	_, err := Parse(strings.NewReader(input))

	// Assert: it is rejected.
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
}

func TestParse_RejectsPathTraversal(t *testing.T) {
	// Arrange: paths that each escape the repository root via a ".." segment.
	cases := []string{"../secret", "a/../../b", "a/..", ".."}

	for _, c := range cases {
		// Act: parse a single-entry manifest containing the case.
		_, err := Parse(strings.NewReader(c + "\n"))

		// Assert: it is rejected.
		if err == nil {
			t.Errorf("path %q: expected error, got nil", c)
		}
	}
}

func TestParse_RejectsDoubleStar(t *testing.T) {
	// Arrange: a manifest containing the reserved recursive-glob syntax.
	input := "packages/**/.env\n"

	// Act: parse the manifest.
	_, err := Parse(strings.NewReader(input))

	// Assert: recursive globbing is rejected with line and pattern context.
	if err == nil {
		t.Fatal("expected error for double-star pattern")
	}
	want := `line 1: "**" is not supported yet: "packages/**/.env"`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("err = %q, want it to contain %q", err, want)
	}
}

func TestParse_RejectsInvalidGlobPattern(t *testing.T) {
	// Arrange: a manifest containing an unterminated character class.
	input := "packages/[abc/.env\n"

	// Act: parse the manifest.
	_, err := Parse(strings.NewReader(input))

	// Assert: malformed syntax is rejected during parsing with line context.
	if err == nil {
		t.Fatal("expected error for invalid glob pattern")
	}
	want := `line 1: invalid glob pattern "packages/[abc/.env"`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("err = %q, want it to contain %q", err, want)
	}
}

func TestParse_AcceptsGlobPattern(t *testing.T) {
	// Arrange: a manifest containing a supported glob pattern.
	input := ".env*\n"

	// Act: parse the manifest.
	entries, err := Parse(strings.NewReader(input))

	// Assert: the pattern is retained for Expand to resolve later.
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(entries) != 1 || entries[0].Path != ".env*" {
		t.Fatalf("entries = %+v, want one .env* pattern", entries)
	}
}
