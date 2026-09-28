package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// gitFiles returns repo-relative slash paths of committed files.
func gitFiles(root string) ([]string, error) {
	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f != "" {
			files = append(files, filepath.ToSlash(f))
		}
	}
	return files, nil
}

// internalLayer maps a repo-relative path to its architecture layer
// (architecture_conformance.md §2): server/internal/<x>/... -> x,
// server/cmd/<y>/ -> cmd/<y>, server/migrations/ -> migrations.
// "" for paths outside the server module.
func internalLayer(rel string) string {
	rel = path.Clean(rel)
	switch {
	case strings.HasPrefix(rel, "server/internal/"):
		rest := strings.TrimPrefix(rel, "server/internal/")
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			return rest[:i]
		}
		return rest
	case strings.HasPrefix(rel, "server/cmd/"):
		rest := strings.TrimPrefix(rel, "server/cmd/")
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			return "cmd/" + rest[:i]
		}
		return "cmd/" + rest
	case strings.HasPrefix(rel, "server/migrations/"):
		return "migrations"
	case strings.HasPrefix(rel, "server/"):
		return "other"
	}
	return ""
}

// goImports returns the import paths of one Go file.
func goImports(root, rel string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join(root, filepath.FromSlash(rel)), nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, imp := range f.Imports {
		out = append(out, strings.Trim(imp.Path.Value, `"`))
	}
	return out, nil
}

// sqlImport reports whether an import is a SQL-vehicle import — only
// durable/, stackpin/, conformance/, migrations/ and cmd/migrate may use it
// (architecture_conformance.md §2, §4 rule 5).
func sqlImport(imp string) bool {
	return imp == "database/sql" || strings.HasPrefix(imp, "github.com/jackc/pgx")
}

func sqlAllowed(layer string) bool {
	switch layer {
	case "durable", "stackpin", "conformance", "migrations", "cmd/migrate":
		return true
	}
	return false
}

// layerFence reports the reason an internal import is forbidden for the
// importer's layer, or "" (architecture_conformance.md §2 dependency matrix:
// edge -> sim/global/durable; sim -> durable; global -> durable; durable,
// protocol, core, observability are leaves).
func layerFence(layer, imp string) string {
	if !strings.HasPrefix(imp, "thinhthan/internal/") && imp != "thinhthan/migrations" {
		return ""
	}
	seg := strings.TrimPrefix(imp, "thinhthan/internal/")
	if i := strings.IndexByte(seg, '/'); i >= 0 {
		seg = seg[:i]
	}
	isMigrations := imp == "thinhthan/migrations" || strings.HasPrefix(imp, "thinhthan/migrations/")
	switch layer {
	case "sim":
		switch {
		case isMigrations:
			return "sim must not import migrations"
		case seg == "edge" || seg == "global":
			return "sim must not import " + seg
		}
	case "global":
		switch {
		case isMigrations:
			return "global must not import migrations"
		case seg == "edge" || seg == "sim":
			return "global must not import " + seg
		}
	case "durable":
		if seg == "sim" || seg == "edge" || seg == "global" {
			return "durable must not import " + seg
		}
	case "edge":
		if isMigrations {
			return "edge must not import migrations"
		}
	case "protocol":
		if seg != "protocol" && seg != "" {
			return "protocol (generated) imports no domain package"
		}
	case "core":
		if seg != "core" && seg != "" {
			return "core must not import other internal layers"
		}
	case "observability":
		if seg == "sim" || seg == "edge" || seg == "durable" || seg == "global" {
			return "observability must not import " + seg
		}
	}
	return ""
}

// ImportDirectionProblems scans every committed server .go file and reports
// imports that violate the §2 matrix (a forbidden import fails Q4).
func ImportDirectionProblems(root string, files []string) []string {
	var problems []string
	for _, rel := range files {
		if !strings.HasSuffix(rel, ".go") || !strings.HasPrefix(rel, "server/") {
			continue
		}
		layer := internalLayer(path.Dir(rel))
		if layer == "" || layer == "other" {
			continue
		}
		imports, err := goImports(root, rel)
		if err != nil {
			problems = append(problems, rel+": cannot parse imports: "+err.Error())
			continue
		}
		for _, imp := range imports {
			if sqlImport(imp) && !sqlAllowed(layer) {
				problems = append(problems, rel+": "+imp+" is forbidden outside durable/stackpin/conformance/migrate")
				continue
			}
			if why := layerFence(layer, imp); why != "" {
				problems = append(problems, rel+": "+why+" ("+imp+")")
			}
		}
	}
	return dedup(problems)
}

// allowedMains are the only func main files the spec permits: the single
// production binary plus the three pinned tool binaries
// (architecture_conformance.md §4: "chỉ ba tool main được phép thêm là
// compiler/verify/migrate"). The scan scope is server/cmd/ exactly — a
// main.go under an internal package's own cmd/ (e.g. the caching
// cachemerge) is a package-internal helper, not a deployable entry point.
var allowedMains = map[string]bool{
	"server/cmd/server/main.go":   true,
	"server/cmd/compiler/main.go": true,
	"server/cmd/verify/main.go":   true,
	"server/cmd/migrate/main.go":  true,
}

// ProductionMainProblems reports package-main func main files under
// server/cmd/ that are not on the allowlist (one production main — no
// extra server binaries).
func ProductionMainProblems(root string, files []string) []string {
	var problems []string
	for _, rel := range files {
		if !strings.HasPrefix(rel, "server/cmd/") || !strings.HasSuffix(rel, ".go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filepath.Join(root, filepath.FromSlash(rel)), nil, parser.SkipObjectResolution)
		if err != nil {
			continue // parse errors are Q3's problem
		}
		if f.Name.Name != "main" {
			continue
		}
		hasMain := false
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "main" {
				hasMain = true
			}
		}
		if hasMain && !allowedMains[rel] {
			problems = append(problems, rel+": func main outside allowed cmd entry points")
		}
	}
	return dedup(problems)
}

