// Package style machine-checks the engineering_conventions.md §2.7 C# style
// rules: Allman braces, 4-space indent, LF/no-BOM/final-newline, private field
// _camelCase, one top-level type per file, folder-mapped namespaces.
package style

import (
	"regexp"
	"strings"
)

// CheckFile returns the §2.7 violations in one .cs file. rel is the
// repo-relative path (filename==typename); wantNS is the expected namespace
// (empty skips the namespace check).
func CheckFile(rel, text, wantNS string) []string {
	var out []string
	out = append(out, checkLineEndings(text)...)
	out = append(out, checkLintSuppressions(text)...)
	out = append(out, checkWhitespace(text)...)
	out = append(out, checkBraces(text)...)
	out = append(out, checkIndent(text)...)
	out = append(out, checkPrivateFields(text)...)
	out = append(out, checkOneTypePerFile(rel, text)...)
	out = append(out, checkNamespace(text, wantNS)...)
	return out
}

func checkLineEndings(text string) []string {
	var out []string
	if strings.HasPrefix(text, "\ufeff") {
		out = append(out, "BOM present")
	}
	if strings.Contains(text, "\r") {
		out = append(out, "CR/CRLF line endings (must be LF)")
	}
	if !strings.HasSuffix(text, "\n") {
		out = append(out, "missing final newline")
	}
	return out
}

// lintSuppressionRe matches the ways a file silences the compiler/linter —
// CODE-003 forbids all of them (warning-free code, no //lint:file-ignore).
var lintSuppressionRe = regexp.MustCompile(`(?m)(//lint:file-ignore|#pragma\s+warning\s+disable|//\s*ReSharper\s+disable)`)

func checkLintSuppressions(text string) []string {
	if lintSuppressionRe.MatchString(text) {
		return []string{"lint/compiler suppression (//lint:file-ignore, #pragma warning disable) is forbidden"}
	}
	return nil
}

func checkWhitespace(text string) []string {
	var out []string
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if l != strings.TrimRight(l, " \t") {
			out = append(out, n(i)+"trailing whitespace")
		}
		if strings.Contains(l, "\t") {
			out = append(out, n(i)+"tab character (indent is 4 spaces)")
		}
	}
	return out
}

// braceAllowedEnd matches a line starting with '}' that may only contain '}',
// ';', ',', ')' and '/' (comment) characters after it — e.g. '}', '};', '},',
// '});', '}) // comment'.
var braceAllowedEnd = regexp.MustCompile(`^}+\s*[;,)/]*\s*(//.*)?$`)

func checkBraces(text string) []string {
	var out []string
	lines := strings.Split(text, "\n")
	inBlockComment := false
	for i, raw := range lines {
		l := raw
		if inBlockComment {
			if idx := strings.Index(l, "*/"); idx >= 0 {
				l = l[idx+2:]
				inBlockComment = false
			} else {
				continue
			}
		}
		if idx := strings.Index(l, "/*"); idx >= 0 {
			if end := strings.Index(l[idx:], "*/"); end >= 0 {
				l = l[:idx] + l[idx+end+2:]
			} else {
				l = l[:idx]
				inBlockComment = true
			}
		}
		if idx := strings.Index(l, "//"); idx >= 0 {
			l = l[:idx]
		}
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}
		// A line ending in '{' must contain only '{' (Allman); an inline
		// '{' mid-line (e.g. a collection initializer) is fine.
		if strings.HasSuffix(trimmed, "{") && trimmed != "{" {
			out = append(out, n(i)+"'{' must be alone on its line")
		}
		// A line starting with '}' may only have brace-end characters.
		if strings.HasPrefix(trimmed, "}") {
			if !braceAllowedEnd.MatchString(trimmed) {
				out = append(out, n(i)+"line starting '}' may only contain '}' ';' ')' '/'")
			}
		}
	}
	return out
}

func checkIndent(text string) []string {
	var out []string
	for i, l := range strings.Split(text, "\n") {
		if l == "" {
			continue
		}
		spaces := 0
		for _, r := range l {
			if r == ' ' {
				spaces++
			} else {
				break
			}
		}
		if spaces%4 != 0 {
			out = append(out, n(i)+"indent "+itoa(spaces)+" not a multiple of 4")
		}
	}
	return out
}

