---
description: Unity client rules — C# 9.0, asmdefs, authority boundary, hot-path hygiene
trigger: glob
globs:
  - "client/**/*.cs"
  - "client/**/*.asmdef"
  - "client/**/*.unity"
  - "client/**/*.prefab"
  - "client/**/*.asset"
  - "client/**/*.meta"
---

# Unity Client Rules

Canonical: `engineering_conventions.md` §2, `04_architecture/client.md`. Unity `6000.6.1f1`, C# 9.0, .NET Standard 2.1, IL2CPP release builds.

## Style

- Allman braces, 4 spaces. PascalCase types/methods, camelCase locals/params, `_camelCase` private fields. C# 9.0 language level — do not use newer syntax.

## Assemblies (acyclic)

```text
ThinhThan.Protocol   generated protobuf only, references nothing
ThinhThan.Core       math, IDs, text, pure domain models
ThinhThan.Net        WSS session, transport, serialization
ThinhThan.Systems    gameplay presentation, interpolation, controllers
ThinhThan.UI         HUD, menus, input overlays
ThinhThan.Tests.*    EditMode / PlayMode
```

Never create an asmdef dependency cycle. `Assets/Scripts/Protocol/` is generated — never hand-edit.

## Authority boundary (hard rule)

Client sends **intent**, never results. Never let client code decide: position legality, damage/heal/shield, hit confirmation, cooldowns, RNG, loot, balances, quest completion, trade/auction ownership, PvP rating, content activation. Predicted/presentation values are provisional until the server event arrives; Just Guard/clutch/reflect/lifesteal/absorb render only after `S2C_COMBAT_EVENT`. Scene state is not persistent authority.

## Layering

```text
Input/UI -> gameplay intent -> network session -> replicated state -> presentation
```

MonoBehaviours bridge Unity lifecycle and bind scene objects — no God Objects, no SQL-shaped or persistence-mutating constructs, no direct socket calls from UI. Business logic lives in plain C# classes where reasonable. Pair event subscriptions and resource handles with lifecycle cleanup; no fire-and-forget async work without cancellation/ownership.

## Hot path (Update/FixedUpdate/network dispatch)

- `GC.Alloc = 0` per frame: no LINQ, boxing, string concat, or collection churn.
- Cache component references; never `GameObject.Find`/`FindObjectOfType`/`Camera.main` per frame.
- Pool projectiles, floating text, VFX. No Instantiate/Destroy spam.

## Serialization & assets

- Never rename a serialized field without `[FormerlySerializedAs]`; never break prefab/scene references.
- ScriptableObjects for static/shared config — not mutable global state.
- Do not touch scenes/prefabs outside task scope; avoid unrelated YAML churn.
- Addressables keys are stable, non-localized presentation IDs; no `Resources`/raw-AssetBundle shortcuts.
- Localization: authored text via Unity Localization keys, `vi-VN` + `en-US` required; gameplay identity never parses localized strings.

## Input

- Input System `1.20.0` only. Movement edges send `C2S_MOVEMENT_EDGE` (108, `DISCRETE_INTENT`) with `PRESS | RELEASE | FLIP` — no `STOP` exists; `client_mono_ms` is advisory, server clamps ≤80ms.

## Tests

- EditMode for pure logic/data/validation; PlayMode for WSS session flow and component lifecycle. Don't use PlayMode where EditMode suffices.