// forbiddenDeps are the banned dependency prefixes (AGENTS.md forbidden list):
// no web frameworks, gRPC, ORMs, third-party loggers, Redis/Kafka/NATS or
// math/rand v1 (use math/rand/v2 or crypto/rand).
var forbiddenDeps = []string{
	"github.com/gin-gonic/",
	"github.com/go-chi/",
	"github.com/labstack/echo",
	"github.com/gofiber/",
	"github.com/gorilla/",
	"google.golang.org/grpc",
	"github.com/grpc-ecosystem/",
	"gorm.io/",
	"github.com/jmoiron/sqlx",
	"go.uber.org/zap",
	"github.com/sirupsen/logrus",
	"github.com/rs/zerolog",
	"github.com/redis/",
	"github.com/go-redis/",
	"github.com/segmentio/kafka-go",
	"github.com/confluentinc/",
	"github.com/nats-io/",
}

// forbiddenDep reports the banned dependency an import/module path belongs
// to, or "".
func forbiddenDep(imp string) string {
	if imp == "math/rand" {
		return "math/rand (use math/rand/v2 or crypto/rand)"
	}
	for _, p := range forbiddenDeps {
		if strings.HasPrefix(imp, p) || imp == strings.TrimSuffix(p, "/") {
			return p
		}
	}
	return ""
}

// ForbiddenDependencyProblems scans every server .go import plus every go.mod
// require line for the banned dependency list.
func ForbiddenDependencyProblems(root string, files []string) []string {
	var problems []string
	for _, rel := range files {
		switch {
		case strings.HasSuffix(rel, ".go") && strings.HasPrefix(rel, "server/"):
			imports, err := goImports(root, rel)
			if err != nil {
				continue
			}
			for _, imp := range imports {
				if why := forbiddenDep(imp); why != "" {
					problems = append(problems, rel+": forbidden dependency "+why+" ("+imp+")")
				}
			}
		case strings.HasSuffix(rel, "go.mod"):
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				continue
			}
			for _, m := range gomodRequireRe.FindAllStringSubmatch(string(data), -1) {
				if why := forbiddenDep(m[1]); why != "" {
					problems = append(problems, rel+": forbidden module "+why+" ("+m[1]+")")
				}
			}
		}
	}
	return dedup(problems)
}

var gomodRequireRe = regexp.MustCompile(`(?m)^\s*(?:require\s+)?\(?\s*([a-z0-9._~/-]+\.[a-z]+[a-z0-9._~/-]*)\s+v[0-9]`)

// SchemaProblems enforces the schema prohibitions: no global_leader_lease
// (global is in-process single-writer, ADR-0052) and no durability column on
// item_instances.
func SchemaProblems(root string, files []string) []string {
	var problems []string
	for _, rel := range files {
		if !strings.HasSuffix(rel, ".sql") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		text := string(data)
		if strings.Contains(text, "global_leader_lease") {
			problems = append(problems, rel+": global_leader_lease is forbidden (ADR-0052)")
		}
		for _, stmt := range strings.Split(text, ";") {
			if strings.Contains(stmt, "item_instances") && durabilityColRe.MatchString(stmt) {
				problems = append(problems, rel+": item_instances carries a durability column")
				break
			}
		}
	}
	return dedup(problems)
}

var durabilityColRe = regexp.MustCompile(`(?i)\bdurability\b`)

// GeneratedBoundaryProblems requires generated dirs to hold only generated
// files (§4 rule 7): server/internal/protocol/** keeps only .pb.go that name
// their proto source; client/Assets/Scripts/Protocol/ keeps .cs with the
// CODE-004 header plus the IMP-000-owned asmdef/csc.rsp/.meta skeleton
// (BLK-007).
func GeneratedBoundaryProblems(root string, files []string) []string {
	var problems []string
	for _, rel := range files {
		switch {
		case strings.HasPrefix(rel, "server/internal/protocol/"):
			if !strings.HasSuffix(rel, ".pb.go") {
				problems = append(problems, rel+": only *.pb.go may live in server/internal/protocol/")
				continue
			}
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				continue
			}
			head := headLines(string(data), 40)
			if !strings.Contains(head, "DO NOT EDIT") || !protoSourceRe.MatchString(head) {
				problems = append(problems, rel+": .pb.go must carry the generated header referencing its proto source")
			}
		case strings.HasPrefix(rel, "client/Assets/Scripts/Protocol/"):
			name := path.Base(rel)
			switch {
			case strings.HasSuffix(name, ".asmdef"), strings.HasSuffix(name, ".meta"), name == "csc.rsp":
			case strings.HasSuffix(name, ".cs"):
				data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
				if err != nil {
					continue
				}
				if !hasGeneratedHeader(string(data)) {
					problems = append(problems, rel+": lacks the CODE-004 generated header — hand-written files are forbidden in generated dirs")
				}
			default:
				problems = append(problems, rel+": unexpected file in generated dir")
			}
		}
	}
	return dedup(problems)
}

var protoSourceRe = regexp.MustCompile(`source:\s+\S+\.proto`)

// hasGeneratedHeader reports whether a .cs file starts with the codegen
// header scripts/codegen.ps1 prepends (CODE-004).
func hasGeneratedHeader(text string) bool {
	head := headLines(text, 10)
	return strings.Contains(head, "#nullable disable") &&
		strings.Contains(head, "#pragma warning disable") &&
		strings.Contains(head, "DO NOT EDIT")
}

func headLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
