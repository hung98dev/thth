# actors_creatures style pack

## Directive (prompt framework)

```
Vietnamese folklore creature sprite for a 2D MMORPG, painterly
brushwork, cel shading with soft rim light, dark ink outline, strong
readable silhouette, top-lit key light, <pose>, <entity descriptor>,
flat solid pure magenta background (RGB 255,0,255) covering the whole
canvas behind the creature, full body visible, feet at bottom edge,
centered
###<negative>
```

Negative: photorealistic, photo, 3d render, multiple creatures, extra
limbs, text, letters, signature, watermark, border, frame, torii gate,
kimono, hanbok, jiangshi hat, talisman script, chinese characters, nom
script, modern flag, scene background, landscape, sky, ground shadow
gradient, vignette.

## Rules

- Textures authored at 2x of the reference cell (ADR-0055); pivot
  Bottom Center (0.5, 0); PPU 100.
- Alpha: background flood cutout + despill + despeckle + interior-hole
  fill + edge dilation >= 4 px (cutout gate §3.2, zero violations).
- Volume: painterly dark-or-light rim (|dL| >= 12 vs inner ring),
  top-lit gradient, L* range >= 40, no flat Lab-bin region > 20%
  (§3.6).
- Motifs: no torii, no Qing/jiangshi clothing, no kimono/hanbok, no
  meaningful Han/Nom script, no modern religious/political symbols
  (§5 ART-010). Vietnamese folkloric identity only.
- Skeletal rigs (MONSTER_MEDIUM, MONSTER_ELITE, BOSS_LARGE,
  WORLD_BOSS) share the 8-part layer convention:
  leg_back, leg_front, torso, arm_back, weapon, arm_front, head, hair.
  Non-humanoid creatures map anatomy to nearest part (tail->leg_back,
  wing->arm_back, etc.) and leave unused layers empty but named.
- Frame-by-frame (MONSTER_SMALL, SPIRIT_BEAST): 4 frames/clip at
  12 fps; idle/move/cast bbox drift <= 8 px, action clips <= 32 px;
  hue consistent with idle frame 0.
- Shared variants are prefab variants of the family base with a
  `SpriteRenderer.color` tint — never a re-render, never placeholder.
- palette.json lists the pack's Lab color centers; >= 85% of every
  sprite's silhouette must sit within deltaE00 <= 8 of a center
  (style-pack gate).

## Anchors / turnarounds

- `anchors/` — 8 approved finished sprites (2 bosses, 4 monsters,
  2 beasts) establishing the silhouette/volume bar.
- `turnarounds/<entity>_{front,front34,back34,side}.png` — img2img
  turnaround sheets for the 8 bosses; reviewer-approved before
  mass production.
