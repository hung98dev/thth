package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CheckQ4Client — the IMP-083-owned portion of Q4: the client API fence
// (ADR-0059 client runtime bans) and canonical-implementation checks. Runs
// only once IMP-083 is DONE; the wiring lives here so IMP-083 fills in the
// concrete rules.
func CheckQ4Client(root string, e *Env) []Check {
	const gate = "Q4"
	id := func(s string) string { return gate + ".client." + s }
	var checks []Check
	checks = append(checks, checkClientAPIFence(root, id)...)
	checks = append(checks, checkCanonicalClient(root, id)...)
	return checks
}

// bannedClientAPI lists the §2.5 client runtime bans as regexes over a
// file's text. Scope: first-party runtime assemblies
// client/Assets/Scripts/{Core,Net,Systems,UI,App}/** — excluding Editor/
// folders, Tests/ and generated Scripts/Protocol/. Per-ban file exemptions
// mirror the spec table (Net background task, App composition statics).
var bannedClientAPI = []struct {
	name     string
	re       *regexp.Regexp
	exemptIn func(rel string) bool
	capSkip  func(m []string) bool // optional post-filter on capture groups
}{
	{"update_outside_frameloop", regexp.MustCompile(`\b(void|private|public|protected|internal)\s+(Update|FixedUpdate|LateUpdate|OnGUI|OnEnable|OnDisable|Start|Awake|OnDestroy)\s*\(`), frameLoopAllowed, nil},
	{"coroutines", regexp.MustCompile(`\b(IEnumerator|StartCoroutine|StopCoroutine|Coroutine)\b`), nil, nil},
	{"linq", regexp.MustCompile(`\bSystem\.Linq\b|using\s+System\.Linq`), nil, nil},
	{"find_calls", regexp.MustCompile(`\b(GameObject\.Find|FindObjectOfType|FindFirstObjectByType|FindAnyObjectByType|FindObjectsOfType|SendMessage|BroadcastMessage|InvokeRepeating)\s*\(|(?:^|[^\.\w])Invoke\s*\(`), nil, nil},
	{"resources_load", regexp.MustCompile(`\bResources\.Load\b|\.WaitForCompletion\s*\(`), nil, nil},
	{"async_void", regexp.MustCompile(`\basync\s+void\b`), nil, nil},
	{"debug_log", regexp.MustCompile(`\bDebug\.Log\w*\b`), func(rel string) bool {
		// The canonical Log facade (§2.5) is the one file allowed Debug.Log.
		return rel == "client/Assets/Scripts/Core/Runtime/Log.cs"
	}, nil},
	{"runtime_material", regexp.MustCompile(`\.material\b|new\s+Material\(`), nil, nil},
	{"static_unity_main", regexp.MustCompile(`\bstatic\s+void\s+Main\s*\(`), nil, nil},
	{"task_thread_outside_net", regexp.MustCompile(`\b(Task\.Run|new\s+Task\s*<|new\s+Task\s*\(|System\.Threading|ThreadPool|new\s+Thread\s*\()`), func(rel string) bool {
		return strings.Contains(rel, "Assets/Scripts/Net/")
	}, nil},
	{"gc_collect", regexp.MustCompile(`\bGC\.Collect\s*\(`), nil, nil},
	{"unity_random", regexp.MustCompile(`\b(UnityEngine\.Random|System\.Random)\b`), nil, nil},
	{"unityevent_fields", regexp.MustCompile(`\bUnityEvent\b`), nil, nil},
	{"static_mutable_outside_app", regexp.MustCompile(`(?m)^\s*(?:public|private|internal|protected)\s+static\s+([\w<>\[\],\?\.]+)\s+(\w+)\s*[=;]`), func(rel string) bool {
		return strings.Contains(rel, "Assets/Scripts/App/")
	}, func(m []string) bool {
		switch m[1] {
		case "readonly", "const", "class", "struct", "partial":
			return true
		}
		return false
	}},
	{"camera_main", regexp.MustCompile(`\bCamera\.main\b`), nil, nil},
}

