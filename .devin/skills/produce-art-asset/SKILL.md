---
name: produce-art-asset
description: Create or source a release art/audio asset — 2x authoring, cutout cleanup, volume/lighting, provenance, quality gates, in-game visual review. Use for IMP-070..076, IMP-104, IMP-105 and any client/Assets/Art change.
---

# Produce an Art Asset

Canonical: `docs/07_content/presentation_asset_manifest.md` §3–§7 (sizes, §3.1a gate scope, §3.2 Cutout Gate, §3.3 Visual Review, §3.5 art direction, §3.6 Volume & Depth Gate), ADR-0055, ADR-0056, `docs/04_architecture/physics_geometry_contract.md`, cultural rules in `docs/07_content/cosmetic_catalog.md`.

## Workflow

1. **Target.** Resolve the catalog ID → `size_profile`/`asset_class` → exact 2x texture size (cell ref px × 2), PPU (100; UI 200), compression class.
2. **Generate/source.** AI: native alpha or flat `#FF00FF` key background; prompt contains the §3.5 directives (top-front key light, 3 value tiers + occlusion, rim light, coloured outline, 3/4 view, material highlights). Free-licensed: only CC0-1.0 / CC-BY-4.0 / OFL-1.1 (fonts) from the original source page.
3. **Finish at target size.** Downscale once (area/Lanczos) to the exact 2x size, then clean edges, remove specks, dilate alpha ≥ 4 px, sharpen. Never ship raw "remove background" output; never let Unity resize.
4. **Gates.** Run the Unity EditMode Cutout + Volume & Depth validators (`bash .devin/scripts/verify_delta.sh --full`). Zero violations; fix the art, never the thresholds.
5. **In game.** Place on the real map background; capture 1280x720, 1920x1080, 2400x1080, day and night, 100% and 200%. Attach to task evidence.
6. **Provenance.** Add the register fragment row (hashes of source and final, tool/version/terms, prompt, inputs, `review_state = PENDING`).
7. **Review.** A different agent (`reviewer`) checks screenshots against §3.3/§3.5 and the cultural rules and sets `APPROVED` or `REJECTED`.

## Acceptance
- Exact 2x size, correct import settings, both gates pass, screenshots approved, provenance row `APPROVED`, no placeholder, no historical person or non-Vietnamese mythology motif.
