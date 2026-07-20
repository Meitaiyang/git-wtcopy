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
