# Cosmetics Style Pack — IMP-074

Style lock for all cosmetic presentation art (frames, title plates,
title glows, auras, trails, shrines, emotes, guild frames, nameplates,
inscription tablets, appearance icons + PSB cells).

- Generator: AI Horde (stablehorde.net API v2), model AlbedoBase XL (SDXL).
- Seed: deterministic `sha256("IMP-074/" + cosmetic_id)` truncated to int32.
- Reference: img2img from the class anchors in this pack per ADR-0076.
- Look: 2D pixel/chibi Vietnamese folk-fantasy; 3/4 view for objects,
  frontal standing pose for appearance cells; top-front key light;
  3-tier volume (core / mid / lit) + occlusion shadow + rim light;
  coloured outline, clean hard cutout, no semi-transparent fringes.
- Palette: `palette.json` (CIELAB anchors) — every produced texture must
  sit >=85% of opaque pixels within DeltaE00 <= 8.
- Alpha: rembg (u2netp) on CPU, then cutout harden + despill + 4px
  dilation; no raw remove-background output ships.
- Forbidden motifs: torii, jiangshi garb, kimono, hanbok, modern
  religious/political insignia, meaningless Han/Nom glyphs, trademarks.
- Layers (appearance PSB): leg_back, leg_front, torso, arm_back,
  weapon, arm_front, head, hair — bound to the shared class skeleton.
