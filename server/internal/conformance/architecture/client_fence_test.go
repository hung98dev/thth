package architecture

import (
	"strings"
	"testing"
)

// ---------- fixtures ------------------------------------------------------

// csFile is a relative client path plus source text.
func fixtureTree(t *testing.T, files map[string]string) (string, []string) {
	t.Helper()
	root := t.TempDir()
	var list []string
	for rel, src := range files {
		writeFile(t, root, rel, src)
		list = append(list, rel)
	}
	return root, list
}

func realClientFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	files, err := repoFiles(root)
	if err != nil {
		t.Fatalf("repo files: %v", err)
	}
	var out []string
	for _, f := range files {
		if runtimeClientScope(f) {
			out = append(out, f)
		}
	}
	return out
}

func realAllowlist(t *testing.T) map[string]map[string]bool {
	t.Helper()
	allow, problems := LoadAllowlist(repoRoot(t))
	if len(problems) != 0 {
		t.Fatalf("real allowlist malformed: %v", problems)
	}
	return allow
}

// ---------- tests ----------------------------------------------------------

func TestClientApiFence(t *testing.T) {
	// Real tree must be clean under the fence + shipped allowlist.
	root := repoRoot(t)
	if got := ClientFenceProblems(root, realClientFiles(t), realAllowlist(t)); len(got) != 0 {
		t.Fatalf("real client tree violates the §2.5 fence: %v", got)
	}

	bad := `
public class Foo
{
    void M()
    {
        var x = GameObject.Find("n");
        var y = Object.FindObjectOfType<T>();
        SendMessage("m");
        Invoke("m", 1f);
        StartCoroutine(Co());
        Resources.Load("p");
        req.WaitForCompletion();
        Debug.Log("x");
        var m = r.material;
        var m2 = new Material(s);
        GC.Collect();
        var r2 = UnityEngine.Random.Range(0, 1);
        var c = Camera.main;
        var l = UnityEngine.Pool.ListPool<int>.Get();
        transform.position = System.Linq.Enumerable.First(list);
    }
    async void Fire() {}
    public UnityEvent OnHit;
    static int counter;
}
class T {}
`
	clean := `
public class Ok
{
    void M()
    {
        var s = $"hole {value} braces {{";
        Log.Info("fine");
        FrameTime.Now();
        // GameObject.Find in a comment is fine
        var t = "Resources.Load in a string is fine";
        action.Invoke();
        transform.Find("child");
        StopAllCoroutinesInComment /* SendMessage( */
    }
}
`
	files := map[string]string{
		"client/Assets/Scripts/Systems/Foo.cs": bad,
		"client/Assets/Scripts/Systems/Ok.cs":  clean,
	}
	root, list := fixtureTree(t, files)
	problems := ClientFenceProblems(root, list, map[string]map[string]bool{})
	joined := strings.Join(problems, "\n")
	for _, want := range []string{
		"find_calls", "coroutines", "resources_load", "debug_log",
		"runtime_material", "gc_collect", "unity_random", "camera_main",
		"unityengine_pool", "linq", "async_void", "unityevent_fields",
		"static_mutable_outside_app",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("fence missed %s; got:\n%s", want, joined)
		}
	}
	for _, p := range problems {
		if strings.HasPrefix(p, "client/Assets/Scripts/Systems/Ok.cs") {
			t.Errorf("clean file flagged: %s", p)
		}
	}
	// Suppressed by an exact allowlist entry.
	allow := map[string]map[string]bool{
		"client/Assets/Scripts/Systems/Foo.cs": {"debug_log": true},
	}
	for _, p := range ClientFenceProblems(root, list, allow) {
		if strings.Contains(p, "debug_log") {
			t.Fatalf("allowlisted symbol still flagged: %s", p)
		}
	}
}

