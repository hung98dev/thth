# Style Pack: interface

Scope: `client/Assets/Art/UI/`, `client/Assets/Art/Items/`, `client/Assets/Art/VFX/`
(IMP-073 interface art: UI chrome, item/equipment/skill/status icons,
skill VFX flipbooks, telegraph decals, font art references).

## Art direction (presentation_asset_manifest.md §3.5, ADR-0055/0056)

Stylized 2D Vietnamese-folklore MMORPG interface art, painted volume:

- single warm top-front key light; three value tiers plus soft occlusion
  shadow under forms; dark coloured outline (never pure black); subtle rim
  light for separation.
- Material language: dark lacquered wood, aged bronze trim, aged gold,
  jade accents, warm parchment — village-hall craft, not high-fantasy neon.
- Element accents follow the Five Phases: kim = aged gold/silver metal,
  moc = jade green, thuy = river blue/cyan, hoa = vermilion/ember, tho = amber/ochre.
- Telegraphs: warm red-white danger fill + ring, PLUS a secondary
  luminance-coded mark (the decal stays readable to colour-blind players
  on any ground). Never encode gameplay outcomes (no numbers, no kill marks).
- Icons: single subject, 3/4 view, large readable silhouette, centred,
  cropped to the 64x64 ref cell (128x128 finished) with 4 px margin.

## Palette

`palette.json` — Lab palette used by the palette gate (>= 85% of
silhouette pixels within DeltaE00 <= 8 of the nearest colour).

## Anchors

`anchor_*.png` (>= 6): reference images each family was img2img-derived
from — item cluster, equipment pieces, UI chrome set, per-element VFX
bursts (kim/hoa/thuy + anchors for moc/tho via the nearest family),
telegraph shape study, avatar portrait study.

## Tooling

All anchors produced with the owner-approved tool (technology_versions.md
§ Content production tools): AI Horde / AlbedoBase XL 3.1, deterministic
seeds. Derived files record `style_pack_id: interface` in their
`generation_record`.
