// Shared spec-document parsers for the wire-schema test suite.
// These read the protected contract docs so tests fail when proto and spec
// diverge, not when test tables drift from the spec.
package protocol_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	protocolv1 "thinhthan/internal/protocol/v1"
)

const enumPrefix = "MESSAGE_ID_"

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// server/internal/testing/protocol/<file> -> repo root
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

func docText(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

// fencedBlocks returns the bodies of every ```-fenced block in doc text.
var fenceRe = regexp.MustCompile("(?s)```[a-z]*\n(.*?)```")

func fencedBlocks(doc string) []string {
	var out []string
	for _, m := range fenceRe.FindAllStringSubmatch(doc, -1) {
		out = append(out, m[1])
	}
	return out
}

var registryRowRe = regexp.MustCompile(`^\s*(\d+)\s+((?:C2S|S2C)_[A-Z0-9_]+)\b`)

// fieldListRowRe matches compact payload rows (`105 S2C_TRANSFER_PREPARE
// transfer_id (16 bytes), ...`) — registry rows carry a direction column
// (`C→S`/`S→C`/`—`) and are excluded by the caller.
var fieldListRowRe = regexp.MustCompile(`^\s*(\d+)\s+((?:C2S|S2C)_[A-Z0-9_]+)\s+(.+)$`)
var typedFieldRe = regexp.MustCompile("`?([a-z_]+)`?\\s*(?:\\(\\s*|:\\s*)(sint32|sint64|int32|int64|uint32|uint64|bool|string|bytes)\\b")

var protoKinds = map[string]protoreflect.Kind{
	"sint32": protoreflect.Sint32Kind, "sint64": protoreflect.Sint64Kind,
	"int32": protoreflect.Int32Kind, "int64": protoreflect.Int64Kind,
	"uint32": protoreflect.Uint32Kind, "uint64": protoreflect.Uint64Kind,
	"bool": protoreflect.BoolKind, "string": protoreflect.StringKind,
	"bytes": protoreflect.BytesKind,
}

// specFieldTypes returns per-message field type annotations parsed from the
// compact field lists of messages.md: message id -> field name -> kind.
func specFieldTypes(t *testing.T) map[int32]map[string]protoreflect.Kind {
	t.Helper()
	out := map[int32]map[string]protoreflect.Kind{}
	doc := docText(t, "docs/05_network/messages.md")
	for _, block := range fencedBlocks(doc) {
		for _, line := range strings.Split(block, "\n") {
			m := fieldListRowRe.FindStringSubmatch(line)
			if m == nil || strings.ContainsAny(m[3], "→—") {
				continue
			}
			id, _ := strconv.ParseInt(m[1], 10, 32)
			fields := out[int32(id)]
			if fields == nil {
				fields = map[string]protoreflect.Kind{}
				out[int32(id)] = fields
			}
			for _, fm := range typedFieldRe.FindAllStringSubmatch(m[3], -1) {
				fields[fm[1]] = protoKinds[fm[2]]
			}
		}
	}
	return out
}

// specMessageIDs parses the fenced registry tables of messages.md:
// id -> payload message name.
func specMessageIDs(t *testing.T) map[int32]string {
	t.Helper()
	out := map[int32]string{}
	for _, block := range fencedBlocks(docText(t, "docs/05_network/messages.md")) {
		for _, line := range strings.Split(block, "\n") {
			m := registryRowRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			id, err := strconv.ParseInt(m[1], 10, 32)
			if err != nil {
				t.Fatalf("bad registry row %q", line)
			}
			if prev, ok := out[int32(id)]; ok && prev != m[2] {
				t.Fatalf("registry id %d maps to both %s and %s", id, prev, m[2])
			}
			out[int32(id)] = m[2]
		}
	}
	if len(out) == 0 {
		t.Fatal("no registry rows parsed from messages.md")
	}
	return out
}

// specErrorCodes returns the Canonical Codes tokens of errors.md in
// row-major document order: fenced blocks inside the `## Canonical Codes`
// section only (ADR-0069).
var errorTokenRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]+$`)

func specErrorCodes(t *testing.T) []string {
	t.Helper()
	doc := docText(t, "docs/05_network/errors.md")
	i := strings.Index(doc, "## Canonical Codes")
	if i < 0 {
		t.Fatal("errors.md lacks ## Canonical Codes")
	}
	sec := doc[i:]
	if j := strings.Index(sec[len("## Canonical Codes"):], "\n## "); j >= 0 {
		sec = sec[:len("## Canonical Codes")+j]
	}
	var codes []string
	for _, block := range fencedBlocks(sec) {
		for _, line := range strings.Split(block, "\n") {
			for _, tok := range strings.Fields(line) {
				if errorTokenRe.MatchString(tok) {
					codes = append(codes, tok)
				}
			}
		}
	}
	if len(codes) == 0 {
		t.Fatal("no error codes parsed from errors.md fenced blocks")
	}
	return codes
}

func messageType(t *testing.T, name string) protoreflect.MessageType {
	t.Helper()
	mt, err := protoregistry.GlobalTypes.FindMessageByName(
		protoreflect.FullName("thinhthan.v1." + name))
	if err != nil {
		t.Fatalf("message type %q not registered: %v", name, err)
	}
	return mt
}

// payloadMessageID maps a payload message name to its registry id.
func payloadMessageID(name string) (int32, bool) {
	v, ok := protocolv1.MessageId_value[enumPrefix+name]
	return v, ok
}

// eachField visits every field of every message in package thinhthan.v1.
func eachField(fn func(msg protoreflect.MessageDescriptor, f protoreflect.FieldDescriptor)) {
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Package() != "thinhthan.v1" {
			return true
		}
		var walk func(msgs protoreflect.MessageDescriptors)
		walk = func(msgs protoreflect.MessageDescriptors) {
			for i := 0; i < msgs.Len(); i++ {
				md := msgs.Get(i)
				fs := md.Fields()
				for j := 0; j < fs.Len(); j++ {
					fn(md, fs.Get(j))
				}
				walk(md.Messages())
			}
		}
		walk(fd.Messages())
		return true
	})
}

// eachPayloadMessage visits the registered payload messages (C2S_/S2C_).
func eachPayloadMessage(fn func(name string, md protoreflect.MessageDescriptor)) {
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Package() != "thinhthan.v1" {
			return true
		}
		msgs := fd.Messages()
		for i := 0; i < msgs.Len(); i++ {
			md := msgs.Get(i)
			name := string(md.Name())
			if strings.HasPrefix(name, "C2S_") || strings.HasPrefix(name, "S2C_") {
				fn(name, md)
			}
		}
		return true
	})
}

// runCodegen executes the canonical generator (ADR-0050: pwsh only).
func runCodegen(t *testing.T) {
	t.Helper()
	root := repoRoot(t)
	cmd := exec.Command("pwsh", "-NoProfile", "-File", "scripts/codegen.ps1", "-RepoRoot", root)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("codegen.ps1 failed: %v\n%s", err, out)
	}
}

// gitStatus reports the porcelain diff of paths under the repo root.
func gitStatus(t *testing.T, paths ...string) string {
	t.Helper()
	args := append([]string{"-C", repoRoot(t), "status", "--porcelain", "--"}, paths...)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		t.Fatalf("git status failed: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func mustMarshal(t *testing.T, m proto.Message) []byte {
	t.Helper()
	raw, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
