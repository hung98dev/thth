# CI cache policy (IMP-106, engineering_conventions.md §6)

Applies to `.github/workflows/verify.yml`. Enforced by
`server/internal/conformance/caching` tests in CI (Q3).

## Mechanism

- Only `actions/cache@55cc8345863c7cc4c66a329aec7e433d2d1c52a9` (v6.1.0, pinned
  in `server/internal/stackpin/pins.go`) may be used. The combined action is
  required — `actions/cache/restore` / `actions/cache/save` are unlisted
  actions and fail Q1 `stackpin.github_actions`.
- `setup-go` keeps `cache: false` — the explicit `actions/cache` step is the
  only Go cache mechanism.
- A cache step restores AND saves (post-job). Save-on-failure poisoning is
  prevented by writing payloads atomically (`.part`/`mv`) and by validating
  content on restore (`docker load` + `docker image inspect`, EDB sha-256
  marker). An unloadable entry degrades to the uncached path, never to a
  failed gate.

## Keys and restore-keys

- Key grammar: `<scope>-${{ runner.os }}-<pin>-<content-hash>`. Every key must
  contain `${{ runner.os }}` plus every input that determines the payload:
  | scope           | pin                                  | content hash |
  |-----------------|--------------------------------------|--------------|
  | `go-build`      | `env.GO_VERSION`                     | `hashFiles('server/go.sum')` |
  | `unity-image`   | `env.UNITY_<OS>_IMAGE_DIGEST`        | (digest is the pin) |
  | `unity-library` | `env.UNITY_<OS>_IMAGE_DIGEST`        | `hashFiles(manifest.json, packages-lock.json, ProjectSettings/**)` |
  | `edb`           | `env.EDB_ZIP_SHA256` + version       | (sha is the pin) |
- `restore-keys:` entries must keep `${{ runner.os }}` AND the pin segment —
  a fallback may only roll the content hash within the same OS + same pinned
  toolchain/image/digest. Bare prefixes (`go-build-`, `unity-image-`) that
  would substitute another pin or OS are forbidden (CI-001).
- `UNITY_<OS>_IMAGE_DIGEST` env values must equal the `@sha256:` suffix of the
  matching `UNITY_<OS>_IMAGE` pin — the tests assert it, so the two env keys
  cannot drift.

## Never cached (CI-002)

No `path`/`key`/`restore-keys` may cover licence or credential state:
`unity-lic`, `Unity_lic.ulf`, `~/.local/share/unity3d`, `ProgramData\Unity`,
`.ulf` files. Licence activation runs every attempt. A cache hit must never
skip a Q0-Q6 gate, the fork/freeze guards, the materialization retry loop,
the `commit unity-materialized` drift check, or the licence activation.

## Telemetry (CI-003)

- Each cached step emits one JSONL line via `cache_telemetry.sh` /
  `cache_telemetry.ps1` to `$RUNNER_TEMP/cache-telemetry.jsonl`
  (`{step, result: hit|miss, wall_seconds}`). `RUNNER_TEMP` is outside the
  workspace so telemetry never dirties the tree (Q6 clean_tree).
- `scripts/verify.ps1` measures its own `go run` wall time, appends a `verify`
  entry (hit = `THINHTHAN_CACHE_HIT_GO`, set from the Go cache step output),
  then runs `server/internal/conformance/caching/cmd/cachemerge` to fold all
  entries into `verify-report.json` as `cached_steps[]`. The merge is
  best-effort — it can never fail verification.
- Evidence manifests (`gates.MergeReports`) decode reports into the fixed
  `VerifyReport` struct, so `cached_steps` is dropped before the manifest —
  evidence identity is cache-independent (CI-004).

## Postgres service container

The Linux `services:` postgres container cannot be cached and stays a service
container — pulling `postgres:18.6` (~90 MB) is seconds; converting it to a
step-managed container would only save that pull, not worth the lifecycle
risk. The Windows EDB binaries ARE cached (large download, sha-asserted).
