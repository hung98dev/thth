# Validation
status: LOCKED

## Scope
Defines mandatory server-side validation for all Unity/network/admin/worker input before authoritative state mutation.

## Trust Boundary
Everything outside the owning authoritative component is untrusted until validated.

This includes:
- Unity payloads,
- reconnect/resume data,
- inter-process messages,
- background job payloads,
- admin/tooling requests,
- values loaded from stale caches.

## Validation Layers
Every inbound operation applies applicable layers in this order:
1. envelope/schema/size,
2. authentication/session,
3. rate limit,
4. ownership/scope,
5. current simulation/durable revision,
6. state-machine legality,
7. spatial/range/collision,
8. resource/currency/item preconditions,
9. idempotency/sequence,
10. transaction/commit constraints.

Failure stops before mutation.

## Numeric Validation
Reject:
- NaN/Inf,
- out-of-range enums,
- negative quantities where not explicitly legal,
- overflow-prone arithmetic,
- non-canonical IDs,
- coordinates outside map bounds/tolerance,
- client-provided monetary totals/prices where server can derive them.

Use checked integer arithmetic for currencies/counts/costs.

## String Validation
Canonical Unicode processing for player-authored text is `../06_data/text.md`.

Bound and validate:
- valid UTF-8 + hard byte length,
- authoritative UAX #29 grapheme count where user-facing limits use graphemes,
- trim/NFC rules,
- server-computed case-folded `name_key` where uniqueness applies,
- allowed/forbidden control characters.

Do not use rune count, client-reported length, PostgreSQL locale-dependent LOWER(), or a different Unicode library as authoritative equality/length.

Escape/parameterize all DB/query/log contexts. Never construct SQL from raw client string concatenation.

## Movement
Server validates movement from accepted input + authoritative previous state + collision/physics rules.

Reject/correct impossible:
- speed,
- displacement,
- portal bypass,
- blocked geometry crossing,
- movement while hard-controlled,
- stale ownership/session input.

Client transform is diagnostic/prediction context only.

## Combat
Validate:
- skill learned/equipped,
- cooldown/resource,
- action state,
- target ownership/hostility,
- range/geometry,
- status/control restrictions,
- content revision,
- hit/effect on server.

Client never supplies accepted damage/heal/crit/drop result.

## Boss Chest Claim (Gilded Chest)
Boss chest interaction is a two-step sequence: a player interacts with the Gilded Chest entity after a boss is defeated. Eligibility was checked at boss-kill time (participation window), but network lag and reconnect mean a player's session may resume after that window closes.

**Re-validation requirement:** At commit time of the chest interaction, the server re-validates the character's contribution record for the **current boss spawn generation** (identified by `public_boss_spawn_generation_id`). The contribution record must exist and must not be expired. Fields verified at commit:
```text
contribution_record.character_id  = interacting character
contribution_record.boss_spawn_id = current boss spawn generation (not a prior spawn)
contribution_record.state         = VALID (not expired / superseded)
```
If no valid contribution record exists at commit time, the interaction is rejected with `CHEST_ELIGIBILITY_INVALID` and no loot table is rolled. No partial loot, no consolation drop, no pending credit is created.

Rationale: without commit-time re-validation, a player can tag a boss for minimum contribution, disconnect, reconnect within the 3-minute chest window, and receive a full loot roll without having meaningfully participated in the kill.

## Economy / Ownership
Every value-changing operation validates canonical PostgreSQL ownership and state inside the transaction where race is possible.

Examples:
- inventory instance owner/container,
- equipment/loadout legality,
- currency balance/cap,
- enhancement inputs/current level,
- trade/Auction escrow/listing state,
- Reward Claim state,
- guild permission/membership revision.

Pre-check outside the transaction may improve UX but cannot replace commit-time validation.

## IDs
Persistent/runtime IDs are parsed against explicit type/namespace.

Do not accept arbitrary client-selected database primary keys without verifying account/character/entity ownership and state.

## IAP Receipt Verification (Outbound Trust Boundary)
The IAP path is an outbound call to a platform payment provider. It is not an inbound gameplay message and not a WebSocket frame, but it is a trust boundary that must be explicitly secured.

**What is validated before grant:**
1. Receipt schema: the `platform_receipt` field is a non-empty opaque token within the allowed byte length; structural pre-checks are platform-specific (app store vs. regional gateway).
2. Platform API response: the server calls the platform's verification endpoint and inspects the canonical response fields (status, product_id, order_id, purchase_time). The game server never interprets a receipt locally without a positive platform confirmation.
3. Product identity: the `product_id` returned by the platform must exactly match the `product_id` in the pending entitlement record. Mismatches are rejected with `IAP_PRODUCT_MISMATCH`.
4. Account binding: the verified receipt must not be already associated with a different `account_id` (see cross-account check in `monetization.md`).

**Provider timeout / error handling:**
- A network timeout or HTTP 5xx from the platform provider is a transient failure. The entitlement remains in `PENDING` state. No grant is committed.
- The operation is retried with exponential backoff (max 3 attempts, cap 30 s). Each retry reuses the same stable `entitlement_id` as the idempotency key; the platform endpoint's own idempotency ensures a second verification call does not double-charge the player.
- After retry exhaustion, the entitlement stays `PENDING` and the client is informed to retry later. Support tooling can manually re-trigger verification.

**Retry double-grant safety:**
- The grant is committed inside a database transaction that checks `grant_state = PENDING` at commit time. A concurrent duplicate verification attempt for the same `entitlement_id` that arrives while one transaction is in flight will see the state already committed and return the existing record without a second grant.
- The platform receipt is stored with the entitlement record before the first outbound call. On retry, the server confirms the stored receipt matches before calling the platform again.

**Ambiguous / unverified responses:**
- A grant is **never** committed on a provider response that is absent, malformed, ambiguous (non-terminal status), or carries a verification status other than a definitive confirmed-purchase signal.
- `PENDING` is the safe default; only a clear positive confirmation transitions to `GRANTED`.

## Admin / Worker
Internal requests require the same domain invariants as public requests.

Admin privilege may authorize an exceptional operation, but the operation remains typed, audited, idempotent where applicable, and cannot bypass storage integrity.

## Failure
Validation failure:
- does not partially mutate state,
- returns stable safe error,
- may emit security/anomaly signal,
- does not expose internal rule thresholds unnecessarily.

## Invariants
- all client data is intent, not result,
- commit-time ownership/value validation is mandatory,
- malformed numbers/strings are rejected before domain use,
- SQL is parameterized,
- spatial/combat outcomes are server-owned,
- internal tooling does not bypass domain invariants,
- IAP grant committed only on definitive positive platform confirmation; PENDING is the safe default,
- IAP retry uses same entitlement_id; commit-time PENDING check prevents double-grant,
- boss chest claim re-validates contribution_record for current public_boss_spawn_generation_id at commit time; no valid record -> CHEST_ELIGIBILITY_INVALID, no loot roll.
