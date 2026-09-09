# The 97-form tree: gates and thresholds

> **Design work, not implemented.** The code still has its 41 forms. This is the
> Claude Design canvas's tree checked against the runtime, with its defect found
> and the fix verified.
>
> **This page no longer has to be believed: it runs.** The gates live in
> `internal/pet/testdata/PUERTAS-97.json` and `internal/pet/reach97_test.go`
> checks them. Change a gate and break a mark, and the suite goes red with the
> mark's name on it.
>
> Original handover: <https://claude.ai/code/artifact/e15cd05f-cb01-4e68-a336-85c61100faee>

> **On the names.** The forms in this document are named the way the canvas names
> them, in Spanish, because that is what `PUERTAS-97.json` and `ATLAS-97.json`
> hold. The 41 forms the code implements have English ids; the 56 new ones do not
> have one yet, and choosing them is part of implementing this. See the last
> section, and [language.md](language.md).

## The rule, in full

Without this none of the sums below can be redone, and the previous version of
this document did not carry it: it pointed at the canvas, which is now spent.

**Level 2, the temperament.** The highest of three counters wins, ties broken by
list order. `evolution.go:97` (`BranchBy`) and `evolution.go:437` (`topBranch`).

| temperament | counter | trades hanging off it |
| --- | --- | --- |
| `pattern` | `methodical` | refactor, tidy |
| `probe` | `inquisitive` | bughunter, architect |
| `ember` | `impulsive` | sprinter, marathon, feral |

**Level 3, the trade.** The same rule between that temperament's siblings, each
with its own counter. That counter is left **out** of the trade's six gates, which
is the exclusion the canvas describes as "the nine that did not win the trade".

| trade | `refactor` | `tidy` | `bughunter` | `architect` | `sprinter` | `marathon` | `feral` |
| --- | --- | --- | --- | --- | --- | --- | --- |
| counter | `diffs` | `ctx_low` | `tests` | `plans` | `short_sessions` | `long_sessions` | `ctx_maxed` |

**Level 5, the mark.** This is where the canvas and this document disagree. The
canvas wants a race of raw counts; what is proposed here is a race of ratios with
a threshold per mark.

The two consequences that get forgotten unless they are written down: **to be in a
trade you have already won a race**, so inside it there are guaranteed
inequalities between counters; and **the tiebreak rewards the first on the list**,
so the order in which the canvas draws the six marks is information, not layout.

## The ten counters are not ten signals

They are ten readings of six things. `internal/pet/feeding.go:58` ties two pairs
and `internal/hook/app.go:411-460` nests the rest:

```
meal "tests"     -> inquisitive, tests   ⟹  inquisitive = tests + plans
meal "task"      -> inquisitive, plans
meal "commit"    -> methodical, diffs    ⟹  methodical  = diffs + compacts
meal "compact"   -> methodical
peak < 40        -> sessions_under_40    ⟹  ctx_low     ≥ sessions_under_40
peak < 60        -> ctx_low
peak ≥ 85        -> impulsive            ⟹  impulsive ≥ ctx_maxed ≥ ctx100_sessions
peak ≥ 95        -> ctx_maxed
peak = 100       -> ctx100_sessions
session < 15 min -> short_sessions, sessions_15min   ⟹  they are the SAME number
session ≥ 90 min -> long_sessions        ⟹  long_sessions ≥ sessions_4h
session ≥ 4 h    -> long_sessions, sessions_4h
```

The first three come from the meals; the rest from the session closing. A race
between a sum and one of its addends has a winner before it starts.

And there is something worse than the individual collisions: **the ten counters
the canvas uses to pick a mark are exactly the ten that already decided the
temperament and the trade** — three from level 2 plus seven from level 3. Level 5
brings no new signal; it hands out again the ones it already spent. That is the
root, and no reassignment of gates removes it entirely.

## The defect, by two yardsticks

| yardstick | reachable | dead |
| --- | ---: | ---: |
| wins even on a tie (order-based tiebreak gifts it the mark) | 21 | 21 |
| wins **without a tie** | 15 | 27 |

