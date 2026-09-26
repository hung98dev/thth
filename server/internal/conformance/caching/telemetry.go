package caching

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// Entry is one cached-step telemetry record appended by the workflow helpers
// (.devin/scripts/cache_telemetry.{sh,ps1}) to cache-telemetry.jsonl.
type Entry struct {
	Step        string  `json:"step"`
	Result      string  `json:"result"` // hit | miss
	WallSeconds float64 `json:"wall_seconds"`
}

// ReadTelemetry parses a JSONL telemetry file, strictly. A missing file is
// not an error — a run without caching produces no file.
func ReadTelemetry(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Entry
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		b := bytes.TrimSpace(sc.Bytes())
		if len(b) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(b, &e); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}
		if e.Step == "" {
			return nil, fmt.Errorf("%s:%d: empty step", path, line)
		}
		if e.Result != "hit" && e.Result != "miss" {
			return nil, fmt.Errorf("%s:%d: result must be hit|miss, got %q", path, line, e.Result)
		}
		if e.WallSeconds < 0 {
			return nil, fmt.Errorf("%s:%d: negative wall_seconds", path, line)
		}
		out = append(out, e)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// MergeTelemetryIntoReport sets cached_steps on a verify-report.json without
// disturbing the fixed schema the evidence manifest consumes (CI-003/CI-004).
// The report is rewritten in place. When the telemetry file has no entries,
// cached_steps becomes [] so consumers can rely on the field existing.
func MergeTelemetryIntoReport(reportPath, telemetryPath string) error {
	entries, err := ReadTelemetry(telemetryPath)
	if err != nil {
		return err
	}
	if entries == nil && !fileExists(telemetryPath) {
		return nil // no telemetry produced at all — leave the report untouched
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return err
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("report %s: %w", reportPath, err)
	}
	if entries == nil {
		entries = []Entry{}
	}
	raw, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	obj["cached_steps"] = raw
	out, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(reportPath, append(out, '\n'), 0o644)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