// frameLoopAllowed lists files where the banned lifecycle methods are legal
// (the single FrameLoop entrypoint itself).
func frameLoopAllowed(rel string) bool {
	return filepath.Base(rel) == "FrameLoop.cs"
}

// runtimeClientScope is the CODE-005 fence scope: first-party runtime
// assemblies only — Editor/, Tests/ and generated Protocol/ are excluded.
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

// loadAllowlist reads server/internal/conformance/architecture/
// client_api_allowlist.txt (IMP-083, protected). Missing file = empty
// allowlist. Entries are `path:symbol  reason`; a reasonless entry fails.
func loadAllowlist(root string) (map[string]map[string]bool, []string) {
	p := filepath.Join(root, "server", "internal", "conformance", "architecture", "client_api_allowlist.txt")
	data, err := os.ReadFile(p)
	if err != nil {
		return map[string]map[string]bool{}, nil
	}
	allow := map[string]map[string]bool{}
	var problems []string
	lineRe := regexp.MustCompile(`^(\S+):(\S+)\s+(.+)$`)
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := lineRe.FindStringSubmatch(line)
		if m == nil {
			problems = append(problems, fmt.Sprintf("allowlist line %d malformed (want `path:symbol  reason`)", i+1))
			continue
		}
		if allow[m[1]] == nil {
			allow[m[1]] = map[string]bool{}
		}
		allow[m[1]][m[2]] = true
	}
	return allow, problems
}

// checkClientAPIFence scans committed first-party runtime client *.cs for
// banned APIs (CODE-005); allowlist entries carry path:symbol + reason.
func checkClientAPIFence(root string, idf func(string) string) []Check {
	out, err := gitDir(root, "ls-files", "--", "client/**/*.cs")
	if err != nil {
		return []Check{Fail(idf("api_fence"), "git ls-files: "+err.Error())}
	}
	allow, allowProblems := loadAllowlist(root)
	var problems []string
	problems = append(problems, allowProblems...)
	for _, rel := range splitLines(out) {
		if !runtimeClientScope(rel) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		text := string(data)
		for _, b := range bannedClientAPI {
			if b.exemptIn != nil && b.exemptIn(rel) {
				continue
			}
			if allow[rel] != nil && allow[rel][b.name] {
				continue
			}
			if b.capSkip != nil {
				hit := false
				for _, m := range b.re.FindAllStringSubmatch(text, -1) {
					if !b.capSkip(m) {
						hit = true
						break
					}
				}
				if hit {
					problems = append(problems, rel+": banned API "+b.name)
				}
				continue
			}
			if b.re.MatchString(text) {
				problems = append(problems, rel+": banned API "+b.name)
			}
		}
	}
	return []Check{statusCheck(idf("api_fence"), problems)}
}

// checkCanonicalClient asserts the single FrameLoop and Log facades exist and
// are unique once the client code lands (canonical implementations, IMP-083).
func checkCanonicalClient(root string, idf func(string) string) []Check {
	out, err := gitDir(root, "ls-files", "--", "client/**/*.cs")
	if err != nil {
		return []Check{Fail(idf("canonical"), "git ls-files: "+err.Error())}
	}
	var frameLoopFiles, logFiles []string
	for _, rel := range splitLines(out) {
		base := filepath.Base(rel)
		if base == "FrameLoop.cs" {
			frameLoopFiles = append(frameLoopFiles, rel)
		}
		if base == "Log.cs" {
			logFiles = append(logFiles, rel)
		}
	}
	var problems []string
	if len(frameLoopFiles) > 1 {
		problems = append(problems, "multiple FrameLoop.cs: "+strings.Join(frameLoopFiles, ", "))
	}
	if len(logFiles) > 1 {
		problems = append(problems, "multiple Log.cs: "+strings.Join(logFiles, ", "))
	}
	return []Check{statusCheck(idf("canonical"), problems)}
}