func TestAllowlistEntriesNeedReason(t *testing.T) {
	// reasonless entry fails
	_, problems := ParseAllowlist("client/Assets/Scripts/Core/Runtime/FrameLoop.cs:update_outside_frameloop\n")
	if len(problems) == 0 {
		t.Fatal("reasonless entry accepted")
	}
	// unknown symbol fails
	_, problems = ParseAllowlist("client/Assets/Scripts/Core/Runtime/FrameLoop.cs:not_a_rule  because\n")
	if len(problems) == 0 {
		t.Fatal("unknown rule symbol accepted")
	}
	// non-runtime path fails
	_, problems = ParseAllowlist("server/internal/x/y.go:debug_log  because\n")
	if len(problems) == 0 {
		t.Fatal("non-client path accepted")
	}
	// duplicate entry fails
	_, problems = ParseAllowlist(
		"client/Assets/Scripts/Core/Runtime/FrameLoop.cs:update_outside_frameloop  r1\n" +
			"client/Assets/Scripts/Core/Runtime/FrameLoop.cs:update_outside_frameloop  r2\n")
	if len(problems) == 0 {
		t.Fatal("duplicate entry accepted")
	}
	// the shipped file parses clean and covers FrameLoop's Unity callbacks
	allow, problems := LoadAllowlist(repoRoot(t))
	if len(problems) != 0 {
		t.Fatalf("shipped allowlist malformed: %v", problems)
	}
	if !allow["client/Assets/Scripts/Core/Runtime/FrameLoop.cs"]["update_outside_frameloop"] {
		t.Fatal("shipped allowlist does not exempt FrameLoop Unity callbacks")
	}
}

func TestFenceExcludesEditorTestsGenerated(t *testing.T) {
	bad := `class B { void M() { GameObject.Find("x"); Debug.Log("y"); } }`
	files := map[string]string{
		"client/Assets/Scripts/Core/Editor/E.cs":   bad,
		"client/Assets/Scripts/UI/Editor/E.cs":     bad,
		"client/Assets/Scripts/Systems/Tests/T.cs": bad,
		"client/Assets/Scripts/Protocol/P.cs":      bad,
		"client/Assets/Scripts/UI/Hud.cs":          bad, // in scope -> flagged
	}
	root, list := fixtureTree(t, files)
	problems := ClientFenceProblems(root, list, nil)
	for _, p := range problems {
		if !strings.HasPrefix(p, "client/Assets/Scripts/UI/Hud.cs") {
			t.Errorf("out-of-scope file flagged: %s", p)
		}
	}
	if len(problems) == 0 {
		t.Fatal("in-scope violation not reported")
	}
}

func TestFrameLoopOnlyUnityCallbacks(t *testing.T) {
	loop := `
public sealed class FrameLoop
{
    void Update() { Tick(); }
    void FixedUpdate() {}
    void LateUpdate() {}
    void OnGUI() {}
}
`
	other := `
public class Other
{
    void Update() {}
    static void OnGUI() {}
}
`
	files := map[string]string{
		"client/Assets/Scripts/Core/Runtime/FrameLoop.cs": loop,
		"client/Assets/Scripts/Systems/Other.cs":          other,
	}
	root, list := fixtureTree(t, files)
	allow, _ := ParseAllowlist(
		"client/Assets/Scripts/Core/Runtime/FrameLoop.cs:update_outside_frameloop  the one frame driver\n")
	problems := ClientFenceProblems(root, list, allow)
	if len(problems) != 2 {
		t.Fatalf("want exactly 2 flags on Other.cs, got %v", problems)
	}
	for _, p := range problems {
		if !strings.HasPrefix(p, "client/Assets/Scripts/Systems/Other.cs") ||
			!strings.Contains(p, "update_outside_frameloop") {
			t.Errorf("unexpected flag: %s", p)
		}
	}
	// Without the allowlist entry FrameLoop.cs is flagged too — the exemption
	// is the entry, not a builtin.
	problems = ClientFenceProblems(root, list, nil)
	var flagged bool
	for _, p := range problems {
		if strings.HasPrefix(p, "client/Assets/Scripts/Core/Runtime/FrameLoop.cs") {
			flagged = true
		}
	}
	if !flagged {
		t.Fatal("FrameLoop callbacks unflagged without the allowlist entry")
	}
	// a callback-looking *call* (prev '.', or a non-declaration) is not flagged
	calls := `
public class C
{
    void M()
    {
        other.Update();
        unityObject.Update(1, 2);
        Update2();
    }
    void Update2() {}
}
`
	files = map[string]string{"client/Assets/Scripts/Systems/C.cs": calls}
	root, list = fixtureTree(t, files)
	if got := ClientFenceProblems(root, list, nil); len(got) != 0 {
		t.Fatalf("non-declaration Update use flagged: %v", got)
	}
}

