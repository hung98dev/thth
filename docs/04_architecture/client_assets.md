# Client Asset Delivery
status: LOCKED

## Scope
Defines Unity presentation-asset packaging, Addressables grouping, compatibility, download/cache behavior, memory ownership, and failure handling.

Decision: `../11_decisions/0014-unity-addressables-asset-delivery.md`.
Exact package pin: `../00_context/technology_versions.md`.

# Canonical System
Launch uses:
```text
com.unity.addressables = 2.11.2
```

All non-trivial dynamically loaded presentation content goes through Addressables.

Do not build a parallel custom AssetBundle/download/cache framework.

# Authority Boundary
Addressables contains client presentation assets such as:
- sprites / sprite atlases,
- animations / animator assets,
- VFX prefabs/materials/shaders,
- audio,
- UI visuals,
- map visual scenes/prefabs,
- cosmetic presentation,
- presentation-only localization/media where compatible.

Addressables does **not** authorize:
- damage/stat values,
- item/currency/reward amounts,
- map access,
- collision legality,
- server spawn locations,
- quest completion,
- RNG results,
- owned entitlements,
- active content revision.

Canonical gameplay IDs/data remain under server/static-content contracts.

A client asset may reference a stable gameplay ID for lookup, but an Addressable key is never that ID's authority.

# Base Install
The shipped player build must contain enough local content to:
- boot,
- show legal/splash/start UI where required,
- authenticate/bootstrap,
- show version/update/download errors,
- reconnect/logout safely,
- load the minimum common UI/fonts/materials needed before remote content is available,
- verify and fetch the compatible Addressables catalog/content.

Do not require a remote asset download merely to display a fatal compatibility or login error.

# Grouping
Use coarse groups that align with actual lifetime/update boundaries rather than one bundle per asset.

Canonical logical grouping:
```text
bootstrap.local
shared.local
region.<region_id>
dungeon.<dungeon_id>
pvp.shared
cosmetic.shared / cosmetic.<pack_or_scope> when justified
audio.shared / regional audio only when bundle-size profiling justifies it
```

Rules:
- shared dependencies are extracted deliberately to avoid duplicate bundle copies,
- region/dungeon groups may be remote,
- frequently co-used assets should not be fragmented into hundreds of tiny bundles,
- giant all-game bundles that force unrelated downloads are also rejected,
- final bundle layout is validated by build-size/load profiling.

Do not derive gameplay access from group membership.

# Stable Asset Keys
Asset keys use ASCII lowercase deterministic names, for example:
```text
asset.map.lang_da.bo_ruong.scene
asset.monster.lang_da.ma_xo.prefab
asset.skill.kim.kiem_quang.vfx
asset.ui.inventory.icon
```

They are presentation IDs only.

Rules:
- do not use localized names as keys,
- do not use Unity GUID as public gameplay identity,
- moving/renaming an asset may change editor internals without changing a stable published asset key unless an intentional asset-key migration occurs,
- duplicate keys fail client-content validation.

# Catalog Identity / Compatibility
A client asset build records:
```text
client_build
asset_catalog_revision
addressables_build_id
platform
content_hash
compatible_protocol_major
compatible_content_schema_range
```

Server bootstrap returns the minimum compatible client build/content contract. Client resolves its own compatible asset catalog before entering gameplay.

Gameplay `content_revision` and `asset_catalog_revision` are not assumed to be numerically equal.

Compatibility mapping is explicit:
```text
client_build + asset_catalog_revision
  -> supported content_schema/version range
```

A presentation-only asset revision may support multiple gameplay content revisions when IDs/contracts remain compatible.

# Publish Order
For a release that requires new remote presentation assets:
1. build Addressables with the pinned Unity/package versions,
2. validate bundle/catalog hashes and dependency graph,
3. upload immutable bundles/catalog to the configured origin/CDN,
4. verify download from production-like endpoints,
5. deploy compatible backend/content/client build gates,
6. only then require the new catalog/build.

Never activate gameplay content that requires client presentation assets not yet available.

Never overwrite a previously published immutable bundle file with different bytes under the same identity.

# Remote URL
Remote base URL is environment configuration:
```text
dev
staging
production
```

No CDN vendor SDK is part of launch client architecture.

Use HTTPS. Client does not receive cloud storage credentials.

# Download / Cache
Addressables owns catalog/bundle download and local caching.

