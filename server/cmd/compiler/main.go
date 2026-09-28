// Command compiler is the content compiler entry point (IMP-003): it compiles
// the 24 launch catalogs in docs/07_content into a CandidateSnapshot, emits
// compile diagnostics in the canonical [file:line:col] [CODE] format, and
// writes a compile report. Exit code is non-zero when any diagnostic exists —
// a content revision with any error never reaches activation.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"thinhthan/internal/config"
)

func main() {
	root := flag.String("root", ".", "repository root containing docs/07_content")
	reportPath := flag.String("report", "", "optional path for the JSON compile report")
	flag.Parse()

	snap, rep, ds := config.Compile(*root)
	for _, d := range ds {
		fmt.Fprintln(os.Stderr, d.String())
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal report:", err)
		os.Exit(2)
	}
	if *reportPath != "" {
		if err := os.WriteFile(*reportPath, data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write report:", err)
			os.Exit(2)
		}
	}
	fmt.Println(string(data))
	if ds.HasErrors() || snap == nil {
		os.Exit(1)
	}
}
