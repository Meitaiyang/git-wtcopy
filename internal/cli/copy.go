package cli

import (
	"flag"
	"io"

	"github.com/meitaiyang/git-wtcopy/internal/copier"
)

func runCopy(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("copy", flag.ContinueOnError)
	fs.SetOutput(stderr)
	force := fs.Bool("force", false, "overwrite files/directories that already exist in this worktree")
	manifestFlag := fs.String("manifest", "", "path to the manifest file (default: .wtcopy at the worktree root)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	mainRoot, dstRoot, entries, code, ok := prepare(stdout, stderr, *manifestFlag)
	if !ok {
		return code
	}

	results := copier.Run(mainRoot, dstRoot, entries, copier.Options{Force: *force})
	return report(stdout, stderr, results)
}