Client UI exposes:
- required download size before a large mandatory download when practical,
- progress,
- retry,
- insufficient-storage/network failure,
- cancel before entering gated content when safe.

Rules:
- completed cached bundles are reused only when catalog/hash says valid,
- corrupted/hash-invalid content is discarded/refetched,
- failed partial download does not become a valid cached bundle,
- retry is bounded/backed off; no tight loop,
- asset download failure never mutates server gameplay state.

# Destination Preload Gate
Before Unity presents/attaches to a destination map/instance, required destination presentation dependencies must be available.

Flow:
1. server authorizes the destination and sends `S2C_TRANSFER_PREPARE`,
2. client resolves/downloads required Addressables dependencies,
3. client sends `C2S_PRESENTATION_READY` for that `transfer_id`,
4. authoritative transfer/handoff follows `concurrency.md` / world contracts,
5. client loads presentation and receives fresh authoritative state/baseline.

The server does not trust a client-provided destination or asset list. Ready is accepted only for the current `transfer_id` and session/ownership epoch.

Timeout uses `TRANSFER_BUDGET_WORLD = 30s` or `TRANSFER_BUDGET_INSTANCE = 120s`. Failure is `TRANSFER_FAILED` plus source/checkpoint recovery; client never substitutes another map coordinate/state.

For very small already-cached groups, `C2S_PRESENTATION_READY` may follow immediately.


# Scene / Map Assets
Unity map scenes/prefabs are presentation/collision-source authoring assets, but server runtime geometry/anchors are compiled/validated server-side under `../06_data/config.md` (Map Geometry Compile) and world/content rules.


A remote Unity scene cannot redefine authoritative collision or spawn truth by itself.

Build validation cross-checks required stable map anchor/asset references against the authored map data before release.

# Memory / Handle Ownership
Every Addressables load has a lifecycle owner.

Rules:
- retain AsyncOperationHandle only while the owning screen/map/entity/shared cache needs it,
- release handles deterministically,
- pooled GameObjects reset replicated state/VFX before reuse,
- map transition unloads no-longer-needed region/dungeon assets except explicit shared/cache policy,
- low-memory/mobile path may evict non-required cached in-memory objects without changing downloaded-cache validity,
- no unbounded static dictionary of loaded UnityEngine.Object references.

Reference-count/leak tests are mandatory for repeated region/dungeon transitions.

# Resources Folder
General game content under Unity Resources folders is forbidden.

Allowed exception:
- tiny bootstrap asset that cannot reasonably be referenced directly/through local Addressables,
- documented in code,
- included in build-size audit.

Do not place maps, monsters, items, skill VFX, cosmetics, large audio, or content catalogs in Resources as a shortcut.

# Direct AssetBundle API
Gameplay/UI feature code does not call raw AssetBundle download/load APIs.

If Addressables internally uses AssetBundles, that remains an implementation detail of the selected Unity package.

# Security
Remote assets are untrusted presentation inputs until validated by the Addressables/catalog/hash path.

Do not:
- execute downloaded native code/scripts,
- load arbitrary user-supplied bundle URLs,
- embed storage write credentials,
- treat remote ScriptableObject values as authoritative gameplay state.

Remote content origins are allowlisted by environment configuration.

# Build / CI Validation
Client-content build fails on:
- duplicate Addressable key,
- missing required asset reference,
- missing required map/monster/skill/cosmetic presentation mapping for the selected release scope,
- dependency cycle/build failure,
- bundle/catalog generated by unpinned editor/package,
- content update changing an immutable published bundle identity,
- Resources growth beyond explicit bootstrap exceptions,
- generated catalog/build metadata missing required compatibility fields.

# Tests
Required:
- offline/base-install boot and useful update error UI,
- local catalog load,
- remote catalog update,
- cached re-entry without redownload,
- corrupted bundle/hash recovery,
- interrupted download/retry,
- insufficient disk/network failure,
- destination gate succeeds only after required assets ready,
- repeated map transitions do not leak handles/memory,
- content/catalog rollback compatibility,
- missing required asset fails before production activation,
- gameplay results remain server-authoritative if local presentation data is tampered with.

# Invariants
```text
Addressables 2.11.2
asset catalog != gameplay content authority
base install can boot/auth/show errors
remote publish precedes server requirement
destination presentation assets required before attach/presentation
published bundle bytes immutable per identity
Resources only explicit tiny bootstrap exception
raw AssetBundle feature pipeline disabled
handles released deterministically
```
