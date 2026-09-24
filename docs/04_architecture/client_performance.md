# Client Performance & Smoothness
status: LOCKED

## Scope
Canonical player-facing performance and smoothness targets for the Unity client: device tiers, frame pacing, memory/GC, load times, input responsiveness, network smoothness, mobile thermal/battery, and how each is measured. Server-side targets stay in `../08_scale_ops/capacity.md`. Measurement uses Unity built-ins only (`FrameTimingManager`, `ProfilerRecorder`); no new package.

## Platforms and Device Tiers
Launch platforms: Windows desktop and Android. iOS is not a launch target (Windows-only CI, ADR-0050).

| Tier | Reference hardware | Target |
|---|---|---|
| `DESKTOP_MIN` | 4-core x64 CPU, 8 GB RAM, DX11 GPU at Intel UHD 620 level, 1920x1080 | 60 FPS, quality `MEDIUM` |
| `ANDROID_MIN` | Android 10, 4 GB RAM, Snapdragon 665 / Helio G85 class (Adreno 610 / Mali-G52) | 30 FPS locked, quality `LOW` |
| `ANDROID_REC` | Android 12, 6 GB RAM, Snapdragon 778G class | 60 FPS, quality `MEDIUM` |

Quality presets `LOW / MEDIUM / HIGH` are chosen automatically at first launch from a 5-second GPU/CPU benchmark and can be changed in settings. They control only presentation: render scale (`LOW` 0.75, others 1.0), active point Light2D budget (`LOW` 4, `MEDIUM` 8, `HIGH` 16), particle budget, parallax layers shown (`LOW` hides `L3`), and post-processing (bloom only on `HIGH`). Presets never change world scale, colliders, hitboxes or telegraph visibility.

## Frame Pacing
Measured in the canonical hotspot scene (18 players + 42 `AI_CLASS_NAMED_MECHANIC` monsters + skill VFX, `../09_testing/load.md` scenario 11) for 5 minutes:

```text
tier           target   p95 frame     p99 frame     hitches (>50 ms)
DESKTOP_MIN    60 FPS   <= 16.7 ms    <= 25 ms      0 per 5 min in combat
ANDROID_REC    60 FPS   <= 16.7 ms    <= 25 ms      <= 1 per 5 min
ANDROID_MIN    30 FPS   <= 33.3 ms    <= 45 ms      <= 1 per 5 min
```
Frame budget on `ANDROID_MIN` (33.3 ms): scripts <= 10 ms, rendering <= 12 ms, remainder for OS/GPU. Batches <= 150 on mobile (SpriteAtlas per region/actor group, no per-frame material instancing). VSync on desktop; `Application.targetFrameRate` = tier target on Android.

## Memory and GC
```text
managed GC allocation per frame in steady gameplay (movement, combat, UI HUD) = 0 bytes
allocations allowed only at load, scene transfer, and opening/closing full-screen UI
total resident memory: ANDROID_MIN <= 1.3 GB, DESKTOP_MIN <= 2.5 GB
```
Pooling rules: `../10_implementation/engineering_conventions.md` §2.3. Addressables group budgets: `../07_content/presentation_asset_manifest.md` §1.

## Load and Transfer Times
```text
cold start to login screen            <= 10 s (ANDROID_MIN), <= 6 s (DESKTOP_MIN)
login to in-world (first map)         <= 8 s
map transfer, same region             <= 3 s
map transfer, new region (bundles)    <= 6 s
reconnect resume to controllable      <= 5 s after connection is restored
```
Every wait longer than 0.5 s shows a progress screen or indicator; no frozen frame longer than 100 ms during loading (async Addressables + incremental instantiation).

## Input Responsiveness
```text
local input -> first visual response (animation/prediction start)  <= 1 rendered frame after input sampling
local movement                                                      predicted immediately (../05_network/synchronization.md)
server-confirmed combat result shown                                <= RTT + 50 ms
```
Input is sampled once per rendered frame before simulation presentation; touch controls have no added debounce.

## Network Smoothness
```text
remote interpolation delay      = 2 snapshot intervals (200 ms at 10 Hz), adaptive 150..300 ms from measured jitter
extrapolation limit             = 250 ms, then freeze/lerp (../05_network/synchronization.md)
local correction                = smooth over 100 ms when error <= 0.5 m; snap above (../04_architecture/physics_geometry_contract.md)
full-quality conditions         = RTT <= 150 ms, jitter <= 30 ms, loss <= 2%: no visible rubber-band, corrections > 0.5 m <= 1 per minute
degraded-but-playable           = RTT <= 300 ms, jitter <= 60 ms, loss <= 5%: client shows the network indicator; no desync
```