The 21 dead under either yardstick: `andamio`, `avalancha`, `cepo`, `cimiento`,
`cristal`, `erizo`, `flecha`, `francotirador`, `fuente`, `grieta`, `incendio`,
`injerto`, `jardinero`, `kraken`, `lienzo`, `lima`, `linterna`, `muelle`,
`oráculo`, `relámpago`, `sabueso`.

The 6 that live only off an exact tie — `cirujano`, `tejedor`, `buey`, `caravana`,
`muro`, `reloj` — need two counters to match to the integer (`compacts` at zero
and `plans` levelling with `methodical`, for instance). In real use that does not
happen: the defect is 27, not 21.

This matters because the previous version compared "21 before" with "42 after" and
they were not the same metric: the 42 was demanded with no ties. With the same
yardstick on both sides, the improvement is **from 15 to 42**.

## The fix: two pieces, and one is already built

**One — fifteen gates change.** No trade keeps a counter next to another that
contains it. It does not depend on any rate: it is algebra.

**Two — every mark asks for a threshold, and the race is of ratios**
(`counter ÷ what it asks for`). The threshold is not a door to cross: it is the
unit converter that lets `bypass_turns`, which goes per turn, be compared with
`ctx100_sessions`, which climbs once per blown session.

Piece two **does not need designing: the runtime already does it**. `ripestMark`
(`evolution.go:343`) picks by ratio, `Unlocks` (`evolution.go:111`) holds each
mark's counter/threshold pair and `Mark.Share` paints it in band 4. It was put
there for this very defect in the 41-form tree — its comment tells how eight of
the forty-one forms had stopped existing — and what is missing is filling
`Unlocks` with 42 entries, not inventing mechanics.

The canvas rejects thresholds on the argument that a pet meeting none of them
would be left with no trade. That is true of a hard door and false of a race of
ratios: the maximum of a ratio always exists, just as the maximum of a count does.

## The rates, measured

366 transcripts from `~/.claude/projects`, 31 days, with the hook's own detection
patterns and its cooldowns applied. *(Measurement inherited from the previous
session; it has not been re-run. What was checked is that this machine's
`pet.json` still has `plans`, `impulsive`, `ctx_maxed` and `ctx100_sessions` at
zero, which is what forces four thresholds to be estimated.)*

| Signal | Per week | Source | Feeds |
| --- | ---: | --- | --- |
| user turns | 682 | measured · transcripts | — |
| turns on bypass | 651 | measured · pet.json, 95% of the total | `bypass_turns` |
| sessions | 34 | measured · 134 with timestamps | — |
| green suites (after cooldown) | 53.3 | measured | `tests` `inquisitive` |
| commits | 46.3 | measured | `diffs` `methodical` |
| sessions >90 min | 22.8 | measured · 68% | `long_sessions` |
| sessions ≥4 h | ~8 | **derived** · 6/17 of the `pet.json` | `sessions_4h` |
| compacts | 5.4 | measured | `methodical` |
| sessions <15 min | 5.0 | measured · 15% | `short_sessions` |
| plan tasks closed | 0 | measured · zero TodoWrite in 366 files | `plans` |
| sessions peaking ≥85% | 7 | **estimated** · the pet is at 0 | `impulsive` |
| sessions peaking ≥95% | 3 | **estimated** · the pet is at 0 | `ctx_maxed` |
| sessions at 100% | 1 | **estimated** · the pet is at 0 | `ctx100_sessions` |

At that rate it is **1,398 xp a week**, so level 5 (2000 xp) arrives in **10
days**. Each threshold is the expected value of its counter on arriving there.

**Cross-check.** `19 diffs / 46.3` = 2.9 days; `23 tests / 53.3` = 3.0;
`12 long_sessions / 22.8` = 3.7, against a `streak: 3` in the `pet.json` of the
time. And the bypass figure comes out of two independent routes that agree to
within 5%.

## The 42 marks

`~~struck through~~` is the canvas's gate; in bold, the new one. The titles were
checked one by one against `ATLAS-97.json`'s parents: all 42 line up.

