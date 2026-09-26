// cachemerge folds cached-step telemetry (hit|miss + wall_seconds JSONL) into
// a verify-report.json as the cached_steps field (IMP-106, CI-003). Invoked
// by scripts/verify.ps1 after the verifier run; best-effort — never fails the
// report.
package main

import (
	"flag"
	"fmt"
	"os"

	"thinhthan/internal/conformance/caching"
)

func main() {
	report := flag.String("report", "", "path to verify-report.json")
	telemetry := flag.String("telemetry", "", "path to cache-telemetry JSONL")
	flag.Parse()
	if *report == "" || *telemetry == "" {
		fmt.Fprintln(os.Stderr, "usage: cachemerge -report <verify-report.json> -telemetry <cache-telemetry.jsonl>")
		os.Exit(2)
	}
	if err := caching.MergeTelemetryIntoReport(*report, *telemetry); err != nil {
		fmt.Fprintf(os.Stderr, "cachemerge: %v\n", err)
		os.Exit(1)
	}
}
