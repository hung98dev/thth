package style

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot walks up from the package dir to the repo root (.git marker).
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		p := filepath.Dir(dir)
		if p == dir {
			t.Fatal("repo root not found")
		}
		dir = p
	}
}

const goodFile = `namespace ThinhThan.Core
{
    public class Good
    {
        private int _count;

        public int Bump()
        {
            _count++;
            return _count;
        }
    }
}
`

func TestBraceLines(t *testing.T) {
	if v := CheckFile("Good.cs", goodFile, "ThinhThan.Core"); len(v) != 0 {
		t.Fatalf("Allman file rejected: %v", v)
	}
	bad := strings.Replace(goodFile, "public class Good\n    {", "public class Good {", 1)
	if v := CheckFile("Bad.cs", bad, "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("inline '{' must be flagged")
	}
	bad = strings.Replace(goodFile, "        }\n", "        } else\n", 1)
	if v := CheckFile("Bad.cs", bad, "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("'} else' must be flagged")
	}
}

func TestIndentAndWhitespace(t *testing.T) {
	bad := strings.Replace(goodFile, "        private int _count;", "   private int _count;", 1)
	if v := CheckFile("Bad.cs", bad, "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("3-space indent must be flagged")
	}
	bad = strings.Replace(goodFile, "        private int _count;", "        private int _count; ", 1)
	if v := CheckFile("Bad.cs", bad, "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("trailing whitespace must be flagged")
	}
	bad = strings.Replace(goodFile, "        private int _count;", "\tprivate int _count;", 1)
	if v := CheckFile("Bad.cs", bad, "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("tab indent must be flagged")
	}
}

func TestLineEndingsBomFinalNewline(t *testing.T) {
	if v := CheckFile("Bad.cs", "\ufeff"+goodFile, "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("BOM must be flagged")
	}
	if v := CheckFile("Bad.cs", strings.ReplaceAll(goodFile, "\n", "\r\n"), "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("CRLF must be flagged")
	}
	if v := CheckFile("Bad.cs", strings.TrimSuffix(goodFile, "\n"), "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("missing final newline must be flagged")
	}
}

func TestPrivateFieldNaming(t *testing.T) {
	bad := strings.Replace(goodFile, "private int _count;", "private int count;", 1)
	if v := CheckFile("Bad.cs", bad, "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("non-underscore private field must be flagged")
	}
	bad = strings.Replace(goodFile, "private int _count;", "private int _Count;", 1)
	if v := CheckFile("Bad.cs", bad, "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("_PascalCase private field must be flagged")
	}
	ok := strings.Replace(goodFile, "private int _count;", "private const int Max = 3;\n        private static int _shared;\n        private readonly int _count;", 1)
	if v := CheckFile("Good.cs", ok, "ThinhThan.Core"); len(v) != 0 {
		t.Fatalf("const/static/readonly fields must be exempt: %v", v)
	}
}

func TestOneTypePerFileAndNamespace(t *testing.T) {
	two := goodFile + "\nnamespace ThinhThan.Core\n{\n    public class Other\n    {\n    }\n}\n"
	if v := CheckFile("Good.cs", two, "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("two types in one file must be flagged")
	}
	if v := CheckFile("WrongName.cs", goodFile, "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("type name != filename must be flagged")
	}
	if v := CheckFile("Good.cs", strings.Replace(goodFile, "ThinhThan.Core", "ThinhThan.Wrong", 1), "ThinhThan.Core"); len(v) == 0 {
		t.Fatal("namespace mismatch must be flagged")
	}
}

func TestEditorconfigGitattributesKeys(t *testing.T) {
	root := repoRoot(t)
	ec, err := os.ReadFile(filepath.Join(root, ".editorconfig"))
	if err != nil {
		t.Fatal(".editorconfig missing")
	}
	for _, key := range []string{"root = true", "end_of_line = lf", "insert_final_newline = true",
		"trim_trailing_whitespace = true", "indent_style = space", "indent_size = 4",
		"indent_style = tab", "indent_size = 2"} {
		if !strings.Contains(string(ec), key) {
			t.Fatalf(".editorconfig missing canonical key %q", key)
		}
	}
	ga, err := os.ReadFile(filepath.Join(root, ".gitattributes"))
	if err != nil {
		t.Fatal(".gitattributes missing")
	}
	s := string(ga)
	if !strings.Contains(s, "text=auto eol=lf") {
		t.Fatal(".gitattributes must force LF")
	}
	// Binary assets must be LFS; unity yaml files must not be LFS.
	for _, pat := range []string{"*.png", "*.wav", "*.fbx"} {
		if !strings.Contains(s, pat+" filter=lfs") {
			t.Fatalf(".gitattributes: %s must be LFS", pat)
		}
	}
	for _, pat := range []string{"*.unity", "*.prefab", "*.asset", "*.meta", "*.mat"} {
		if strings.Contains(s, pat+" filter=lfs") || !strings.Contains(s, pat+" merge=unityyamlmerge eol=lf") {
			t.Fatalf(".gitattributes: %s must be unityyamlmerge + LF, never LFS", pat)
		}
	}
}

func TestGoFmtVetStaticcheckWired(t *testing.T) {
	root := repoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "server", "internal", "conformance", "gates", "q3.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"gofmt", "vet", "staticcheck"} {
		if !strings.Contains(string(src), want) {
			t.Fatalf("q3 verifier must wire %s", want)
		}
	}
}

func TestLintFileIgnoreRejected(t *testing.T) {
	for _, supp := range []string{"//lint:file-ignore", "#pragma warning disable", "// ReSharper disable"} {
		bad := supp + "\n" + goodFile
		if v := CheckFile("Bad.cs", bad, "ThinhThan.Core"); len(v) == 0 {
			t.Fatalf("%q must be flagged (CODE-003)", supp)
		}
	}
	if v := CheckFile("Good.cs", goodFile, "ThinhThan.Core"); len(v) != 0 {
		t.Fatalf("clean file rejected: %v", v)
	}
}

func TestAllocPassAndBenchReportWired(t *testing.T) {
	root := repoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "server", "internal", "conformance", "gates", "q3.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"-benchmem", "-bench", "benchtime"} {
		if !strings.Contains(string(src), want) {
			t.Fatalf("q3 verifier must wire bench flag %q (allocation + bench report)", want)
		}
	}
}