## Mobile Sustained Performance
Battery drain cannot be measured in the cloud pipeline, so sustained performance is gated by proxies:
```text
30-minute game-loop run on ANDROID_REC-class device   p50 >= 45 FPS for the whole run (thermal throttling allowed below 60)
ANDROID_MIN-class device, 30 FPS cap                  average CPU utilisation <= 50%, no frame p95 regression > 10% between minute 1 and minute 30
```
A battery-saver toggle caps FPS at 30 on any tier.

## Measurement and Gates
- Desktop: PlayMode performance tests run the hotspot scene with `FrameTimingManager`/`ProfilerRecorder` on the cloud Windows runner, which must have a GPU at `DESKTOP_MIN` level or better (Owner Setup, `../10_implementation/audit_gates.md`); a runner without such a GPU makes the gate an `OPS` blocker, never a skipped pass.
- Every PR, device-independent budgets (runs on the Windows runner, always required once the owning task is DONE):
  ```text
  managed GC allocation per frame in the hotspot scene       = 0 bytes
  batches <= 150, SetPass calls <= 60 (LOW preset)          texture memory within presentation_asset_manifest.md §1 budgets
  active point Light2D and particle counts <= preset budget  desktop frame targets above
  ```
- Android device runs: an IL2CPP Android build of the hotspot scene runs as a Unity game-loop test on Firebase Test Lab physical devices (one `ANDROID_MIN`-class and one `ANDROID_REC`-class model, recorded in Owner Setup); CI downloads the frame timings and gates them. No device is attached to the runner. Cadence stays inside the free quota: at most one scheduled run per day on `main` when client code/assets changed since the last device run, plus the mandatory launch-candidate run. A run blocked by exhausted quota is reported `DEFERRED(quota)` and retried the next day; it never fails or blocks ordinary PRs. The launch-candidate gate requires a passing device run on the release commit and waits (it never skips) until quota allows.
- Network smoothness tests use the client network emulator (latency, jitter, loss) against a local server in PlayMode.
- The 30-minute sustained runs execute on Firebase Test Lab during the launch-candidate gate.
- Any metric above its target fails the gate; targets change only by spec change (gate ratchet, ADR-0050).

## Requirement IDs
Every ID below must be named in at least one task packet's acceptance and covered by a named test; Q0 fails on an uncovered ID (`../10_implementation/audit_gates.md` requirement coverage).

| ID | Requirement (section) | Gate |
|---|---|---|
| `PERF-001` | Auto benchmark picks LOW/MEDIUM/HIGH; presets change presentation only (Device Tiers) | every PR |
| `PERF-002` | DESKTOP_MIN frame pacing p95/p99/hitches (Frame Pacing) | every PR |
| `PERF-003` | ANDROID_MIN and ANDROID_REC frame pacing (Frame Pacing) | device run |
| `PERF-004` | 0 bytes managed GC per frame in steady gameplay (Memory and GC) | every PR |
| `PERF-005` | resident memory caps per tier (Memory and GC) | every PR (desktop) + device run |
| `PERF-006` | batches <= 150, SetPass <= 60, light/particle budgets per preset (Measurement) | every PR |
| `PERF-007` | cold start, login, map transfer and reconnect times (Load and Transfer Times) | every PR (desktop) + device run |
| `PERF-008` | progress shown for waits > 0.5 s; no frozen frame > 100 ms while loading | every PR |
| `PERF-009` | input -> first visual response <= 1 frame; confirmed result <= RTT + 50 ms (Input Responsiveness) | every PR |
| `PERF-010` | interpolation/extrapolation/correction values (Network Smoothness) | every PR |
| `PERF-011` | full-quality and degraded network conditions under emulated latency/jitter/loss | every PR |
| `PERF-012` | 30-minute sustained proxies on ANDROID_REC / ANDROID_MIN (Mobile Sustained Performance) | launch candidate |
| `PERF-013` | battery-saver toggle caps FPS at 30 | every PR |

## Invariants
```text
presentation quality presets never change gameplay geometry, hitboxes or telegraph readability
zero managed allocation per frame in steady gameplay
every wait > 0.5 s shows progress
client frame rate never changes server simulation (20 Hz)
```
