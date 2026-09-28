package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Typed cell parsing per content_authoring_contract.md §2.

var (
	idRe      = regexp.MustCompile(`^[a-z0-9_]+(\.[a-z0-9_]+)*$`)
	idTokenRe = regexp.MustCompile(`[a-z0-9_]+(\.[a-z0-9_]+)+`)
	rangeRe   = regexp.MustCompile(`^(-?[0-9]+(\.[0-9]+)?)\.\.(-?[0-9]+(\.[0-9]+)?)$`)
	numUnitRe = regexp.MustCompile(`^(-?[0-9]+(\.[0-9]+)?)(m/s|ms|mps|m|s|px|%)?$`)
	intCommas = regexp.MustCompile(`^-?[0-9][0-9,]*$`)
)

// IsID reports whether s matches the String ID grammar.
func IsID(s string) bool { return idRe.MatchString(s) }

// ExtractIDs returns all backtick-stripped dotted-id tokens in a cell.
func ExtractIDs(s string) []string {
	s = strings.ReplaceAll(s, "`", "")
	return idTokenRe.FindAllString(s, -1)
}

// ParseInt parses an integer cell, tolerating thousands separators in
// display columns ("14,600") which are stripped before parse.
func ParseInt(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	if intCommas.MatchString(s) {
		s = strings.ReplaceAll(s, ",", "")
	}
	return strconv.ParseInt(s, 10, 64)
}

// ParseFloat parses a decimal cell, tolerating thousands separators.
func ParseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	if intCommas.MatchString(s) {
		s = strings.ReplaceAll(s, ",", "")
	}
	return strconv.ParseFloat(s, 64)
}

// ParseRange parses a "min..max" cell into an inclusive range.
func ParseRange(s string) (lo, hi float64, err error) {
	m := rangeRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, 0, fmt.Errorf("not a min..max range: %q", s)
	}
	lo, _ = strconv.ParseFloat(m[1], 64)
	hi, _ = strconv.ParseFloat(m[3], 64)
	if lo > hi {
		return 0, 0, fmt.Errorf("range min > max: %q", s)
	}
	return lo, hi, nil
}

// ParseMagnitude parses "1.9m", "280ms", "8.0s", "350px", "3%" or a bare
// number into its float value; units are validated by the caller context.
func ParseMagnitude(s string) (float64, error) {
	s = strings.TrimSpace(s)
	m := numUnitRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("not a magnitude: %q", s)
	}
	return strconv.ParseFloat(m[1], 64)
}

// SplitList splits a comma-separated cell into trimmed items.
func SplitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(strings.ReplaceAll(p, "`", ""))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// CheckEnum validates s against an allowed set; empty allowed -> skip.
func CheckEnum(ds *Diagnostics, file string, line, col int, colName, s string, allowed map[string]bool) {
	v := strings.TrimSpace(s)
	if v == "" || allowed[v] {
		return
	}
	ds.Add(file, line, col, CodeValueOutOfBounds,
		fmt.Sprintf("%s=%q not in declared enum", colName, v))
}

// CheckEnumRequired is CheckEnum for columns the spec requires non-empty:
// a blank cell is a VALUE_OUT_OF_BOUNDS diagnostic, not a silent skip.
func CheckEnumRequired(ds *Diagnostics, file string, line, col int, colName, s string, allowed map[string]bool) {
	v := strings.TrimSpace(s)
	if v == "" {
		ds.Add(file, line, col, CodeValueOutOfBounds,
			colName+" is required and empty")
		return
	}
	CheckEnum(ds, file, line, col, colName, s, allowed)
}
