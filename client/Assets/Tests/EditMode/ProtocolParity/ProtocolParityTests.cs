// Protocol parity tests: every registered MessageId resolves to a same-named
// generated class, and every Go-authored golden binary fixture round-trips
// without losing field values on the C# side. Protobuf field ordering is not a
// cross-runtime wire guarantee, so parity compares message semantics, not bytes.
// Fixtures are shared with the Go suite at proto/testdata/golden/.
//
// Names resolve through [pbr::OriginalName] attributes and generated
// MessageDescriptors — never Enum.ToString(), which yields PascalCased C#
// member names (MESSAGE_ID_C2S_HELLO -> MessageId.C2SHello).
using System;
using System.IO;
using System.Reflection;
using Google.Protobuf;
using Google.Protobuf.Reflection;
using NUnit.Framework;
using ThinhThan.Protocol.V1;

namespace ThinhThan.Tests.EditMode.ProtocolParity
{
    public class ProtocolParityTests
    {
        private const int ExpectedMessageCount = 209;
        private const string ProtoPrefix = "MESSAGE_ID_";
        private const string MessageNamespace = "ThinhThan.Protocol.V1";

        private static string GoldenDir()
        {
            // <repo>/client/Assets -> <repo>/proto/testdata/golden
            var assets = Path.GetFullPath(UnityEngine.Application.dataPath);
            var root = Path.GetFullPath(Path.Combine(assets, "..", ".."));
            return Path.Combine(root, "proto", "testdata", "golden");
        }

        // Proto enum name (e.g. MESSAGE_ID_C2S_HELLO) for a numeric value.
        private static string ProtoEnumName(int value)
        {
            var member = typeof(MessageId).GetMember(
                Enum.GetName(typeof(MessageId), value))[0];
            var attr = (OriginalNameAttribute)member.GetCustomAttribute(
                typeof(OriginalNameAttribute));
            return attr.Name;
        }

        // Generated message class for a proto enum name.
        private static Type ResolveMessageType(string protoEnumName)
        {
            Assert.IsTrue(protoEnumName.StartsWith(ProtoPrefix),
                $"enum name {protoEnumName} lacks the {ProtoPrefix} prefix");
            var messageName = protoEnumName.Substring(ProtoPrefix.Length);
            return typeof(MessageId).Assembly.GetType(MessageNamespace + "." + messageName);
        }

        private static IMessage Parse(Type type, ByteString data)
        {
            var parser = (MessageParser)type.GetProperty("Parser",
                BindingFlags.Public | BindingFlags.Static).GetValue(null);
            return parser.ParseFrom(data);
        }

        [Test]
        public void GeneratedRegistry_Covers_All_Message_Ids()
        {
            var members = typeof(MessageId).GetFields(BindingFlags.Public | BindingFlags.Static);
            Assert.AreEqual(ExpectedMessageCount + 1, members.Length,
                "MessageId enum must hold exactly the spec registry plus UNSPECIFIED");

            foreach (var member in members)
            {
                var value = (int)member.GetValue(null);
                var protoName = ProtoEnumName(value);
                Assert.IsTrue(protoName.StartsWith(ProtoPrefix),
                    $"enum member {member.Name} lacks {ProtoPrefix} OriginalName");
                if (value == 0)
                {
                    Assert.AreEqual(ProtoPrefix + "UNSPECIFIED", protoName);
                    continue;
                }
                Assert.IsNotNull(ResolveMessageType(protoName),
                    $"no generated class for {protoName}");
            }

            Assert.IsFalse(Enum.IsDefined(typeof(MessageId), 737),
                "message_id 737 is retired and must stay unassigned");
        }

