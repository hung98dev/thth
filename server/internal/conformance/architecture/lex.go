package architecture

import "strings"

// csTok is one C# token: an identifier/keyword or a single punctuation char.
// Comments and string literal text never produce tokens; interpolation holes
// are lexed as normal code.
type csTok struct {
	ident bool
	s     string
	line  int
}

// lexer modes for lexCSharp's mode stack.
const (
	mCode   = iota // normal code
	mString        // inside a string literal body
	mHole          // interpolation hole: code until the matching }
)

type lexFrame struct {
	kind     int
	verbatim bool // verbatim string (@"...": "" escape, may span lines)
	interp   bool // interpolated string ($"...": {..} holes)
	depth    int  // hole brace depth
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
func isIdentChar(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// lexCSharp tokenizes C# 9 source, skipping comments and string literal text
// (with escapes) while keeping interpolation-hole code. Char literals,
// verbatim and interpolated strings are handled; raw strings ("""...""") are
// C# 11 and out of the project's language version.
func lexCSharp(src string) []csTok {
	var toks []csTok
	stack := []lexFrame{{kind: mCode}}
	line := 1
	i, n := 0, len(src)
	at := func(j int) byte {
		if j < n {
			return src[j]
		}
		return 0
	}
	push := func(f lexFrame) { stack = append(stack, f) }
	pop := func() {
		if len(stack) > 1 {
			stack = stack[:len(stack)-1]
		}
	}
	emit := func(ident bool, s string) {
		toks = append(toks, csTok{ident: ident, s: s, line: line})
	}
	for i < n {
		top := &stack[len(stack)-1]
		c := src[i]
		switch top.kind {
		case mString:
			switch {
			case top.verbatim && c == '"' && at(i+1) == '"':
				i += 2 // "" escape inside verbatim
				continue
			case !top.verbatim && c == '\\' && at(i+1) != 0:
				i += 2 // \x escape
				continue
			case c == '"':
				pop()
				i++
				continue
			case c == '\n':
				line++
				if !top.verbatim {
					pop() // defensive: non-verbatim strings cannot span lines
				}
				i++
				continue
			case top.interp && c == '{' && at(i+1) == '{':
				i += 2 // {{ escape
				continue
			case top.interp && c == '}' && at(i+1) == '}':
				i += 2 // }} escape
				continue
			case top.interp && c == '{':
				push(lexFrame{kind: mHole})
				i++
				continue
			default:
				i++
				continue
			}
		case mHole:
			if c == '{' {
				top.depth++
				i++
				continue
			}
			if c == '}' {
				if top.depth == 0 {
					pop()
				} else {
					top.depth--
				}
				i++
				continue
			}
			// otherwise fall through to code lexing
		}
		// code context (mCode, or mHole after brace handling)
		switch {
		case c == '\n':
			line++
			i++
			continue
		case c <= ' ':
			i++
			continue
		case isIdentStart(c):
			j := i + 1
			for j < n && isIdentChar(src[j]) {
				j++
			}
			emit(true, src[i:j])
			i = j
			continue
		case c == '/' && at(i+1) == '/':
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		case c == '/' && at(i+1) == '*':
			i += 2
			for i < n && !(src[i] == '*' && at(i+1) == '/') {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			i += 2
			continue
		case c == '\'':
			i++
			for i < n && src[i] != '\'' && src[i] != '\n' {
				if src[i] == '\\' {
					i++
				}
				i++
			}
			i++
			continue
		case c == '@' || c == '$':
			// Possible string prefix: @" $@ $@" @$" must be immediately
			// followed by '"'. Otherwise @ starts a verbatim identifier and
			// $ is just a punct.
			j := i + 1
			verb, interp := c == '@', c == '$'
			if at(j) == '@' || at(j) == '$' {
				if at(j) == '@' {
					verb = true
				} else {
					interp = true
				}
				j++
			}
			if at(j) == '"' {
				push(lexFrame{kind: mString, verbatim: verb, interp: interp})
				i = j + 1
				continue
			}
			if c == '@' && isIdentStart(at(i+1)) {
				k := i + 1
				for k < n && isIdentChar(src[k]) {
					k++
				}
				emit(true, src[i+1:k])
				i = k
				continue
			}
			emit(false, string(c))
			i++
			continue
		case c == '"':
			push(lexFrame{kind: mString})
			i++
			continue
		default:
			emit(false, string(c))
			i++
			continue
		}
	}
	return toks
}

// dottedName returns the qualified name ending at token i by walking a
// `a.b.c` chain to the left, and the index of the chain's first token.
func dottedName(toks []csTok, i int) (string, int) {
	parts := []string{toks[i].s}
	j := i
	for j >= 2 && toks[j-1].s == "." && toks[j-2].ident {
		parts = append([]string{toks[j-2].s}, parts...)
		j -= 2
	}
	return strings.Join(parts, "."), j
}