| trade | mark | gate | asks | title |
| --- | --- | --- | ---: | --- |
| refactor | `cirujano` | `ctx_low` | 80 | `bisturí` |
| refactor | `tejedor` | `plans` | 29 | `telar` |
| refactor | `molde` | ~~`methodical`~~ → **`impulsive`** | 10 | `imprenta` |
| refactor | `lima` | `tests` | 76 | `espejo` |
| refactor | `injerto` | ~~`inquisitive`~~ → **`short_sessions`** | 7 | `raíz` |
| refactor | `tijera` | `long_sessions` | 33 | `guillotina` |
| tidy | `monje` | ~~`methodical`~~ → **`impulsive`** | 10 | `abad` |
| tidy | `jardinero` | `plans` | 29 | `bosque` |
| tidy | `fuente` | `diffs` | 66 | `acueducto` |
| tidy | `cristal` | `tests` | 76 | `prisma` |
| tidy | `nieve` | `short_sessions` | 7 | `ventisca` |
| tidy | `lienzo` | ~~`inquisitive`~~ → **`long_sessions`** | 33 | `mural` |
| bughunter | `sabueso` | `plans` | 29 | `lobo` |
| bughunter | `exterminador` | ~~`inquisitive`~~ → **`impulsive`** | 10 | `avispa` |
| bughunter | `cepo` | ~~`diffs`~~ → **`short_sessions`** | 7 | `red` |
| bughunter | `linterna` | `methodical` | 74 | `faro` |
| bughunter | `anzuelo` | `ctx_low` | 80 | `arpón` |
| bughunter | `lupa` | `long_sessions` | 33 | `microscopio` |
| architect | `cartógrafo` | ~~`inquisitive`~~ → **`impulsive`** | 10 | `atlas` |
| architect | `oráculo` | `tests` | 76 | `esfinge` |
| architect | `andamio` | `methodical` | 74 | `catedral` |
| architect | `brújula` | `ctx_low` | 80 | `sextante` |
| architect | `cimiento` | ~~`diffs`~~ → **`short_sessions`** | 7 | `muralla` |
| architect | `maqueta` | `long_sessions` | 33 | `ciudad` |
| sprinter | `relámpago` | ~~`diffs`~~ → **`long_sessions`** | 33 | `tormenta` |
| sprinter | `francotirador` | `tests` | 76 | `halcón` |
| sprinter | `flecha` | `methodical` | 74 | `saeta` |
| sprinter | `muelle` | `plans` | 29 | `resorte` |
| sprinter | `chispazo` | ~~`impulsive`~~ → **`ctx_maxed`** | 4 | `descarga` |
| sprinter | `patín` | `ctx_low` | 80 | `cohete` |
| marathon | `buey` | ~~`diffs`~~ → **`short_sessions`** | 7 | `mamut` |
| marathon | `topo` | `methodical` | 74 | `gusano` |
| marathon | `ancla` | `ctx_low` | 80 | `puerto` |
| marathon | `caravana` | `plans` | 29 | `legión` |
| marathon | `muro` | `tests` | 76 | `bastión` |
| marathon | `reloj` | ~~`inquisitive`~~ → **`ctx_maxed`** | 4 | `calendario` |
| feral | `gremlin` | ~~`impulsive`~~ → **`bypass_turns`** | 931 | `diablo` |
| feral | `kraken` | `long_sessions` | 33 | `leviatán` |
| feral | `avalancha` | ~~`diffs`~~ → **`ctx100_sessions`** | 2 | `glaciar` |
| feral | `erizo` | `short_sessions` | 7 | `espina` |
| feral | `incendio` | ~~`methodical`~~ → **`sessions_4h`** | 12 | `volcán` |
| feral | `grieta` | `tests` | 76 | `abismo` |

### Why `feral` cannot carry `impulsive`

In its own branch `impulsive` is the big counter by construction: it beats
`methodical` and `inquisitive` because it won the temperament, and it contains
`ctx_maxed`, which in turn beats `short_sessions` and `long_sessions` because it
won the trade. Almost everything a `feral` mark could ask for sits underneath it.

With `gremlin` asking for `impulsive` at 10 — the trade's lowest threshold — that
kills `kraken` (`long_sessions/33`) and `grieta` (`tests/76`), and raising the
threshold fixes nothing: raise it enough for those to win and `gremlin` never wins
at all. There is no value that balances. It is the canvas's own defect,
reintroduced inside the fix.

