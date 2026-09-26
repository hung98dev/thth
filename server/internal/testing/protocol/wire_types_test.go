// Wire-type contract tests (ADR-0064): canonical scalar types, UUID bytes,
// OperationResult embedding, and the append-only ErrorCode numbering rules.
package protocol_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	protocolv1 "thinhthan/internal/protocol/v1"
)

// uuidFieldNames collects field names annotated `: UUID` in messages.md.
var uuidAnnotRe = regexp.MustCompile("`([a-z_]+)`?\\s*:\\s*UUID")

func uuidFieldNames(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, m := range uuidAnnotRe.FindAllStringSubmatch(
		docText(t, "docs/05_network/messages.md"), -1) {
		out[m[1]] = true
	}
	// Fields universally understood as UUIDs across sections that the prose
	// does not annotate per-message.
	for _, n := range []string{
		"session_id", "account_id", "character_id", "device_id",
		"operation_id", "item_instance_id", "guild_id", "party_id",
		"match_id", "challenge_id", "entry_id", "trade_id", "settlement_id",
		"listing_id", "proceeds_id", "escrow_asset_id", "reward_claim_id",
		"claim_id", "report_id", "chat_message_id", "resumed_character_id",
		"requester_character_id", "target_character_id", "sender_character_id",
		"inviter_character_id", "friend_character_id", "blocked_character_id",
		"applicant_character_id", "leader_character_id", "deposited_by",
		"public_boss_spawn_generation_id", "equipped_item_instance_id",
		"unequipped_item_instance_id",
	} {
		out[n] = true
	}
	return out
}

// TestUuidFieldsAreBytes16: every UUID on the wire is 16-byte `bytes`
// (ADR-0064) — never string.
func TestUuidFieldsAreBytes16(t *testing.T) {
	uuids := uuidFieldNames(t)
	eachField(func(md protoreflect.MessageDescriptor, f protoreflect.FieldDescriptor) {
		name := string(f.Name())
		if uuids[name] {
			if f.Kind() != protoreflect.BytesKind {
				t.Fatalf("%s.%s is a UUID but is %s, want bytes", md.Name(), name, f.Kind())
			}
			if f.IsList() {
				t.Fatalf("%s.%s is a UUID but is repeated", md.Name(), name)
			}
			return
		}
		// Bytes fields that are not UUIDs must be opaque payload (the envelope
		// payload or a mechanic-specific sub-message blob); repeated bytes is
		// a UUID list (roster_character_ids, removed, ...).
		if f.Kind() == protoreflect.BytesKind && name != "payload" && !f.IsList() &&
			!strings.HasSuffix(name, "_id") && !strings.HasSuffix(name, "_ids") &&
			!strings.HasSuffix(name, "_payload") && name != "deposited_by" {
			t.Fatalf("%s.%s: bytes field is neither a UUID *_id nor Envelope.payload",
				md.Name(), name)
		}
	})
}

// TestErrorCodeEnumCoversErrorsMdInOrder: every Canonical Code appears in the
// enum, and enum numbers strictly follow document order.
func TestErrorCodeEnumCoversErrorsMdInOrder(t *testing.T) {
	prev := int32(0)
	for i, code := range specErrorCodes(t) {
		enumName := "ERROR_CODE_" + code
		v, ok := protocolv1.ErrorCode_value[enumName]
		if !ok {
			t.Fatalf("errors.md row %d code %s missing from ErrorCode", i+1, enumName)
		}
		if v <= prev {
			t.Fatalf("%s = %d breaks row-major order (prev %d)", enumName, v, prev)
		}
		prev = v
	}
}

// TestErrorCodeNumbersAppendOnly: each code's number equals its 1-based
// document position — numbers can never be resequenced, only appended.
func TestErrorCodeNumbersAppendOnly(t *testing.T) {
	if protocolv1.ErrorCode_value["ERROR_CODE_UNSPECIFIED"] != 0 {
		t.Fatal("ERROR_CODE_UNSPECIFIED must be 0")
	}
	for i, code := range specErrorCodes(t) {
		want := int32(i + 1)
		if got := protocolv1.ErrorCode_value["ERROR_CODE_"+code]; got != want {
			t.Fatalf("ERROR_CODE_%s = %d, want %d (append-only row position)",
				code, got, want)
		}
	}
}

