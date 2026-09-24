# Guild Progression
status: LOCKED

## Scope
Defines Guild Level/EXP, contribution, weekly Five-Element Ritual, Blessing vote, capacity unlocks, and anti-FOMO.

## Core Loop
```text
play together -> Guild EXP + contribution -> fill five ritual vessels -> weekly Ritual -> 3 Blessings -> member vote -> 1 active Blessing
```

## Guild Level
New guild Lv1/0 EXP; max Lv30. Guild EXP is cumulative/non-spendable.

### EXP Threshold
For L 1..30:
```text
required_total_guild_exp(L) = 100 * (L - 1)^2 + 200 * (L - 1)
```
Examples:
```text
Lv1=0
Lv5=2,400
Lv10=9,900
Lv20=39,900
Lv30=89,900
```
One grant may cross levels; level never decreases normally.

## Capacity
Members: Lv1-4 30; 5-9 35; 10-14 40; 15-19 45; 20-24 50; 25-29 55; 30 60.
Vice Leaders: Lv1-14 1; 15-29 2; 30 3.
Unlocks: Lv1 COMMON vault; Lv10 RESERVE vault + rated Guild War.

## Guild EXP Sources
Baseline per eligible guild event:
```text
party dungeon completion 20
configured public/world boss 20
configured world event 15
explicit guild activity 30
weekly Ritual completion 200
```
No ordinary kill/login/character EXP/trade/craft grant. Guild War owns its capped override.

## Member Contribution
Non-spendable metric. Baseline: dungeon 20, boss 20, world event 15, guild activity 30, valid ritual contribution 5. Leaving preserves historical total; rejoin same guild resumes total but cycle contribution starts 0.

# Five-Element Ritual
Cycle = 7 days, Monday 00:00 UTC. Five vessels KIM/MOC/THUY/HOA/THO.
At cycle start snapshot active members M:
```text
M_effective = clamp(M,5,40)
required_points_per_element = 120 + 12*M_effective
```

## Ritual Points
Every eligible event grants explicit points to exactly one authoritative element:
```text
eligible party dungeon completion = 12 points
eligible configured boss = 12
configured world event = 10
explicit guild activity = 15
```
Element assignment uses one of four modes:
- `FIXED_ELEMENT`: the activity definition specifies a single authoritative element that always receives points from that activity type.
- `ACTIVITY_ELEMENT`: the element is derived from the thematic/elemental tag on the activity being performed (e.g., a boss tagged `MOC` grants to the MOC vessel).
- `CHARACTER_CLASS_ELEMENT`: the element is derived from the contributing character's class Ngũ Hành affinity.
- `SERVER_ROTATION`: the server selects the receiving element from a deterministic rotation derived from `guild_id + cycle_id`; the current element is published in the guild UI but no member may override it. The rotation cycles KIM→MỌC→THỦY→HỎA→THỔ and then repeats, advancing once per qualifying activity.

Client never chooses arbitrary points/element. One activity operation grants ritual points once, even if many guild members participate. Overflow does not spill. One Ritual completion/cycle.

## Anti-FOMO
No daily ritual streak requirement, level decay, or missed-cycle power loss. Joining mid-cycle can contribute. Streak rewards cosmetic only.

# Blessing Draft / Vote
Ritual completion deterministically creates 3 distinct candidates from guild_id + cycle + catalog revision. Vote window 24h. Eligible snapshot member still current at vote time; one account max one vote; all weight 1. Highest votes wins; ties lexical ID; zero votes highest candidate priority then lexical.

One active Blessing/guild, duration 7 days from finalization. Current members receive; leave/kick removes. Rated Guild War disables initial catalog.

## Initial Blessings
```text
guild.blessing.advancement
PvE character EXP +5%

guild.blessing.hunt
eligible ELITE/BOSS tagged material drop weight +10% relative
never applies to equipment, Soul, cosmetic, or explicitly rare-jackpot tables unless that drop table opts in

guild.blessing.craft
base enhancement success +300 bp, respecting crafting cap

guild.blessing.exploration
NPC travel common cost -25%

guild.blessing.endurance
PvE MAX_HP +3%

guild.blessing.activity
ritual point grants and member contribution from explicit GUILD_ACTIVITY events +10%, rounded down
```
ACTIVITY intentionally does not increase Guild EXP at Lv30.

## Quality Bands
Lv1 advancement/hunt/exploration; Lv10 +craft; Lv20 +endurance; Lv30 +activity. Higher levels expand variety, not raw tiers.

## Ritual Streak
4/8/12 consecutive cycles may grant crest/title/shrine/profile cosmetics only.

## Persistence / Idempotency
Persist EXP/level/revision, cycle/vessels/streak, draft/candidates/votes, active Blessing/expiry, member contribution. Stable operation IDs for every grant/vote/finalization/unlock.

## Invariants
```text
GUILD_MAX_LEVEL=30
Lv30 threshold=89,900
ritual=7 days
five vessels
one account <=1 vote/draft
one active Blessing
HUNT bonus is relative/tagged, not +percentage-points
```
