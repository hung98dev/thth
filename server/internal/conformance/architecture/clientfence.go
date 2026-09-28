package architecture

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// runtimeClientScope is the CODE-005 fence scope: first-party runtime
// assemblies under client/Assets/Scripts/{Core,Net,Systems,UI,App}/ — every
// Scripts/ .cs except Editor/ folders, Tests/ and generated Protocol/
// (engineering_conventions.md §2.5).
func runtimeClientScope(rel string) bool {
	if !strings.HasPrefix(rel, "client/Assets/Scripts/") || !strings.HasSuffix(rel, ".cs") {
		return false
	}
	if strings.Contains(rel, "/Editor/") || strings.Contains(rel, "/Tests/") ||
		strings.HasPrefix(rel, "client/Assets/Scripts/Protocol/") {
		return false
	}
	return true
}

// fenceRuleIDs are the allowlist symbol names of the §2.5 bans; they match
// the rule ids the wired verifier gate uses so one allowlist file serves both.
var fenceRuleIDs = map[string]bool{
	"update_outside_frameloop":   true,
	"coroutines":                 true,
	"linq":                       true,
	"find_calls":                 true,
	"resources_load":             true,
	"async_void":                 true,
	"debug_log":                  true,
	"runtime_material":           true,
	"static_unity_main":          true,
	"task_thread_outside_net":    true,
	"gc_collect":                 true,
	"unity_random":               true,
	"unityevent_fields":          true,
	"static_mutable_outside_app": true,
	"camera_main":                true,
	"unityengine_pool":           true,
}

// LoadAllowlist parses client_api_allowlist.txt: one `path:symbol  reason`
// entry per line (# comments and blank lines skipped). A malformed or
// reasonless entry and an unknown rule symbol are problems (the file is
// protected — a dead entry is a silent missing exemption).
func LoadAllowlist(root string) (map[string]map[string]bool, []string) {
	p := filepath.Join(root, "server", "internal", "conformance", "architecture", "client_api_allowlist.txt")
	data, err := os.ReadFile(p)
	if err != nil {
		return map[string]map[string]bool{}, nil // missing file = empty allowlist
	}
	return ParseAllowlist(string(data))
}

// ParseAllowlist parses allowlist text (testable without a repo tree).
func ParseAllowlist(text string) (map[string]map[string]bool, []string) {
	allow := map[string]map[string]bool{}
	var problems []string
	entryRe := regexp.MustCompile(`^(\S+):(\S+)\s+(\S.*)$`)
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := entryRe.FindStringSubmatch(line)
		if m == nil {
			problems = append(problems, fmt.Sprintf("allowlist line %d malformed (want `path:symbol  reason`)", i+1))
			continue
		}
		if !strings.HasPrefix(m[1], "client/Assets/Scripts/") || !strings.HasSuffix(m[1], ".cs") {
			problems = append(problems, fmt.Sprintf("allowlist line %d path %s is not a client runtime .cs", i+1, m[1]))
		}
		if !fenceRuleIDs[m[2]] {
			problems = append(problems, fmt.Sprintf("allowlist line %d names unknown rule %s", i+1, m[2]))
		}
		if allow[m[1]] == nil {
			allow[m[1]] = map[string]bool{}
		}
		if allow[m[1]][m[2]] {
			problems = append(problems, fmt.Sprintf("allowlist line %d duplicates %s:%s", i+1, m[1], m[2]))
		}
		allow[m[1]][m[2]] = true
	}
	return allow, problems
}

// hit is one banned-API detection: rule id + line number.
type hit struct {
	rule string
	line int
}

// ClientFenceProblems scans committed first-party runtime .cs for the §2.5
// banned APIs (CODE-005); a hit is suppressed only by an exact path:symbol
// allowlist entry.
func ClientFenceProblems(root string, files []string, allow map[string]map[string]bool) []string {
	var problems []string
	for _, rel := range files {
		if !runtimeClientScope(rel) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		for _, h := range scanClientFile(string(data), rel) {
			if allow[rel] != nil && allow[rel][h.rule] {
				continue
			}
			problems = append(problems, fmt.Sprintf("%s:%d banned API %s", rel, h.line, h.rule))
		}
	}
	return dedup(problems)
}

