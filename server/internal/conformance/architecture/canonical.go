package architecture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
)

// goConcernOwners maps a §2.6 concern's Go type-name pattern to the paths
// allowed to declare it; an empty owner list means no implementation is
// legal anywhere (the concern is caller-owned, e.g. pooling buffers §1.7).
var goConcernOwners = []struct {
	concern string
	pattern *regexp.Regexp
	owners  []string
}{
	{"logging", regexp.MustCompile(`^(I?Log|.*Logger|Log)$`), []string{"server/internal/observability/", "server/internal/core/"}},
	{"randomness", regexp.MustCompile(`(.*Random.*|.*Rng.*)`), []string{"server/internal/core/rng/"}},
	{"pooling", regexp.MustCompile(`^(Pool.*|.*Pool)$`), nil},
	{"work scheduling", regexp.MustCompile(`(.*Scheduler.*|.*FrameLoop.*)`), nil},
}

// GoCanonicalProblems enforces CODE-006 on the server: a type whose name
// matches a §2.6 concern pattern must live in the concern's owner path, and
// math/rand/v2 may be imported only by core/rng (all server randomness goes
// through the seeded PCG-64 streams).
func GoCanonicalProblems(root string, files []string) []string {
	var problems []string
	for _, rel := range files {
		if !strings.HasSuffix(rel, ".go") || !strings.HasPrefix(rel, "server/") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filepath.Join(root, filepath.FromSlash(rel)), nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if p == "math/rand/v2" && !strings.HasPrefix(rel, "server/internal/core/rng/") {
				problems = append(problems, rel+": math/rand/v2 outside core/rng (randomness has one owner — CODE-006)")
			}
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				for _, c := range goConcernOwners {
					if !c.pattern.MatchString(ts.Name.Name) {
						continue
					}
					if underAny(rel, c.owners) {
						continue
					}
					problems = append(problems, fmt.Sprintf(
						"%s: type %s matches the %s concern outside its owner path (CODE-006)",
						rel, ts.Name.Name, c.concern))
				}
			}
		}
	}
	return dedup(problems)
}

func underAny(rel string, dirs []string) bool {
	for _, d := range dirs {
		if strings.HasPrefix(rel, d) {
			return true
		}
	}
	return false
}
