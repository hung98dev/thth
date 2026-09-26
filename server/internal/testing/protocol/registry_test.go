// Package protocol_test verifies the canonical wire registry:
//   - every MessageId enum value maps 1:1 to a same-named message type;
//   - golden binary fixtures decode deterministically;
//   - codegen drift + file-boundary rules (asmdef preserved, no .meta writes);
//   - the generated C# header (CODE-004).
package protocol_test

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	protocolv1 "thinhthan/internal/protocol/v1"
)

var updateGolden = flag.Bool("update-golden", false,
	"rewrite proto/testdata/golden/*.bin from the compiled schema")

// TestMessageRegistryMapping: enum <-> spec registry <-> compiled messages.
func TestMessageRegistryMapping(t *testing.T) {
	spec := specMessageIDs(t)

	// Forward: every non-zero enum value is a spec-registered payload message.
	for value, enumName := range protocolv1.MessageId_name {
		if !strings.HasPrefix(enumName, enumPrefix) {
			t.Fatalf("enum value %s lacks %s prefix", enumName, enumPrefix)
		}
		msgName := strings.TrimPrefix(enumName, enumPrefix)
		if value == 0 {
			if msgName != "UNSPECIFIED" {
				t.Fatalf("enum value 0 must be MESSAGE_ID_UNSPECIFIED, got %s", enumName)
			}
			continue
		}
		specName, ok := spec[value]
		if !ok {
			t.Fatalf("enum %s = %d is not registered in messages.md", enumName, value)
		}
		if specName != msgName {
			t.Fatalf("id %d: spec names it %s, enum names it %s", value, specName, msgName)
		}
		messageType(t, msgName)
	}

	// Set equality: spec registry == enum minus UNSPECIFIED.
	if got, want := len(protocolv1.MessageId_name)-1, len(spec); got != want {
		t.Fatalf("registry size %d != spec table size %d", got, want)
	}

	// Reverse: every C2S_/S2C_ message compiled into thinhthan.v1 is registered.
	eachPayloadMessage(func(name string, _ protoreflect.MessageDescriptor) {
		if _, ok := protocolv1.MessageId_value[enumPrefix+name]; !ok {
			t.Fatalf("payload message %s has no %s enum value", name, enumPrefix)
		}
	})

	// Retired IDs stay unassigned.
	if _, ok := protocolv1.MessageId_name[737]; ok {
		t.Fatal("message_id 737 is retired and must stay unassigned")
	}
}

// TestBinaryEncodingParity decodes every golden fixture and re-encodes it
// deterministically; bytes must be reproduced exactly.
func TestBinaryEncodingParity(t *testing.T) {
	dir := goldenDir(t)

	if *updateGolden {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeGoldenFixtures(t, dir)
		t.Skip("fixtures regenerated")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read golden dir: %v", err)
	}
	count := 0
	for _, e := range entries {
		base := e.Name()
		if e.IsDir() || !strings.HasSuffix(base, ".bin") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, base))
		if err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		if base == "envelope.bin" {
			checkEnvelopeFixture(t, raw)
			count++
			continue
		}
		stem := strings.TrimSuffix(base, ".bin")
		parts := strings.SplitN(stem, "_", 2)
		idNum, err := strconv.Atoi(parts[0])
		if err != nil || len(parts) != 2 {
			t.Fatalf("%s: fixture name must be <id>_<NAME>.bin", base)
		}
		msgName := parts[1]
		if enumName, ok := protocolv1.MessageId_name[int32(idNum)]; !ok ||
			strings.TrimPrefix(enumName, enumPrefix) != msgName {
			t.Fatalf("%s: id/name does not match registry", base)
		}
		m := messageType(t, msgName).New().Interface()
		if err := proto.Unmarshal(raw, m); err != nil {
			t.Fatalf("%s: unmarshal: %v", base, err)
		}
		out, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
		if err != nil {
			t.Fatalf("%s: marshal: %v", base, err)
		}
		if string(out) != string(raw) {
			t.Fatalf("%s: re-encoded bytes differ\n got: % x\nwant: % x", base, out, raw)
		}
		count++
	}
	if count == 0 {
		t.Fatal("no golden fixtures found")
	}
	t.Logf("%d golden fixtures decoded and re-encoded identically", count)
}

