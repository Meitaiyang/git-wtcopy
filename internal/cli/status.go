package cli

import (
	"flag"
	"io"

	"github.com/meitaiyang/git-wtcopy/internal/copier"
)

func runStatus(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	manifestFlag := fs.String("manifest", "", "path to the manifest file (default: .wtcopy at the worktree root)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	mainRoot, dstRoot, entries, code, ok := prepare(stdout, stderr, *manifestFlag)
	if !ok {
		return code
	}

	results := copier.Run(mainRoot, dstRoot, entries, copier.Options{DryRun: true})
	return report(stdout, stderr, results)
}