        [Test]
        public void Golden_Fixtures_Decode_And_RoundTrip_Semantically()
        {
            var dir = GoldenDir();
            Assert.IsTrue(Directory.Exists(dir), $"golden dir missing: {dir}");

            var count = 0;
            foreach (var file in Directory.GetFiles(dir, "*.bin"))
            {
                var raw = File.ReadAllBytes(file);
                var stem = Path.GetFileNameWithoutExtension(file);

                if (stem == "envelope")
                {
                    var env = Envelope.Parser.ParseFrom(raw);
                    Assert.IsTrue(Enum.IsDefined(typeof(MessageId), (int)env.MessageId),
                        $"envelope.bin carries unknown message_id {env.MessageId}");
                    var innerName = ProtoEnumName((int)env.MessageId);
                    var innerType = ResolveMessageType(innerName);
                    Assert.IsNotNull(innerType, $"envelope.bin: no class for {innerName}");
                    Assert.IsNotNull(Parse(innerType, env.Payload));
                    var roundTrippedEnvelope = Envelope.Parser.ParseFrom(env.ToByteArray());
                    Assert.AreEqual(env, roundTrippedEnvelope,
                        "envelope.bin semantic round-trip differs");
                    count++;
                    continue;
                }

                var split = stem.IndexOf('_');
                Assert.Greater(split, 0, $"{stem}: fixture name must be <id>_<NAME>.bin");
                var id = int.Parse(stem.Substring(0, split));
                Assert.IsTrue(Enum.IsDefined(typeof(MessageId), id),
                    $"{stem}: id {id} not in registry");
                var protoEnumName = ProtoEnumName(id);
                Assert.AreEqual(protoEnumName.Substring(ProtoPrefix.Length),
                    stem.Substring(split + 1),
                    $"{stem}: id/name mismatch against registry");

                var type = ResolveMessageType(protoEnumName);
                Assert.IsNotNull(type, $"{stem}: no generated class");

                var msg = Parse(type, ByteString.CopyFrom(raw));
                var roundTripped = Parse(type, ByteString.CopyFrom(msg.ToByteArray()));
                Assert.AreEqual(msg, roundTripped,
                    $"{stem}: semantic round-trip differs");
                count++;
            }
            Assert.Greater(count, 0, "no golden fixtures found");
        }

        [Test]
        public void Golden_Combat_Event_Carries_Optional_Secondary_Results()
        {
            var raw = File.ReadAllBytes(Path.Combine(GoldenDir(), "304_S2C_COMBAT_EVENT.bin"));
            var evt = S2C_COMBAT_EVENT.Parser.ParseFrom(raw);
            Assert.IsTrue(evt.JustGuardWindow);
            Assert.IsTrue(evt.JustGuardTriggered);
            Assert.IsTrue(evt.JustGuardHint);
            Assert.IsTrue(evt.BeastPassive2Success);
            Assert.IsTrue(evt.HasReflectDamageInstance);
            Assert.AreEqual(55, evt.ReflectDamageInstance);
            Assert.AreEqual(21, evt.LifestealHealAmount);
            Assert.AreEqual(34, evt.AbsorbShieldAmount);
        }

        [Test]
        public void Golden_State_Delta_Carries_SelfAck_And_Wrappers()
        {
            var raw = File.ReadAllBytes(Path.Combine(GoldenDir(), "303_S2C_STATE_DELTA.bin"));
            var delta = S2C_STATE_DELTA.Parser.ParseFrom(raw);
            Assert.IsNotNull(delta.SelfAck, "303 must carry SelfAck (ADR-0069)");
            Assert.AreEqual(7UL, delta.SelfAck.LastProcessedClientSeq);
            Assert.AreEqual(1, delta.Entities.Count);
            var entity = delta.Entities[0];
            Assert.AreEqual(800L, entity.Hp);
            Assert.AreEqual(1, entity.Statuses.Entries.Count,
                "statuses use the StatusList wrapper, never optional repeated");
            Assert.AreEqual(1, entity.EquippedCosmetics.Entries.Count);
            Assert.AreEqual(CosmeticSlot.WeaponTrail,
                entity.EquippedCosmetics.Entries[0].Slot);
        }
    }
}