var (
	// unityCallbacks covers every Unity lifecycle callback the wired Q4
	// gate confines to FrameLoop (PERF-020): the §2.5 Update family plus
	// OnEnable/OnDisable/Start/Awake/OnDestroy — kept identical to
	// gates.CheckQ4Client's banned set so one allowlist serves both.
	unityCallbacks = map[string]bool{
		"Update": true, "FixedUpdate": true, "LateUpdate": true, "OnGUI": true,
		"OnEnable": true, "OnDisable": true, "Start": true, "Awake": true,
		"OnDestroy": true,
	}
	findAPIs = map[string]bool{
		"FindObjectOfType": true, "FindObjectsOfType": true,
		"FindFirstObjectByType": true, "FindAnyObjectByType": true,
		"FindObjectsByType": true,
	}
	messagingAPIs = map[string]bool{
		"SendMessage": true, "BroadcastMessage": true, "InvokeRepeating": true,
	}
	coroutineAPIs = map[string]bool{
		"IEnumerator": true, "StartCoroutine": true, "StopCoroutine": true,
		"StopAllCoroutines": true, "Coroutine": true,
	}
	threadAPIs = map[string]bool{
		"Task": true, "Thread": true, "ThreadPool": true,
	}
	// nonTypeKeywords are ident tokens that cannot be a return type, so a
	// preceding one proves the following ident+`(` is not a declaration.
	nonTypeKeywords = map[string]bool{
		"return": true, "if": true, "while": true, "for": true, "foreach": true,
		"switch": true, "case": true, "throw": true, "new": true, "in": true,
		"out": true, "ref": true, "is": true, "as": true, "await": true,
		"using": true, "checked": true, "unchecked": true, "fixed": true,
		"lock": true, "goto": true, "sizeof": true, "typeof": true,
		"nameof": true, "default": true, "else": true, "do": true, "get": true,
		"set": true, "add": true, "remove": true, "var": true, "base": true,
		"this": true, "when": true, "from": true, "where": true,
	}
	// staticSkips are the tokens after `static` that prove the member is not
	// a mutable field.
	staticSkips = map[string]bool{
		"readonly": true, "const": true, "class": true, "struct": true,
		"interface": true, "enum": true, "delegate": true, "operator": true,
		"partial": true, "void": true,
	}
)

