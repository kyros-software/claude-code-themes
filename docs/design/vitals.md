# The pet's usage

What exactly the statusline's pet measures, and what takes it from *fresh* to
*k.o.*

This is **one of the two layers**. Life belongs to the moment: it goes up and down
with usage and it comes back when you compact. The other layer, the progress — the
XP that picks what it evolves into — is in [evolution.md](evolution.md), and it
never goes down.

## One number

Everything — state, eyes, feet, head, colour — comes out of **one number between 0
and 100**: how full the context window of **this session** is.

```
usage = context_window.used_percentage
```

A real example: context at 36% → `lively`. Neither the five hours nor the seven
days come into it: they belong to the **account**, not to the session, and they
have their own place in band 1.

## Why the context only

This is the third answer to the same question, and the two before it were not
thrown away on a whim. Each fixed something real and broke something else.

**First it was a 50/30/20 weighted average**, with a reasonable argument behind
it: the context is the only thing you can manage in the moment — you compact, you
close the session, you open another — so it weighs more, but the limits squeeze
too.

What that argument did not see is that **an average dilutes** exactly the case
that matters. With the window completely full and the quotas idle:

```
average:  0.5·100 + 0.3·20 + 0.2·10  =  58   →  "easy", turquoise
```

The context spent, no room to work in, and the pet saying it is comfortable. That
is not an unfortunate weighting: it is the number lying at precisely the moment it
needed not to.

**Then it was the tightest neck**, `max(ctx, 5h, 7d)`, which is what the project's
first version measured (`statusline.sh`, commit `05bf5c7`) under a line that still
sounds right: *«no finge emociones; refleja el cuello más apretado»* — it does not
fake emotions; it reflects the tightest neck. It fixed the dilution outright and
brought a problem that document already warned about — "if the 7-day limit is
sitting at 95%, the pet is `drowning` all week even if you open the session with
an empty window" — and which was judged a cheap price.

It was not, because the problem is worse than that warning said. **The quotas
belong to the account.** Every open session reads the same number, so the pet
stopped describing the session it lives in:

```
session A   window at  6%,  5h at 81%   →  tired
session B   window at 64%,  5h at 81%   →  tired
```

Two windows with nothing in common, two identical pets, and a `/clear` that
changed nothing because what governed was not the context. The reading was not
lying about the account; it was lying about **the session**, which is what the pet
talks about.

**And the context really is an experience.** A full window is a slower, thicker
Claude, something you feel while you work, in this terminal, in this conversation.
`sluggish` and `tired` describe that. A quota at 81% is not felt in any answer: it
is felt when it cuts you off, and that does not need a face, it needs a number.

## The quotas do not disappear

They are still in band 1, as numbers, **painted with this same ladder**:

```
████░░░░░░░░░░░░ 7% · 1M ctx │ xhigh │ 5h 82%  7d 21%
```

A `5h` at 95 comes out in `drowning`'s indigo, so the thing about to stop you is
the strongest colour on the line even with the pet in green. That is all they
need: they say how much of the day is left, and that reads as a figure.

API accounts do not receive `rate_limits`, so there is nothing to read there — and
that used to force a special case into the pet. Not any more.

## The curve comes from the first version

The thresholds are not arbitrary and have never been touched. They are the comfort
curve `statusline.sh` drew, **quadratic** — high and flat at the bottom, falling
away only near the top, because *44% is not half a life* — solved for usage:

```
life = 100 · (1 − (usage/100)²)      →      usage = 100 · √(1 − life/100)

life 95 → 22.36 → cap 22        life 40 → 77.46 → cap 78
life 80 → 44.72 → cap 45        life 20 → 89.44 → cap 89
life 60 → 63.25 → cap 63
```

The curve has survived all three inputs intact; the only thing that drifted was
what was fed into it. A test pins it
(`TestTheThresholdsAreTheFirstVersionsComfortCurve`).

## Band 1's bar measures the same thing

The bar and the pet are **one measurement**. It is the only arrangement that has
not failed, and the other two were tried:

- The bar measured the context and only **borrowed the colour** of the neck: with
  the context at 48% and the 5h quota at 67 you got a half-mast bar next to the
  word `sluggish`, which is the reading for 67. Two numbers on the same line, and
  the one in charge was the one you could not see.
- The bar was promoted to the neck to close that gap, and then the band printed
  `82% 5h` three columns before printing `5h 82%` again, while the context — the
  only one of the three that belongs to this session — was left with a bare `7%`
  and no bar.

Now the length, the number and the colour are the context, and the pet is that
same context. They cannot disagree, because there are not two things.

The colour, on top of that, is **the pet's body**: its branch's ramp at the step
the state picks. It used to be the state ladder's colour, which agreed with the
pet about *how* the session is going but not about *who* is living it — one ladder
for every pet, when since the atlas the hue belongs to the branch. A blue
`bughunter` next to a green bar that meant the same thing.

## Where each state falls

| Usage | State | Eyes | Head | Feet |
| --- | --- | --- | --- | --- |
| ≤22% | fresh ✦ | `>` `<` | up | walking |
| ≤45% | lively | `>` `<` | up | walking |
| ≤63% | easy | `o` `o` | up | walking |
| ≤78% | sluggish | `▬` `▬` | up | still |
| ≤89% | tired | `_` `_` | sunk | still |
| <100% | drowning | `x` `x` | sunk | still |
| **100%** | k.o. | `x` `x` | sunk | on its back, feet in the air |

At **hunger ≥7** the eyes do not change shape: their colour goes out. Hunger
belongs to the other layer and does not touch the state.

They are **four independent signals** that drop away in order: first the eyes,
then the step of the feet, then the head sinks and at the end the silhouette lies
down. At a glance you can tell *tired* from *drowning* without reading the label.

**The k.o. no longer needs a back door.** `StateFor` carried a second argument
whose only job was to force the k.o. when the context reached 100, because an
average of three numbers does not reach 100 unless all three do: with ctx, 5h and
7d at 100, 90 and 90 the average came out 95 — *drowning* — and that sprite was
never seen. A number that already is the context reaches 100 on its own.

## When the data is missing

An old CLI does not send `context_window`. Then usage is 0, the pet comes out
fresh, and **band 1 draws no bar**: a bar at 0% would be a measurement nobody has
taken. No state is invented, and no quota is put in its place.

## What it does NOT measure

Not the **cost in dollars**, not the **session time**, not the **lines touched**,
not git's state, not the cache, and not — since this version — the **account's
quotas**. All of that shows in the bands, but none of it reaches the pet.

Nor does it measure **progress**. The pet being *k.o.* does not send it back to a
larva: the silhouette is chosen by the XP, and usage does not touch the XP. A
blown-out `surgeon` is still a surgeon, with the face of one who has seen things.

The counters that open the ember branch — `impulsive` from a peak of 85% on,
`ctx_maxed` from 95% on — read that same context peak. For a while they read the
neck, on the argument that a tight quota is also working at the limit; the result
was that opening four sessions in parallel paid for the branch without a single
window ever being filled.

## Honesty

The pet **does not fake emotions**. It does not cheer up because the code compiles
or sadden because a test fails: it reflects a real, checkable number, and now on
top of that a number the session showing it is responsible for. If it is tired, it
is because your window is at 85%.

---

See also the [README](../../README.md) for the bands, the palette and the rest of
the statusline, and [evolution.md](evolution.md) for the other layer: XP, hunger,
food and the 41 evolutions.