// TestErrorCodeFencedRowMajorOrder: codes are numbered only from the fenced
// blocks of ## Canonical Codes (ADR-0069) — exactly specLen values assigned.
func TestErrorCodeFencedRowMajorOrder(t *testing.T) {
	spec := specErrorCodes(t)
	// enum must hold UNSPECIFIED + exactly the fenced codes, in order
	if got := len(protocolv1.ErrorCode_name); got != len(spec)+1 {
		t.Fatalf("ErrorCode has %d values; errors.md fenced blocks define %d (+UNSPECIFIED)",
			got, len(spec))
	}
	for i, code := range spec {
		name, ok := protocolv1.ErrorCode_name[int32(i+1)]
		if !ok || name != "ERROR_CODE_"+code {
			t.Fatalf("row %d: enum has %q, errors.md has %q", i+1, name, "ERROR_CODE_"+code)
		}
	}
}

// TestResultsEmbedOperationResult: every *_RESULT message carries
// OperationResult as field 1 (protobuf_conventions.md § 6).
func TestResultsEmbedOperationResult(t *testing.T) {
	eachPayloadMessage(func(name string, md protoreflect.MessageDescriptor) {
		if !strings.HasSuffix(name, "_RESULT") {
			return
		}
		f := md.Fields().ByNumber(1)
		if f == nil || f.Kind() != protoreflect.MessageKind ||
			f.Message().FullName() != "thinhthan.v1.OperationResult" ||
			f.Name() != "result" {
			t.Fatalf("%s: field 1 must be `OperationResult result`", name)
		}
	})
}

// TestOutcomeMessagesHaveNoOperationResult: *_OUTCOME state events carry no
// OperationResult (ADR-0069) — outcomes are not operation answers.
func TestOutcomeMessagesHaveNoOperationResult(t *testing.T) {
	eachPayloadMessage(func(name string, md protoreflect.MessageDescriptor) {
		if !strings.HasSuffix(name, "_OUTCOME") {
			return
		}
		fs := md.Fields()
		for i := 0; i < fs.Len(); i++ {
			f := fs.Get(i)
			if f.Kind() == protoreflect.MessageKind &&
				f.Message().FullName() == "thinhthan.v1.OperationResult" {
				t.Fatalf("%s.%s must not carry OperationResult", name, f.Name())
			}
		}
	})
}

// TestAdr0064MessagesRegistered: ADR-0064 session/registry IDs exist and
// canonical wire scalars hold (positions sint32, ticks uint64, absolute
// timestamps int64, money int64, bitmasks uint32).
func TestAdr0064MessagesRegistered(t *testing.T) {
	for _, id := range []int32{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
		439, 440, 441, 653, 654, 655, 710, 819,
	} {
		enumName, ok := protocolv1.MessageId_name[id]
		if !ok {
			t.Fatalf("ADR-0064 session/registry id %d missing", id)
		}
		// C2S_CHARACTER_DETACH (10) and friends are legitimately empty.
		messageType(t, strings.TrimPrefix(enumName, enumPrefix))
	}

	moneyRe := regexp.MustCompile(`price|fee|tax|proceeds_amount|amount|_common$|_premium$|_special$`)
	typed := specFieldTypes(t)
	eachField(func(md protoreflect.MessageDescriptor, f protoreflect.FieldDescriptor) {
		name := string(f.Name())
		// An explicit type annotation in messages.md wins over conventions
		// (e.g. 105 ready_deadline_ms is a documented uint32 budget).
		if id, ok := payloadMessageID(string(md.Name())); ok {
			if want, ok := typed[id][name]; ok && f.Kind() != want {
				t.Fatalf("%s.%s: messages.md types it %s, schema has %s",
					md.Name(), name, want, f.Kind())
				return
			} else if ok {
				return
			}
		}
		check := func(want protoreflect.Kind, rule string) {
			if f.Kind() != want {
				t.Fatalf("%s.%s: %s requires %s, got %s",
					md.Name(), name, rule, want, f.Kind())
			}
		}
		switch {
		case strings.HasSuffix(name, "_mm") || strings.HasSuffix(name, "_mm_s"):
			check(protoreflect.Sint32Kind, "position/velocity")
		case strings.HasSuffix(name, "_tick") || name == "server_tick":
			check(protoreflect.Uint64Kind, "tick")
		case name == "flags" || name == "input_flags":
			check(protoreflect.Uint32Kind, "bitmask")
		case strings.HasSuffix(name, "_at") || strings.HasSuffix(name, "_at_ms") ||
			name == "server_time_ms" || strings.HasPrefix(name, "expires_at"):
			check(protoreflect.Int64Kind, "absolute timestamp")
		case moneyRe.MatchString(name) && !strings.HasSuffix(name, "_id"):
			check(protoreflect.Int64Kind, "money")
		case name == "quantity" || strings.HasSuffix(name, "_quantity") ||
			strings.HasSuffix(name, "_count") || name == "stacks":
			check(protoreflect.Uint32Kind, "count")
		}
	})
}