// scanClientFile runs every §2.5 rule over one file's token stream.
func scanClientFile(src, rel string) []hit {
	toks := lexCSharp(src)
	var hits []hit
	report := func(rule string, t csTok) { hits = append(hits, hit{rule: rule, line: t.line}) }
	inNet := strings.HasPrefix(rel, "client/Assets/Scripts/Net/")
	inApp := strings.HasPrefix(rel, "client/Assets/Scripts/App/")
	inCameraSvc := strings.HasPrefix(rel, "client/Assets/Scripts/Systems/Camera/")
	isLogFacade := rel == "client/Assets/Scripts/Core/Runtime/Log.cs"

	prev := func(i int) (csTok, bool) {
		if i-1 >= 0 {
			return toks[i-1], true
		}
		return csTok{}, false
	}
	next := func(i int) (csTok, bool) {
		if i+1 < len(toks) {
			return toks[i+1], true
		}
		return csTok{}, false
	}

	for i, t := range toks {
		if !t.ident {
			continue
		}
		name := t.s
		dot, chainStart := dottedName(toks, i)
		p, hasPrev := prev(i)
		n, hasNext := next(i)

		// PERF-020: Unity frame callbacks may be *declared* only in FrameLoop.
		if unityCallbacks[name] && hasNext && n.s == "(" && hasPrev &&
			p.ident && !nonTypeKeywords[p.s] {
			report("update_outside_frameloop", t)
			continue
		}
		// CODE-005 banned API surface.
		switch {
		case coroutineAPIs[name]:
			report("coroutines", t)
		case dot == "System.Linq" || strings.HasPrefix(dot, "System.Linq."):
			report("linq", t)
		case strings.HasPrefix(dot, "GameObject.Find"):
			report("find_calls", t)
		case findAPIs[name] && hasNext && (n.s == "(" || n.s == "<"):
			report("find_calls", t)
		case messagingAPIs[name] && hasNext && (n.s == "(" || n.s == "<"):
			report("find_calls", t)
		case name == "Invoke" && hasNext && n.s == "(" && (!hasPrev || p.s != "."):
			// bare Invoke() is Unity messaging; delegate .Invoke() is legal
			report("find_calls", t)
		case strings.HasPrefix(dot, "Resources.Load"):
			report("resources_load", t)
		case name == "WaitForCompletion" && hasPrev && p.s == "." && hasNext && n.s == "(":
			report("resources_load", t)
		case name == "async" && hasNext && n.ident && n.s == "void":
			report("async_void", t)
		case !isLogFacade && strings.HasPrefix(dot, "Debug.Log"):
			report("debug_log", t)
		case name == "material" && hasPrev && p.s == ".":
			report("runtime_material", t)
		case name == "Material" && hasNext && n.s == "(" && isConstructed(toks, chainStart):
			report("runtime_material", t)
		case name == "Main" && hasNext && n.s == "(" && hasPrev && p.s == "void" &&
			i >= 2 && toks[i-2].s == "static":
			report("static_unity_main", t)
		case !inNet && (strings.HasPrefix(dot, "System.Threading") || threadAPIs[name]):
			report("task_thread_outside_net", t)
		case dot == "GC.Collect" && hasNext && n.s == "(":
			report("gc_collect", t)
		case strings.HasPrefix(dot, "UnityEngine.Random") || strings.HasPrefix(dot, "System.Random"):
			report("unity_random", t)
		case name == "UnityEvent":
			report("unityevent_fields", t)
		case name == "main" && dot == "Camera.main" && !inCameraSvc:
			report("camera_main", t)
		case name == "Pool" && strings.HasPrefix(dot, "UnityEngine.Pool"):
			report("unityengine_pool", t)
		}
		// static mutable fields outside ThinhThan.App
		if !inApp && name == "static" {
			if j := staticFieldNameEnd(toks, i); j > 0 {
				report("static_mutable_outside_app", toks[j])
			}
		}
	}
	return hits
}

// isConstructed reports whether the dotted chain ending the token at
// chainStart is a `new T(` construction (the token before the chain is new).
func isConstructed(toks []csTok, chainStart int) bool {
	return chainStart >= 1 && toks[chainStart-1].ident && toks[chainStart-1].s == "new"
}

// staticFieldNameEnd reports the token index of a field name when the
// `static` at i introduces a mutable field declaration (type + name + `=` or
// `;`), else 0. Methods, properties, types and static readonly/const are not
// mutable fields.
func staticFieldNameEnd(toks []csTok, i int) int {
	j := i + 1
	if j >= len(toks) || !toks[j].ident {
		return 0
	}
	// modifiers between static and the type
	for j < len(toks) && toks[j].ident && toks[j].s != "" {
		if staticSkips[toks[j].s] {
			return 0
		}
		if isModifier(toks[j].s) {
			j++
			continue
		}
		break
	}
	// type name: idents joined by . < > [ ] ? , *
	depth := 0
	for j < len(toks) {
		t := toks[j]
		if t.ident {
			j++
			continue
		}
		switch t.s {
		case ".":
			j++
		case "<":
			depth++
			j++
		case ">":
			depth--
			j++
		case "[", "]", "?", ",", "*":
			j++
		default:
			goto typeDone
		}
	}
typeDone:
	if depth != 0 || j >= len(toks) || !toks[j-1].ident {
		return 0
	}
	// last ident before terminator is the field name; it must be followed by
	// = or ;
	if j-1 < i+1 {
		return 0
	}
	nameIdx := j - 1
	if toks[j].s == "=" || toks[j].s == ";" {
		return nameIdx
	}
	return 0
}