func checkEnvelopeFixture(t *testing.T, raw []byte) {
	t.Helper()
	var env protocolv1.Envelope
	if err := proto.Unmarshal(raw, &env); err != nil {
		t.Fatalf("envelope.bin: unmarshal: %v", err)
	}
	enumName, ok := protocolv1.MessageId_name[int32(env.GetMessageId())]
	if !ok {
		t.Fatalf("envelope.bin: unknown message_id %d", env.GetMessageId())
	}
	m := messageType(t, strings.TrimPrefix(enumName, enumPrefix)).New().Interface()
	if err := proto.Unmarshal(env.GetPayload(), m); err != nil {
		t.Fatalf("envelope.bin payload (%s): %v", enumName, err)
	}
	out, err := proto.MarshalOptions{Deterministic: true}.Marshal(&env)
	if err != nil || string(out) != string(raw) {
		t.Fatalf("envelope.bin: re-encoded bytes differ (err=%v)", err)
	}
}

// TestCodegenDriftCheck mirrors verify gate Q2: regeneration must produce
// zero uncommitted drift in every generated path.
func TestCodegenDriftCheck(t *testing.T) {
	runCodegen(t)
	if drift := gitStatus(t,
		"proto/testdata/golden",
		"server/internal/protocol/v1",
		"client/Assets/Scripts/Protocol"); drift != "" {
		t.Fatalf("codegen drift detected:\n%s", drift)
	}
}

// TestGeneratedCSharpHeader enforces CODE-004: every generated C# file begins
// with `#nullable disable` + the protobuf pragma disable set.
func TestGeneratedCSharpHeader(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "client", "Assets", "Scripts", "Protocol")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".cs") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		head := "#nullable disable\n#pragma warning disable 1591, 0612, 3021, 8981\n"
		if !strings.HasPrefix(string(data), head) {
			t.Fatalf("%s: missing CODE-004 header", e.Name())
		}
		count++
	}
	if count == 0 {
		t.Fatal("no generated C# files found")
	}
}

// TestAdr0060MessagesRegistered asserts every message ID added or completed
// by ADR-0060 exists in the enum and resolves to a message type.
func TestAdr0060MessagesRegistered(t *testing.T) {
	var ids []int32
	add := func(lo, hi int32) {
		for i := lo; i <= hi; i++ {
			ids = append(ids, i)
		}
	}
	add(111, 117) // dungeon entry/exit
	ids = append(ids, 208)
	add(426, 438) // durable additions (inventory expand, cosmetics, beast, souls)
	add(507, 515) // content additions
	ids = append(ids, 650, 651, 652)
	add(814, 818) // duel + pvp result family
	for _, id := range ids {
		enumName, ok := protocolv1.MessageId_name[id]
		if !ok {
			t.Fatalf("ADR-0060 message id %d missing from MessageId enum", id)
		}
		messageType(t, strings.TrimPrefix(enumName, enumPrefix))
	}
}

// TestErrorEnumMatchesErrorsMd: the generated ErrorCode enum is exactly the
// Canonical Codes of errors.md — no missing and no extra values.
func TestErrorEnumMatchesErrorsMd(t *testing.T) {
	spec := specErrorCodes(t)
	got := map[string]int32{}
	for value, name := range protocolv1.ErrorCode_name {
		if value == 0 {
			continue
		}
		got[strings.TrimPrefix(name, "ERROR_CODE_")] = value
	}
	want := map[string]int32{}
	for i, code := range spec {
		if prev, dup := want[code]; dup {
			t.Fatalf("errors.md lists %s twice (rows %d and %d)", code, prev, i+1)
		}
		want[code] = int32(i + 1)
	}
	for name, v := range want {
		if gv, ok := got[name]; !ok {
			t.Fatalf("ERROR_CODE_%s missing from enum", name)
		} else if gv != v {
			t.Fatalf("ERROR_CODE_%s = %d, want %d (errors.md row-major)", name, gv, v)
		}
	}
	for name, v := range got {
		if _, ok := want[name]; !ok {
			t.Fatalf("enum has ERROR_CODE_%s = %d not in errors.md", name, v)
		}
	}
}

// TestCodegenPreservesProtocolAsmdef: regeneration must leave the IMP-000
// authored ThinhThan.Protocol.asmdef byte-identical (ADR-0068/ADR-0072).
func TestCodegenPreservesProtocolAsmdef(t *testing.T) {
	runCodegen(t)
	if drift := gitStatus(t,
		"client/Assets/Scripts/Protocol/ThinhThan.Protocol.asmdef"); drift != "" {
		t.Fatalf("codegen rewrote ThinhThan.Protocol.asmdef:\n%s", drift)
	}
}

// TestCodegenNeverWritesMeta: regeneration must never create or modify .meta
// files (those are editor-materialized in CI, agent_execution_protocol.md §4b).
func TestCodegenNeverWritesMeta(t *testing.T) {
	runCodegen(t)
	for _, line := range strings.Split(gitStatus(t,
		"client/Assets/Scripts/Protocol", "proto"), "\n") {
		if strings.HasSuffix(line, ".meta") {
			t.Fatalf("codegen wrote .meta files:\n%s", line)
		}
	}
}