// TestNoOptionalRepeatedFields: ADR-0069 — lists needing presence use wrapper
// messages (StatusList/CosmeticList); `optional repeated` is forbidden syntax.
func TestNoOptionalRepeatedFields(t *testing.T) {
	srcDir := filepath.Join(repoRoot(t), "proto", "thinhthan", "v1")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatal(err)
	}
	optionalRepeated := regexp.MustCompile(`\boptional\s+repeated\b`)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".proto") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			code := strings.SplitN(line, "//", 2)[0]
			if optionalRepeated.MatchString(code) {
				t.Fatalf("%s:%d: `optional repeated` is forbidden", e.Name(), i+1)
			}
		}
	}
	// Descriptor sanity: a repeated field can never carry presence.
	eachField(func(md protoreflect.MessageDescriptor, f protoreflect.FieldDescriptor) {
		if f.IsList() && f.HasPresence() {
			t.Fatalf("%s.%s: repeated field claims optional presence", md.Name(), f.Name())
		}
	})
}

// TestAdr0069MessagesRegistered asserts the ADR-0069 schema corrections:
// S2C_RESUME_CREDENTIAL (16), SelfAck inside 303, StatusList/CosmeticList
// wrapper types, CharacterSummary.is_attached, 107.reason, the 813/818
// *_OUTCOME renames.
func TestAdr0069MessagesRegistered(t *testing.T) {
	if protocolv1.MessageId_name[16] != enumPrefix+"S2C_RESUME_CREDENTIAL" {
		t.Fatal("message_id 16 must be MESSAGE_ID_S2C_RESUME_CREDENTIAL")
	}
	messageType(t, "S2C_RESUME_CREDENTIAL")
	messageType(t, "StatusList")
	messageType(t, "CosmeticList")

	delta := messageType(t, "S2C_STATE_DELTA").Descriptor()
	if f := delta.Fields().ByName("self_ack"); f == nil ||
		f.Kind() != protoreflect.MessageKind ||
		f.Message().Name() != "SelfAck" {
		t.Fatal("S2C_STATE_DELTA must carry `SelfAck self_ack`")
	}

	summary := messageType(t, "CharacterSummary").Descriptor()
	if f := summary.Fields().ByName("is_attached"); f == nil || f.Kind() != protoreflect.BoolKind {
		t.Fatal("CharacterSummary must carry `bool is_attached` (ADR-0069)")
	}

	corr := messageType(t, "S2C_MOVEMENT_CORRECTION").Descriptor()
	if f := corr.Fields().ByName("reason"); f == nil ||
		f.Kind() != protoreflect.EnumKind ||
		f.Enum().Name() != "MovementCorrectionReason" {
		t.Fatal("S2C_MOVEMENT_CORRECTION (107) must carry `reason`")
	}

	if protocolv1.MessageId_name[813] != enumPrefix+"S2C_SPARRING_OUTCOME" {
		t.Fatal("message_id 813 must be MESSAGE_ID_S2C_SPARRING_OUTCOME")
	}
	if protocolv1.MessageId_name[818] != enumPrefix+"S2C_DUEL_OUTCOME" {
		t.Fatal("message_id 818 must be MESSAGE_ID_S2C_DUEL_OUTCOME")
	}
}