var memberModifiers = map[string]bool{
	"public": true, "private": true, "protected": true, "internal": true,
	"new": true, "sealed": true, "override": true, "abstract": true,
	"virtual": true, "extern": true, "unsafe": true, "volatile": true,
	"event": true, "static": true,
}

func isModifier(s string) bool { return memberModifiers[s] }

// clientConcernOwners maps a §2.6 concern's C# type-name pattern to its
// owner directory (engineering_conventions.md §2.6 client owner column).
var clientConcernOwners = []struct {
	concern string
	pattern *regexp.Regexp
	owner   string
}{
	{"frame driver", regexp.MustCompile(`(FrameLoop|FrameTime|FrameDriver)`), "client/Assets/Scripts/Core/Runtime/"},
	{"work scheduling", regexp.MustCompile(`(FrameBudget|Scheduler)`), "client/Assets/Scripts/Core/Runtime/"},
	{"pooling", regexp.MustCompile(`^(I?Pool|.*Pool)$`), "client/Assets/Scripts/Core/Runtime/"},
	{"logging", regexp.MustCompile(`^(I?Log|.*Logger)$`), "client/Assets/Scripts/Core/Runtime/"},
	{"time", regexp.MustCompile(`(Clock|FrameTime)`), "client/Assets/Scripts/Core/Runtime/"},
	{"randomness", regexp.MustCompile(`(Random|Rng)`), "client/Assets/Scripts/Core/Runtime/"},
	{"camera service", regexp.MustCompile(`(CameraService)`), "client/Assets/Scripts/Systems/Camera/"},
	{"quality governor", regexp.MustCompile(`(QualityGovernor|Governor)`), "client/Assets/Scripts/Core/Performance/"},
}

var typeDeclKw = map[string]bool{
	"class": true, "struct": true, "interface": true, "enum": true, "record": true,
}

// CanonicalImplementationProblems enforces CODE-006 on the client: a second
// type whose name or base type matches a §2.6 concern pattern outside the
// concern's owner path fails Q4.
func CanonicalImplementationProblems(root string, files []string) []string {
	var problems []string
	for _, rel := range files {
		if !runtimeClientScope(rel) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		for _, name := range declaredTypeNames(lexCSharp(string(data))) {
			for _, c := range clientConcernOwners {
				if !c.pattern.MatchString(name) {
					continue
				}
				if strings.HasPrefix(rel, c.owner) {
					continue
				}
				problems = append(problems, fmt.Sprintf(
					"%s: type %s matches the %s concern outside its owner %s (CODE-006)",
					rel, name, c.concern, c.owner))
			}
		}
	}
	return dedup(problems)
}

// declaredTypeNames returns the names declared by class/struct/interface/
// enum/record declarations plus their base-type names (the §2.6 pattern
// applies to the declared name AND its bases).
func declaredTypeNames(toks []csTok) []string {
	var names []string
	for i, t := range toks {
		if !t.ident || !typeDeclKw[t.s] {
			continue
		}
		if i+1 >= len(toks) || !toks[i+1].ident {
			continue
		}
		j := i + 1
		names = append(names, toks[j].s)
		j++
		// skip generic parameter list on the declared name
		if j < len(toks) && toks[j].s == "<" {
			depth := 0
			for j < len(toks) {
				if toks[j].s == "<" {
					depth++
				}
				if toks[j].s == ">" {
					depth--
					if depth == 0 {
						j++
						break
					}
				}
				j++
			}
		}
		// base-type list: `class A : B, IC` — collect idents until the body
		if j < len(toks) && toks[j].s == ":" {
			j++
			for j < len(toks) {
				if toks[j].s == "{" || toks[j].s == ";" || (toks[j].ident && toks[j].s == "where") {
					break
				}
				if toks[j].s == "(" { // positional record params are not bases
					break
				}
				if toks[j].ident {
					names = append(names, toks[j].s)
				}
				j++
			}
		}
	}
	return names
}
