# actors_players — Player class style pack

## Prompt framework (lighting & volume)

All produced sprites share one framework:

- Subject: chibi Vietnamese folk-fantasy villager, big-headed proportions.
- View: three-quarter front (idle/showcase), 4-angle turnarounds for reference.
- Key light: single top-front source; cast shadow inside the silhouette
  down-left.
- Volume: painted shading, three value tiers (lit / mid / shadow) plus
  occlusion-dark contact edges; subtle rim light on the lit side.
- Outline: coloured outline slightly darker than the local fill (no pure
  black outline).
- Materials: woven cloth, bamboo, wood, bronze/iron, paper, cord.
- Background: solid pure magenta chroma key, removed by post-process.

Base prompt:

    2D hand-painted chibi character concept art, Vietnamese folk fantasy
    MMORPG, big-headed small-bodied proportions, painted volume shading
    with three value tiers and occlusion shadow edges, colored outline
    slightly darker than fill, subtle rim light, one key light from top
    front, cast shadow down-left inside silhouette, full body single
    character three-quarter front view, arms relaxed at sides, empty open
    hands, <class identity>, muted earth tones, plain solid pure magenta
    background, no scenery no props

Negative prompt:

    text, watermark, signature, border, frame, extra limbs, extra
    characters, scenery, ground plane, floor, shadow on floor, gradient
    background, western armor, plate mail, crystal, robe, onmyoji, katana,
    cultivation robes, lowres, blurry, cropped

## Palette

`palette.json` is the Lab cluster set extracted from the shipped
silhouettes (24 centres); the gate keeps >=85% of silhouette pixels
within deltaE00 <= 8 of a palette colour.
