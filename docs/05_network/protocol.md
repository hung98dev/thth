# Network Protocol
status: LOCKED

## Scope
Defines launch transport, framing, serialization, handshake, size limits, and connection-level behavior between Unity and the Go backend.

Decision: `../11_decisions/0008-client-network-transport-protocol.md`.

## Transports
Launch uses:
- HTTPS over TLS for authentication/bootstrap and non-realtime control-plane requests.
- One long-lived secure WebSocket (`wss`) connection for realtime gameplay.

Do not run a second UDP/QUIC gameplay path at launch. A transport change requires measured evidence and a new ADR.

## Serialization
Realtime gameplay payloads use Protocol Buffers with generated C# and Go types. Exact `protoc`, C# runtime, Go runtime, Go generator, and WebSocket-library versions are canonical in `../00_context/technology_versions.md`; generated outputs must be reproducible from those pins.

Rules:
- no JSON on the gameplay hot path,
- unknown additive protobuf fields are ignored,
- stable message IDs are never derived from localized/display strings,
- generated schemas are checked into/build-generated deterministically,
- floating-point fields are not used for persistent money/count identity.

## Application Envelope
Every gameplay frame resolves:
```text
protocol_major
protocol_minor
message_id
session_epoch
client_seq or server_seq
correlation_id when request/response applies
payload
```

The server validates envelope fields before payload dispatch.

`session_epoch` must match current authenticated session authority. Older epochs are rejected.

## Message Framing
One WebSocket binary message contains exactly one application envelope.

Launch limits:
```text
hard inbound frame limit      = 64 KiB
hard outbound frame limit     = 256 KiB
normal command target         <= 4 KiB
normal realtime update target <= 32 KiB
```

Payloads exceeding normal targets require an explicit message definition and review. Hard-limit violation closes the connection with a protocol error.

Large static/content assets are not delivered over gameplay WebSocket.

## Compression
Realtime WebSocket per-message compression is disabled at launch.

Bandwidth is controlled with:
- AOI filtering,
- state deltas,
- bounded snapshot rate,
- compact protobuf payloads,
- removal of redundant fields.

Do not trade tick latency/CPU predictability for generic compression without profiling.

## Handshake
Connection sequence:
1. client completes HTTPS authentication/bootstrap,
2. bootstrap returns short-lived gameplay connection credential plus endpoint and required protocol/content compatibility,
3. client opens `wss`,
4. client sends `C2S_HELLO` (message_id 1),
5. server validates credential, protocol version, client build/content compatibility, and session replacement rules,
6. server sends `S2C_HELLO_OK` (message_id 2) with session epoch and authoritative connection parameters,
7. character/session attach follows,
8. realtime messages become legal only after attach succeeds.

Gameplay credentials are single-purpose, short-lived, and are never database credentials.

## Sequence Semantics
Transport order does not replace application validation.

Per connection:
- `client_seq` is monotonic for client gameplay intents,
- duplicate/older sequence is rejected,
- movement-state intents (`C2S_INPUT_STATE`, delivery class `REPLACEABLE_STATE`) may be coalesced before simulation,
- movement-edge intents (`C2S_MOVEMENT_EDGE`, delivery class `DISCRETE_INTENT`) are **never coalesced or merged**; each edge is validated individually to preserve the onset signal required by timing-sensitive mechanics,
- discrete actions remain distinct,
- `server_seq` is monotonic for outbound authoritative messages,
- request/response mutations use stable operation/correlation identity where retry can occur.

Sequence counters reset only with a new session epoch/connection contract.

## Heartbeat
Application heartbeat exists even though WebSocket/TCP has transport keepalive.

Default:
```text
heartbeat interval = 5s
connection considered lost after 15s without valid traffic/heartbeat
```

Platform/network tuning may change these values through runtime config without changing authority semantics.

Heartbeat fields (used for RTT and Just Guard latency compensation, `../01_gameplay/combat.md`):
```text
C2S_HEARTBEAT (4): client_mono_ms uint64, echo_server_ms uint64 (0 if none received yet)
S2C_HEARTBEAT (5): server_ms uint64
RTT sample = server receive time of C2S_HEARTBEAT - echo_server_ms (when echo_server_ms != 0)
```

## TLS
Public client traffic must use TLS. Plaintext gameplay connections are not allowed outside isolated local development.

TLS termination may occur at an Edge/load-balancer layer, but authenticated session identity and trusted forwarding metadata must be cryptographically/operationally protected between edge and backend.

## Connection Backpressure
Each connection has bounded inbound/outbound queues.

When outbound queue is saturated:
1. obsolete replaceable state updates may be superseded,
2. required authoritative events/results are never silently dropped,
3. the connection is marked unhealthy and closed if it cannot keep up.

The server never buffers unbounded history for a slow client.

## Forbidden
- client-supplied authoritative position/damage/reward result,
- unbounded frame/body allocation,
- reflection-based dynamic message dispatch from arbitrary client type names,
- gameplay credential reuse as a general API token,
- direct database access from Unity,
- silent protocol fallback to an older incompatible schema.

## Invariants
```text
launch realtime transport = secure WebSocket
serialization = Protocol Buffers
one WS binary message = one application envelope
realtime per-message compression = disabled
inbound hard frame <= 64 KiB
session epoch is mandatory after authentication
client result is never authority
```