The runtime had already solved it in the equivalent branch: `Unlocks` opens
`feral`'s two marks with `bypass_turns` and `ctx100_sessions`, never with
`impulsive`. The same is done here, and `gremlin` gets back the gate the code
already gives it.

`incendio` is then left without `bypass_turns` and takes `sessions_4h`, which the
hook already feeds (`app.go:457`) and which burns nicely with the volcano's story.
Its threshold is the only **derived** one rather than measured: 33 `long_sessions`
through the 6/17 proportion this machine's `pet.json` shows.

## What is not solid

- **The root is still there.** Level 5 feeds on the same ten counters already
  spent above. This fix rearranges them so that no collision is fatal, and a test
  verifies it; but every cell depends on the relationship between two thresholds,
  and the thresholds come out of measured rates, they are not free. **The
  underlying alternative is the one the runtime already took**: 14 counters of the
  marks' own (`diff_streak`, `repro_before_fix`, `sessions_under_40`,
  `widest_commit`, `longest_plan`…), none of which decides any branch. With those,
  the problem does not exist by construction. Changing all 42 gates to their own
  counters is redoing the canvas's level 5, and that is an open design decision,
  not a fix.
- **Four thresholds are estimated** — `impulsive`, `ctx_maxed`, `ctx100_sessions`,
  `plans` — and one derived — `sessions_4h`. This profile has them at zero.
- **The distribution is a model, not a measurement.** That all 42 are reachable is
  verified; how often each comes up depends on the user profile assumed, and there
  is no data there. A uniform sampling gives very lopsided distributions
  (`chispazo` 99% of `sprinter`); that does not describe real use, but it warns
  that the balance is not proven.
- **Two title factors are invention, not data**: `methodical`'s and `impulsive`'s,
  which have no relative among the fourteen the canvas calibrated. They carry the
  median of the other twelve. If the canvas turns up with its 42 factors, those two
  are the first to replace.
- **`gremlin` is no longer the feral branch's default mark**, because
  `bypass_turns` at 931 is not crossed by accident. Who occupies that slot is
  design, not arithmetic.

## What is verified: 95 of the 97

| forms | how many | state |
| --- | ---: | --- |
| the root `chispa` | 1 | trivial |
| temperaments | 3 | verified |
| trades | 7 | verified |
| marks | 42 | **verified, 42/42 with no ties** |
| titles | 42 | **verified, with the number set** |
| secrets (`fénix`, `quimera`) | 2 | outside this tree — their own rule, already in the runtime |

## The 42 title thresholds

A title competes with nobody: it sits behind its mark and asks for *more of the
same counter* (`TitleUnlock`, `evolution.go:182`). The question is not who wins, it
is whether that counter can go on climbing **without losing the mark along the
way** — and that is not free, because every counter feeds something else. Reaching
`volcán` is 40 `sessions_4h`, which is 40 `long_sessions`, which is the gate
`kraken` is waiting at. Verified mark by mark: it can.

The number is what was nowhere to be found. **Ten of the twelve factors are
inherited from the canvas**, which already calibrated a title for that same counter
or for a verified twin; the other two carry the median of the twelve the canvas did
set.

| counter | mark | title | factor | where the factor comes from |
| --- | ---: | ---: | ---: | --- |
| `short_sessions` | 7 | 21 | ×3 | `sessions_15min` is the **same number** (`app.go:452-453`); `storm` asks ×3 |
| `sessions_4h` | 12 | 40 | ×3.33 | same counter; `mammoth` asks ×3.33 |
| `ctx100_sessions` | 2 | 7 | ×3.33 | same counter; `leviathan` asks ×3.33 |
| `bypass_turns` | 931 | 3,100 | ×3.33 | same counter; the canvas asked 100 over `gremlin`'s 30 |
| `ctx_low` | 80 | 240 | ×3 | `sessions_under_40` is the same peak at another cut; `abbot` asks ×3 |
| `ctx_maxed` | 4 | 13 | ×3.33 | same family of peaks; `leviathan` asks ×3.33 |
| `long_sessions` | 33 | 110 | ×3.33 | same family of durations; `mammoth` asks ×3.33 |
| `tests` | 76 | 255 | ×3.33 | same habit as `test_streak`; `wasp` asks ×3.33 |
| `plans` | 29 | 115 | ×4 | same habit as `plans_before_code`; `sphinx` asks ×4 |
| `diffs` | 66 | 165 | ×2.5 | same habit as `diff_streak`; `scalpel` asks ×2.5 |
| `methodical` | 74 | 240 | ×3.23 | **no relative**: median of the canvas's twelve factors |
| `impulsive` | 10 | 30 | ×3.23 | **no relative**: median of the canvas's twelve factors |