func TestCanonicalImplementationsUnique(t *testing.T) {
	// Real tree must satisfy CODE-006.
	root := repoRoot(t)
	if got := CanonicalImplementationProblems(root, realClientFiles(t)); len(got) != 0 {
		t.Fatalf("real client tree: %v", got)
	}
	if got := GoCanonicalProblems(root, realGoFiles(t)); len(got) != 0 {
		t.Fatalf("real server tree: %v", got)
	}
	// second implementations outside owner paths fail
	files := map[string]string{
		"client/Assets/Scripts/Core/Runtime/FrameLoop.cs":       "public sealed class FrameLoop {}\n",
		"client/Assets/Scripts/Systems/BadLogger.cs":            "public class BadLogger {}\n",
		"client/Assets/Scripts/UI/MyPool.cs":                    "public class MyPool {}\n",
		"client/Assets/Scripts/App/AltClock.cs":                 "public interface IClock {}\npublic class AltClock : IClock {}\n",
		"client/Assets/Scripts/Net/ClientRandom.cs":             "public class ClientRandom {}\n",
		"client/Assets/Scripts/Systems/Camera/CameraService.cs": "public class CameraService {}\n",
		"client/Assets/Scripts/Systems/PlainThing.cs":           "public class PlainThing {}\n",
	}
	root2, list := fixtureTree(t, files)
	problems := CanonicalImplementationProblems(root2, list)
	joined := strings.Join(problems, "\n")
	for _, want := range []string{"BadLogger", "MyPool", "AltClock", "ClientRandom"} {
		if !strings.Contains(joined, want) {
			t.Errorf("duplicate %s implementation not flagged; got:\n%s", want, joined)
		}
	}
	for _, p := range problems {
		if strings.Contains(p, "CameraService.cs") || strings.Contains(p, "PlainThing.cs") ||
			strings.Contains(p, "FrameLoop.cs") {
			t.Errorf("owner/legal file flagged: %s", p)
		}
	}
	// Go side: Logger outside observability/core, rand v2 outside core/rng
	tmp := t.TempDir()
	writeFile(t, tmp, "server/internal/sim/mylogger.go", "package sim\n\ntype SimLogger struct{}\n")
	writeFile(t, tmp, "server/internal/edge/r.go", goFile("math/rand/v2"))
	writeFile(t, tmp, "server/internal/observability/l.go", "package observability\n\ntype Logger struct{}\n")
	gfiles := []string{
		"server/internal/sim/mylogger.go",
		"server/internal/edge/r.go",
		"server/internal/observability/l.go",
	}
	gp := GoCanonicalProblems(tmp, gfiles)
	gj := strings.Join(gp, "\n")
	if !strings.Contains(gj, "SimLogger") || !strings.Contains(gj, "math/rand/v2") {
		t.Fatalf("Go canonical violations missed: %v", gp)
	}
	for _, p := range gp {
		if strings.Contains(p, "observability/l.go") {
			t.Errorf("owner path flagged: %s", p)
		}
	}
}