var privateFieldRe = regexp.MustCompile(`^\s*(?:private|internal)\s+(.+?)\s*;`)
var fieldNameRe = regexp.MustCompile(`^[\w<>\[\],\?\.]+\s+([A-Za-z][A-Za-z0-9_]*)\s*(?:=.*)?$`)

func checkPrivateFields(text string) []string {
	var out []string
	for i, l := range strings.Split(text, "\n") {
		m := privateFieldRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		rest := m[1]
		// const/static (incl. readonly static) fields are exempt.
		fields := strings.Fields(rest)
		isStaticOrConst := false
		for len(fields) > 0 {
			switch fields[0] {
			case "const", "static", "readonly", "volatile", "unsafe", "partial":
				if fields[0] == "const" || fields[0] == "static" {
					isStaticOrConst = true
				}
				fields = fields[1:]
			default:
				goto modifiersDone
			}
		}
	modifiersDone:
		if isStaticOrConst {
			continue
		}
		rest = strings.Join(fields, " ")
		fm := fieldNameRe.FindStringSubmatch(rest)
		if fm == nil {
			continue
		}
		name := fm[1]
		if !strings.HasPrefix(name, "_") || len(name) < 2 || name[1] < 'a' || name[1] > 'z' {
			out = append(out, n(i)+"private instance field '"+name+"' must be _camelCase")
		}
	}
	return out
}

var typeDeclRe = regexp.MustCompile(`^\s*(?:(?:public|internal|private|protected|static|abstract|sealed|partial|readonly|file|unsafe|new)\s+)*(class|struct|interface|enum|record|delegate)\s+([A-Za-z_][A-Za-z0-9_]*)`)

// checkOneTypePerFile counts only top-level (namespace-scope) type
// declarations: a declaration at brace depth ≤ 1 — inside the namespace
// block but outside any type — is top-level; deeper ones are nested types,
// which §2.7 does not forbid.
func checkOneTypePerFile(rel, text string) []string {
	var out []string
	var types []string
	depth := 0
	inBlockComment := false
	for _, raw := range strings.Split(text, "\n") {
		l := raw
		if inBlockComment {
			if idx := strings.Index(l, "*/"); idx >= 0 {
				l = l[idx+2:]
				inBlockComment = false
			} else {
				continue
			}
		}
		if idx := strings.Index(l, "/*"); idx >= 0 {
			if end := strings.Index(l[idx:], "*/"); end >= 0 {
				l = l[:idx] + l[idx+end+2:]
			} else {
				l = l[:idx]
				inBlockComment = true
			}
		}
		if idx := strings.Index(l, "//"); idx >= 0 {
			l = l[:idx]
		}
		if m := typeDeclRe.FindStringSubmatch(l); m != nil && depth <= 1 {
			types = append(types, m[2])
		}
		depth += strings.Count(l, "{") - strings.Count(l, "}")
	}
	if len(types) > 1 {
		out = append(out, "multiple top-level types: "+strings.Join(types, ", "))
	}
	base := rel[strings.LastIndexByte(rel, '/')+1:]
	want := strings.TrimSuffix(base, ".cs")
	if len(types) == 1 && types[0] != want {
		out = append(out, "type "+types[0]+" != filename "+want)
	}
	return out
}

var nsRe = regexp.MustCompile(`(?m)^\s*namespace\s+([A-Za-z_][A-Za-z0-9_.]*)`)

// checkNamespace asserts the declared namespace equals the caller-provided
// expectation (skipped when wantNS is empty).
func checkNamespace(text, wantNS string) []string {
	if wantNS == "" {
		return nil
	}
	m := nsRe.FindStringSubmatch(text)
	if m == nil {
		return []string{"no namespace declared"}
	}
	if m[1] != wantNS {
		return []string{"namespace " + m[1] + " != " + wantNS}
	}
	return nil
}

func n(i int) string {
	// 1-based line number
	return "line " + itoa(i+1) + ": "
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	return string(b[p:])
}