`TitleAsks` got its fourteen numbers from the canvas, one per title, replacing a
uniform multiplier that had been invented in that file — so writing a new uniform
multiplier here would have put back exactly what that change removed. Two invented
numbers out of forty-two, both marked, is the floor reachable without going back to
the canvas.

A test checks that no factor falls outside the ×2.0–×4.5 range the canvas spent on
its fourteen, so that a future edit cannot slip a ×10 in without anything firing.

## How it was verified

- **Reachability** — `go test ./internal/pet/ -run 'NinetySeven|Temperament|Title'`.
  A constructive witness per mark: the minimum state that leads to the trade is
  built and the target counter raised, always climbing, never falling. 42/42 with
  no ties.
- **Cross-check** — three independent methods agree on the diagnosis (21 dead under
  the canvas's rule): exhaustive enumeration, a 4 M-state sampling and the
  constructive witness. The list comes out identical name by name.
- **What caught a false negative** — the first search was random and declared two
  reachable marks dead (`incendio`, `linterna`). That is why the repo's test is
  constructive: a blind search errs towards the side that looks prudent.
- **The titles** — the table's 42 mark/title pairs were checked against
  `ATLAS-97.json`'s parents. All 42 line up.

## The atlas is complete

`internal/pet/testdata/ATLAS-97.json` — the 97 forms with name, parent, note, base
colour, seven-step ramp and the seven five-row silhouettes. The same schema as the
41-form `ATLAS.json` the tests use, which is left untouched until this is
implemented.

Extracted from the canvas in five pieces (the whole files exceed
`DesignSync.get_file`'s 256 KiB cap) and verified:

| Check | Result |
| --- | --- |
| forms | 97 |
| structure | all of them: 7 states × 5 rows × 9 columns, a 7-step ramp |
| tree | 1 + 3 temperaments + 7 trades + 42 marks + 42 titles + 2 secrets |
| against `ATLAS.json` | 41 in common, 40 identical character for character |
| duplicate silhouettes | 0 — the canvas promises "no two alike" |
| distinct ramps | 10 — matches "Ten ramps" in `ramps.go:15` |
| variants | 97 × 7 = 679, the canvas's own two numbers (371 + 308) |

The last three hold over separately extracted data. The variants one follows from
the structure and is not independent evidence about the parser; what it does say is
that the canvas counted 97 forms.

**The one discrepancy: `diablo` has been redrawn.** Same ramp and same base colour,
but three of its five rows change — the horns go from `^ ╲ ╱ ^` to `^^ ╲ ^^`, the
base from `▝▙▄█▄▟▘` to `▝▙▄▀▄▟▘` and the feet come together. It is a design change
made after the code, not an extraction error.

## What is left to implement it

No data is missing any more. What is left is the work: `ATLAS.json` becomes the
97-form one, `Unlocks` goes from 14 to 42 entries with the ones from
`PUERTAS-97.json`, `sprites.go`, `ramps.go`, `evolution.go` and `names.go` grow by
the 56 new forms, the tests that today assert 41 forms and 287 variants become 97
and 679, and the `pet.json`s that already carry a mark whose gate changes have to be
migrated.

**A warning about the names.** `PUERTAS-97.json` and `ATLAS-97.json` name the forms
in Spanish, which is how the canvas names them. The code's **ids** are English
(`monk`, `gremlin`, `feral`) and are written into the `pet.json`s people already
have, so they are not renamed: `names.go` translates. Implementing this includes
giving each of the 56 new forms an English id and deciding the mapping — the gates
file does not carry it, and should not: it is the canvas's data, not the runtime's
table.
