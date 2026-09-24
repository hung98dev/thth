# Reward Claims
status: LOCKED

## Scope
Defines one minimal persistent overflow path for rewards already earned but not deliverable because inventory/currency capacity is unavailable. This is not player mail.

## Principle
A reward must never be silently deleted, duplicated, rerolled, or left dependent on a despawning runtime entity.

## Reward Slots
A reward source may be one indivisible bundle or may explicitly declare independently settleable slots with stable `reward_slot` IDs.

Typical launch drop-table slots are intentionally independent:
```text
currency_common
regional_material
recovery_hp
recovery_mp
equipment
soul
support_item
cosmetic_material
```

An independent slot is committed/claimed separately from sibling slots. Therefore a capped currency slot cannot block an already-earned item/Soul/equipment slot from delivery.

Random selection is finalized before the slot commits. A retry or later claim never rerolls item ID, equipment slot, Soul ID, quantity, or weighted choice.

## Claim Creation
If an authoritative reward slot has already been earned and its complete all-or-nothing delivery cannot fit, create one persistent:
```text
reward_claim_id
```
with:
```text
owner_character_id
source_type
source_reference
reward_slot
finalized_reward_payload
created_at
state
```

For an item/equipment reward, the pending claim stores the complete immutable item-creation payload rather than a normal owned `item_instance_id`. The payload contains every finalized value needed for exact later delivery, including item ID, quantity, effective binding/source override, generated roll/enhancement/provenance state when applicable, and required content revision. See ADR-0012.
States:
```text
PENDING -> CLAIMING -> CLAIMED
PENDING -> EXPIRED only when the source explicitly allows expiry
```
Initial gameplay reward claims do not expire.

## Eligible Sources
Monster, boss, dungeon, quest, world event, PvP/Guild reward, administrative compensation, and other explicit reward flows may create claims. A source may instead prevalidate capacity and avoid claim creation when nothing has yet been earned.

## Delivery
Claiming revalidates ownership and complete destination capacity for that claim/slot. Delivery is atomic:
```text
PENDING + capacity -> exact delivery/materialization + CLAIMED
failure -> no destination mutation + remains PENDING
```

For item/equipment claims, successful delivery creates or merges the exact finalized value into the character inventory in the same transaction that marks the claim CLAIMED. Claim failure creates no materialized item.
One claim is never partially delivered. Sibling reward slots may settle independently only when the source definition explicitly authored them as independent slots.

## Currency Overflow
A reward claim may hold an earned currency credit that would exceed a canonical balance cap. Claim succeeds only when the resulting balance is within cap. Currency is never silently clamped.

High-frequency repeatable rewards must not create one PENDING row per capped currency tick. Compatible currency-only overflow may consolidate into one aggregate pending claim per:
```text
owner_character_id + currency_id + source_family
```

Aggregate currency claim requirements:
- every contributed credit keeps its original `source_reward_operation_id + reward_slot` idempotency key in an append-only contribution ledger,
- retrying a contributed operation adds zero additional amount,
- the aggregate amount is not spendable and is not a fourth currency,
- item/Soul/equipment sibling slots continue settling independently,
- claiming the aggregate credit remains all-or-nothing against the canonical currency cap.

If safe consolidation is unavailable and creating another claim would exceed the claim cap, the reward-producing action must fail before the reward is earned where possible; an already-earned reward may never be deleted.

## Not Mail
Reward Claims have:
- no player-to-player sending
- no attachments authored by players
- no chat/message body
- no trading
- no COD
They are a recovery/settlement mechanism only.

## Capacity / Abuse
A character may have at most `100` active PENDING non-aggregate claims. Aggregate currency overflow claims count as one claim each regardless of contribution count.

Before an activity creates a new non-aggregate claim beyond the cap, reward-producing entry/settlement must fail safely or compatible claims must consolidate according to an explicitly supported rule. Never delete oldest rewards automatically.

## Idempotency
Canonical creation/delivery key:
```text
source_reward_operation_id + owner_character_id + reward_slot
```
One earned reward slot creates at most one direct delivery or claim contribution. Claiming uses `reward_claim_operation_id`.

## Persistence
Claims and aggregate contribution ledgers survive disconnect/restart. Runtime monster/boss/dungeon cleanup never removes a pending claim.

## UI
UI may expose a simple `Unclaimed Rewards` panel and claim button. Aggregate currency overflow may be displayed as one row. This must not become a social mailbox.

## Invariants
```text
earned reward cannot be silently destroyed
independent reward slots do not block each other
committed random reward never rerolls
pending item claim has no normal item_instance_id until successful materialization
claim is character-owned
claim is not mail
claim delivery is atomic
currency overflow aggregate is not spendable
pending claims survive source cleanup/restart
PENDING non-aggregate claim cap = 100
one source operation + owner + reward_slot -> at most one credit/delivery
```
