// Golden binary fixtures: canonical populated instances serialized with
// deterministic encoding into proto/testdata/golden/<id>_<NAME>.bin plus one
// envelope.bin. The C# EditMode suite decodes the same files for parity.
package protocol_test

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	protocolv1 "thinhthan/internal/protocol/v1"
)

// uuid is a fixed 16-byte fixture value; distinct inputs keep field-level
// decode checks honest.
func uuid(t *testing.T, hexStr string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(strings.ReplaceAll(hexStr, "-", ""))
	if err != nil || len(raw) != 16 {
		t.Fatalf("bad fixture uuid %q", hexStr)
	}
	return raw
}

// goldenSamples covers every proto file and every field-shape kind used on
// the wire (scalars, enums, nested, repeated, optional, oneof, bytes).
var goldenSamples = map[string]func(t *testing.T) proto.Message{
	"C2S_HELLO": func(t *testing.T) proto.Message {
		return &protocolv1.C2S_HELLO{
			Credential:      &protocolv1.C2S_HELLO_GameplayTicket{GameplayTicket: "ticket-abc"},
			ClientBuild:     "client-0.1.0",
			Platform:        protocolv1.SessionPlatform_SESSION_PLATFORM_WINDOWS,
			ContentRevision: "rev-1",
			DeviceId:        uuid(t, "00112233-4455-6677-8899-aabbccddeeff"),
			Locale:          "vi-VN",
		}
	},
	"S2C_HELLO_OK": func(t *testing.T) proto.Message {
		return &protocolv1.S2C_HELLO_OK{
			SessionId:              uuid(t, "c76fa3f6-7143-5e62-aeb3-07dd23e58771"),
			SessionEpoch:           9,
			AccountId:              uuid(t, "11111111-2222-3333-4444-555555555555"),
			ServerTimeMs:           1_758_000_000_000,
			HeartbeatIntervalMs:    5000,
			ConnectionTimeoutMs:    15000,
			ResumeCredential:       "resume-cred",
			ResumeExpiresAtMs:      1_758_000_600_000,
			ProtocolMinor:          0,
			ContentRevision:        "rev-1",
			PendingDeletion:        false,
			ResumeRotateIntervalMs: 300000,
		}
	},
	"S2C_ERROR": func(t *testing.T) proto.Message {
		return &protocolv1.S2C_ERROR{
			ErrorCode:      protocolv1.ErrorCode_ERROR_CODE_RATE_LIMITED,
			Retryability:   protocolv1.Retryability_RETRYABILITY_BACKOFF,
			RetryAfterMs:   5000,
			SafeMessageKey: "loc.error.rate_limited",
			CloseAfter:     false,
		}
	},
	"C2S_INPUT_STATE": func(t *testing.T) proto.Message {
		return &protocolv1.C2S_INPUT_STATE{InputFlags: 5, ClientMonoMs: 100}
	},
	"C2S_INTERACT": func(t *testing.T) proto.Message {
		return &protocolv1.C2S_INTERACT{
			InteractKind:                protocolv1.InteractKind_INTERACT_KIND_COOK,
			TargetId:                    "cooking_hearth.lang_cau_thon",
			OperationId:                 uuid(t, "243f43cd-583e-5a98-9083-d8a73b8bd70b"),
			RecipeId:                    "recipe.food.ca_chep_nuong",
			PublicBossSpawnGenerationId: uuid(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
		}
	},
	"S2C_MOVEMENT_CORRECTION": func(t *testing.T) proto.Message {
		return &protocolv1.S2C_MOVEMENT_CORRECTION{
			LastProcessedClientSeq: 7,
			ServerTick:             12345,
			XMm:                    1000,
			YMm:                    -500,
			VxMmS:                  150,
			VyMmS:                  -75,
			Facing:                 protocolv1.Facing_FACING_LEFT,
			MovementState:          protocolv1.MovementState_MOVEMENT_STATE_RUN,
			Reason:                 protocolv1.MovementCorrectionReason_MOVEMENT_CORRECTION_REASON_ILLEGAL_MOVE,
		}
	},
	"S2C_STATE_DELTA": func(t *testing.T) proto.Message {
		return &protocolv1.S2C_STATE_DELTA{
			BaselineId: 42,
			ServerTick: 21,
			SelfAck: &protocolv1.SelfAck{
				LastProcessedClientSeq: 7,
				XMm:                    1000,
				YMm:                    -500,
				VxMmS:                  150,
				VyMmS:                  -75,
				MovementState:          protocolv1.MovementState_MOVEMENT_STATE_RUN,
			},
			Entities: []*protocolv1.EntityDelta{{
				EntityId:    2,
				Level:       proto.Uint32(12),
				XMm:         proto.Int32(1100),
				Hp:          proto.Int64(800),
				MaxHp:       proto.Int64(1000),
				Flags:       proto.Uint32(uint32(protocolv1.EntityFlag_ENTITY_FLAG_IN_COMBAT)),
				EncounterId: proto.Uint64(7),
				Statuses: &protocolv1.StatusList{Entries: []*protocolv1.EntityStatusView{{
					EffectId:       "effect.doc.burn",
					SourceEntityId: 9,
					Stacks:         2,
					ExpiresAtTick:  99999,
				}}},
				EquippedCosmetics: &protocolv1.CosmeticList{Entries: []*protocolv1.EquippedCosmetic{{
					Slot:       protocolv1.CosmeticSlot_COSMETIC_SLOT_WEAPON_TRAIL,
					CosmeticId: "cosmetic.weapon.rong_lua",
				}}},
			}},
		}
	},
	"S2C_COMBAT_EVENT": func(t *testing.T) proto.Message {
		return &protocolv1.S2C_COMBAT_EVENT{
			EventId:               9001,
			ServerTick:            21,
			ActionInstanceId:      55,
			SourceEntityId:        2,
			TargetEntityId:        8,
			SkillId:               "skill.kim.thien_ma_tram",
			ResultKind:            protocolv1.CombatResultKind_COMBAT_RESULT_KIND_DAMAGE,
			Outcome:               protocolv1.CombatOutcome_COMBAT_OUTCOME_HIT,
			DamageElement:         protocolv1.DamageElement_DAMAGE_ELEMENT_KIM,
			PostMitigationDamage:  250,
			ShieldAbsorbed:        10,
			HpDamage:              240,
			TargetHpAfter:         560,
			TargetShieldAfter:     0,
			JustGuardWindow:       true,
			JustGuardTriggered:    true,
			JustGuardHint:         true,
			BeastPassive2Success:  true,
			ReflectDamageInstance: proto.Int64(55),
			LifestealHealAmount:   proto.Int64(21),
			AbsorbShieldAmount:    proto.Int64(34),
		}
	},
	"C2S_INVENTORY_MUTATE": func(t *testing.T) proto.Message {
		return &protocolv1.C2S_INVENTORY_MUTATE{
			OperationId:    uuid(t, "01020304-0506-0708-0909-aabbccddeeff"),
			Op:             protocolv1.InventoryMutateOp_INVENTORY_MUTATE_OP_SPLIT,
			ItemInstanceId: uuid(t, "99998888-7777-6666-5555-444433332222"),
			ToSlot:         7,
			Quantity:       3,
		}
	},
	"S2C_INVENTORY_RESULT": func(t *testing.T) proto.Message {
		return &protocolv1.S2C_INVENTORY_RESULT{
			Result: &protocolv1.OperationResult{
				OperationId: uuid(t, "01020304-0506-0708-0909-aabbccddeeff"),
				Status:      protocolv1.ResultStatus_RESULT_STATUS_SUCCESS,
			},
			Op:                protocolv1.InventoryMutateOp_INVENTORY_MUTATE_OP_SPLIT,
			InventoryRevision: 77,
			ChangedSlots: []*protocolv1.InventorySlotView{{
				Slot:           3,
				ItemInstanceId: uuid(t, "99998888-7777-6666-5555-444433332222"),
				ItemId:         "item.material.sat_tinh",
				Quantity:       2,
			}},
		}
	},
	"S2C_QUEST_UPDATE": func(t *testing.T) proto.Message {
		return &protocolv1.S2C_QUEST_UPDATE{
			QuestId: "quest.main.act1.thu_linh",
			State:   protocolv1.QuestState_QUEST_STATE_ACTIVE,
			Objectives: []*protocolv1.QuestObjectiveView{{
				ObjectiveIndex: 0, Current: 2, Required: 5,
			}},
			OperationId: uuid(t, "5e5e5e5e-1111-2222-3333-444455556666"),
			Status:      protocolv1.ResultStatus_RESULT_STATUS_SUCCESS,
			ExpGained:   120,
			Granted: []*protocolv1.ItemGrant{{
				ItemInstanceId: uuid(t, "aaaa0000-bbbb-1111-cccc-2222dddd3333"),
				ItemId:         "item.potion.hoi_mau",
				Quantity:       2,
			}},
			CurrencyDelta: []*protocolv1.CurrencyDelta{{
				CurrencyId: "currency.common",
				Amount:     500,
			}},
		}
	},
	"S2C_GUILD_STORAGE_STATE": func(t *testing.T) proto.Message {
		return &protocolv1.S2C_GUILD_STORAGE_STATE{
			GuildId:         uuid(t, "77777777-8888-9999-aaaa-bbbbbbbbbbbb"),
			StorageRevision: 12,
			Items: []*protocolv1.GuildStorageItem{{
				ItemInstanceId: uuid(t, "abcdef01-2345-6789-abcd-ef0123456789"),
				ItemId:         "item.material.huyet_ngoc",
				Quantity:       15,
				Section:        protocolv1.GuildStorageSection_GUILD_STORAGE_SECTION_COMMON,
				DepositedBy:    uuid(t, "c76fa3f6-7143-5e62-aeb3-07dd23e58771"),
				DepositedAt:    1_758_000_000_000,
			}},
		}
	},
	"S2C_AUCTION_LIST_RESULT": func(t *testing.T) proto.Message {
		return &protocolv1.S2C_AUCTION_LIST_RESULT{
			Result: &protocolv1.OperationResult{
				OperationId: uuid(t, "00112233-4455-6677-8899-aabbccddeeff"),
				Status:      protocolv1.ResultStatus_RESULT_STATUS_SUCCESS,
			},
			ListingId:  uuid(t, "fedcba98-7654-3210-fedc-ba9876543210"),
			ListingFee: 1000,
			ExpiresAt:  1_758_086_400_000,
		}
	},
	"S2C_MATCH_STATE": func(t *testing.T) proto.Message {
		return &protocolv1.S2C_MATCH_STATE{
			MatchId:         uuid(t, "13572468-ace0-2468-ace0-13579bdf2468"),
			PvpModeId:       "pvp.duel_1v1",
			State:           protocolv1.MatchLifecycle_MATCH_LIFECYCLE_ACTIVE,
			StateDeadlineMs: 1_758_000_030_000,
			RoundNumber:     2,
			Teams: []*protocolv1.MatchTeam{{
				TeamIndex: 0,
				Score:     1,
				Members: []*protocolv1.MatchTeamMember{{
					CharacterId: uuid(t, "c76fa3f6-7143-5e62-aeb3-07dd23e58771"),
					DisplayName: "KimSatThu",
					ClassId:     "class.kim",
					Ready:       true,
					Connected:   true,
				}},
			}},
			Result: protocolv1.MatchResult_MATCH_RESULT_WIN,
		}
	},
	"S2C_SPARRING_OUTCOME": func(t *testing.T) proto.Message {
		return &protocolv1.S2C_SPARRING_OUTCOME{
			ChallengeId: uuid(t, "24683579-bdf1-3579-bdf1-24680ace1357"),
			Outcome:     protocolv1.SparringOutcome_SPARRING_OUTCOME_ACCEPTED,
		}
	},
}

// goldenDir is proto/testdata/golden under the repo root.
func goldenDir(t *testing.T) string {
	return filepath.Join(repoRoot(t), "proto", "testdata", "golden")
}

// writeGoldenFixtures serializes the sample set with deterministic encoding.
func writeGoldenFixtures(t *testing.T, dir string) {
	t.Helper()
	names := make([]string, 0, len(goldenSamples))
	for name := range goldenSamples {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		id := protocolv1.MessageId_value[enumPrefix+name]
		raw, err := proto.MarshalOptions{Deterministic: true}.Marshal(goldenSamples[name](t))
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		file := filepath.Join(dir, strconv.Itoa(int(id))+"_"+name+".bin")
		if err := os.WriteFile(file, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	env := &protocolv1.Envelope{
		ProtocolMajor: 1,
		MessageId:     103,
		SessionEpoch:  9,
		ClientSeq:     42,
		ServerSeq:     0,
		CorrelationId: 0,
		Payload:       mustMarshal(t, goldenSamples["C2S_INTERACT"](t)),
	}
	raw, err := proto.MarshalOptions{Deterministic: true}.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "envelope.bin"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Log("wrote", len(names)+1, "fixtures to", dir)
}
