# The pet's evolutions

The pet has **two layers that do not mix**:

| | what it measures | where it comes from | goes up and down |
| --- | --- | --- | --- |
| **life** | how it is doing *now* | context usage and quota | yes, all the time |
| **progress** | what you have done | XP banked in `~/.claude/pet.json` | up by eating, down from hunger |

**Life** picks the eyes, the feet and the colour — it is in [vitals.md](vitals.md).
**Progress** picks the silhouette, which is what this document is about.

## One template, seven states

Every evolution is a **template of 5 rows and 9 columns** with two eye slots and a
row of feet:

```
  |   |     <- the branch's mark
 ▗█┼█┼█▖    <- the evolution's body
▐█ o o █▌   <- the eyes are put there by the state
 ▝█┼█┼█▘
 ▘▘   ▝▝    <- the feet are put there by the state
```

The life state **changes neither the silhouette nor the hue**: it fills the slots
in and walks down the branch's ramp. The colour belongs to the evolution, not to
the state. That is how the 41 evolutions get their seven states without drawing
287 sprites.

The eyes follow a rule: the evolution puts its own in while it is intact (*fresh*
and *lively*), and from *easy* down the state is in charge
(`o o` → `▬ ▬` → `_ _` → `x x`). That way tiredness reads at a glance even when
you do not know which pet it is.

## The tree

```
spark
├─ pattern  compacts and commits short
│  ├─ refactor  many small diffs
│  │  ├─ surgeon.......  20 diffs in a row without a rejection
│  │  └─ weaver........  one refactor touching 10+ files
│  └─ tidy  never goes past 60%
│     ├─ monk..........  5 sessions never past 40% of context
│     └─ gardener......  docs and cleanup two days running
├─ probe  reads, plans, tests
│  ├─ bughunter  tests and fixes in a chain
│  │  ├─ bloodhound....  repro before the fix, 10 times
│  │  └─ exterminator..  15 green suites without a red one
│  └─ architect  long plans, docs
│     ├─ cartographer..  a 10-task plan closed all the way
│     └─ oracle........  5 plans written before touching code
└─ ember  goes to the limit without braking
   ├─ sprinter  short, fast sessions
   │  ├─ bolt..........  10 sessions under 15 minutes
   │  └─ sniper........  8 tasks closed with one single tool
   ├─ marathon  sessions of hours
   │  ├─ ox............  3 sessions of more than 4 hours
   │  └─ mole..........  5 days running in the same repo
   └─ feral  at the limit, never compacting
      ├─ gremlin.......  30 turns with permissions on bypass
      └─ kraken........  3 sessions touching 100% of context
```

### The titles

Behind every mark there is a title, and it asks for **more of the same habit**.
The fourteen numbers come out of the [«Cómo llegar a cada forma»][canvas] canvas,
which gives a factor per title rather than one multiplier: each habit was weighed
separately.

| mark | title | asks for | | mark | title | asks for |
| --- | --- | --- | --- | --- | --- | --- |
| `surgeon` 20 | `scalpel` | **50** diffs in a row | | `cartographer` 10 | `atlas` | **20** tasks in one plan |
| `weaver` 10 | `loom` | **25** files in one go | | `oracle` 5 | `sphinx` | **20** plans before code |
| `monk` 5 | `abbot` | **15** sessions under 40% | | `bolt` 10 | `storm` | **30** sessions under 15 min |
| `gardener` 2 | `forest` | **7** days of docs | | `sniper` 8 | `falcon` | **25** one-tool tasks |
| `bloodhound` 10 | `wolf` | **30** repros before the fix | | `ox` 3 | `mammoth` | **10** sessions over 4 h |
| `exterminator` 15 | `wasp` | **50** green suites in a row | | `mole` 5 | `worm` | **20** days in the same repo |
| | | | | `gremlin` 30 | `devil` | **200** turns on bypass |
| | | | | `kraken` 3 | `leviathan` | **10** sessions at 100% |

Two are not the canvas's number, and it is worth knowing why:

- **`atlas`.** The canvas asks for "5 closed plans of 10 tasks", which is a
  *count* of big plans. The counter that exists, `longest_plan`, is the longest
  plan ever closed — a maximum, not a count. Putting 50 there would ask for one
  fifty-task plan, which is a different thing, not a harder one.
- **`devil`.** The canvas asks for 100 turns on bypass. `bypass_turns` climbs
  about thirty times a day — measured — so 100 is a title that arrives already
  earned, and that is not a title. 200 puts it a week away, like the rest.

[canvas]: https://claude.ai/design/p/4639e060-9aec-4ae3-855a-f8530ae9ab34

**You do not choose the branch.** At each fork the behaviour counter that is
highest at that moment wins. Change your habits before you level up and you change
branch.

### "Highest" is not the raw number

The counters the forks read **are not the same kind of number**. Four climb once
per *event* and never stop — a commit, a `/compact`, a closed plan task — and five
climb at most once per *session*, when it closes. A day has a dozen commits and
three or four sessions, so comparing `methodical` with `impulsive` raw let the
units decide before the habit had opened its mouth.

Measured on a real `pet.json`, after three weeks working exactly at the limit:
`methodical` 39, `impulsive` 2. And the 2 is not for want of trying, it is the
ceiling. To be `ember` you had to bank more sessions at the limit than
commits+compacts **over the whole life of the pet**, which took out 3 trades, 6
marks and 6 titles: **16 of the 41 forms**, unreachable by playing.

It is the same defect that was fixed at level 5 — "the first to cross its
threshold wins" — one rung higher and worse, because here there is no threshold to
cross and therefore nothing to normalise against. Every counter is now divided by
its **scale**, which is what a day of that habit yields, and the fork goes back to
being a race between ways of working:

| Counter | Scale | Where it comes from |
| --- | --- | --- |
| `methodical` | 10 | commits at 12 xp, plus the compacts |
| `inquisitive` | 8 | green suites at 15 xp — where the hourly brake also lands |
| `diffs` | 10 | commits only |
| `tests` | 8 | the same brake |
| `plans` | 21 | plan tasks at 6 xp |
| `impulsive` | 3 | one per session ≥85% |
| `ctx_low` | 3 | one per session <60% |
| `ctx_maxed` | 3 | one per session ≥95% |
| `short_sessions` | 3 | one per session <15 min |
| `long_sessions` | 2 | one per session ≥90 min — fewer fit in a day |

The first five come out of the budget the design already had — a normal day is 128
XP, the same number behind the suite's hourly brake — divided by what each meal
pays. The five session ones cannot go past how many sessions fit in a day,
measured at three or four. They are a **calibration, not a law**, and they live in
`BranchScale` (`internal/pet/evolution.go`).

### And a branch you have taken defends itself

Dividing by the scale fixes *who* wins the fork, not *how often* it changes hands.
Two habits running level do not stay ahead of one another for long: measured,
`methodical` 4.60 days against `inquisitive` 4.38, which is 0.22 of a gap — **two
suites one way, three commits the other**. The pet changed name and sprite several
times in an afternoon, at a fork that had never actually been decided.

So a branch you have taken **defends itself**: to take it away you have to beat it
by a whole day of the habit (`BranchMargin`, 1.0, in the unit the scales already
put things in). A day is the choice and not half of one, because half a day is
five commits and that fits in an afternoon; a day of habit is not crossed there
and back inside one session. Checked against the real `pet.json`:

| inquisitive's lead | form |
| --- | --- |
| +0.80 days | `tidy` |
| +0.92 days | `tidy` |
| **+1.05 days** | `bughunter` |

It is a delay, not a lock: the counters only climb, so any branch can eventually
be taken — `TestADefendedForkCanStillBeTaken` demands it from both sides of every
fork, against every possible defender. What changes is the wait: the test's `ember`
player crosses level 2 on day 2 wearing whichever branch was a hair ahead at that
instant, and does not take the fork off it until day 8, when the lead is a whole
day.

**The price, with the eyes open.** The counters cannot say who was ahead yesterday
— they only climb — so the decision has to be **stored**: `branch` in `pet.json`,
one child per crossed fork. That breaks a property the rest of this file does hold:
the form stops being a pure function of the counters, and **two pets with the same
numbers can wear different forms** depending on the way they came. It is the only
fact about a pet that lives only in the file. It was known before it was chosen.

`RememberBranch` writes it, and `Save` writes it — not each caller — for the same
reason `RememberForm` does: six paths persist this file and only two had any reason
to think about branches. A fork that never gets written down is a fork with nobody
defending it, which is the hysteresis quietly not happening. And it only writes down
the forks **already crossed**: noting the level 3 one with the pet at level 2 would
hand it a defender chosen a level early.

On the way in, the field passes a stricter gate than the rung's: the pair has to
name a real choice — a fork the tree has, and a child of it — or it is dropped. The
level 5 forks do not go in: those are decided by `ripestMark` and are not defended.

What defends them is a test that plays, not one that fills counters in by hand:
`TestTheEmberBranchSurvivesANormalDayOfWork` simulates somebody who works at the
limit **and commits as well**, and demands both directions — that they reach `ember`
with a couple of commits a day, and that they do **not** if they commit all day long.
Because dividing is not putting a thumb on the scale.

**Why the old tests did not see it.** `TestEveryFormIsReachableFromAVeteran` writes
the counter by hand (`steer`, "one point above the sibling"), so it tests "reachable
if the number were higher", never "the number can get there by playing". And
`TestEveryTemperamentIsReachableByPlaying` did play, but only the **pure** player:
its `ember` case closes nothing at all, only `feed` and a high peak. The branch
looked alive and nobody who actually worked could take it.

**The chimera's tie was scaled too.** It compared the three temperaments raw, so the
secret went to whoever happened to have the same number of commits as suites — a
coincidence of units, not two ways of working that came out level. They now tie in
days. Chimeras already granted stay: `CheckSecrets` does not rewrite a secret
already set.

**And all three branches can be taken.** The impulsive one was dead: its counter was
only raised by *blowing the context*, which is the one meal that **subtracts** XP.
The arithmetic closed on itself — every point of `impulsive` cost 15 XP, and every
meal that gave them back fed a rival — so somebody blowing the context all day ended
up with 800 impulsive and nailed to level 1 with 0 XP. A third of the tree behind a
branch nobody could climb.

It now pays from the **session's context peak**: from `ImpulsivePeak` (85%) on it
counts as having worked at the limit, which is what the canvas asks for — «tira al
límite sin frenar», push to the limit without braking — and not the same thing as
crashing. It is the mirror of
`ctx_low`, which rewards whoever stays under 60 and leads to `tidy`.

**And one level down the same thing was happening.** `ctx_maxed`, which picks
`feral` out of its two siblings, had the arithmetic closed in exactly the same way:
its only source was the blow-up, while `short_sessions` and `long_sessions` were
collected for free just by having sessions. The three counters that read the ember
branch are now **three notches of the same gesture**, and none of them is paid for
in XP:

| Session's peak | Counter | What for |
| --- | --- | --- |
| ≥ 85 (`ImpulsivePeak`) | `impulsive` | level 2 — `ember` |
| ≥ 95 (`FeralPeak`) | `ctx_maxed` | level 3 — `feral` |
| ≥ 100 | `ctx100_sessions` | the `kraken` mark |

The blow-up (`overflow`) no longer feeds any habit: it is what it always should have
been, a −15 XP penalty that also breaks the clean streaks.

**Mind the tie.** A session over 90 minutes with the peak high raises `ctx_maxed`
*and* `long_sessions`, one each, and the counters stay tied for ever — but `marathon`
wins, because fewer long sessions than sessions fit in a day, and a long session is
therefore more of a working day than one at the limit. `feral` is for whoever fills
the window **fast**: short sessions at the limit. That is the distinction the branch
is drawing, and a test pins it
(`TestALongSessionAtTheLimitStillGoesToMarathon`).

**The tree's names are ids, not text.** `spark`, `bughunter` or `exterminator` are
what has been written in `pet.json` since the Python version, and renaming them would
rewrite every life file out there. What you read on screen is whichever language the
theme is set to — the canvas's Spanish column, *chispa*, *cazabugs*,
*exterminador*, or the ids themselves in English — and it lives in
`internal/pet/names.go`. A form with no reading falls back to its own id: a missing
name is a name yet to be chosen, not a bug. See [language.md](language.md).

Which of the two marks you get is not decided by the XP but by the **habit**: they
unlock on meeting their condition while in the parent evolution. The XP still sets
the *when* — they are level 5 forms — but it no longer hands anything out: from
there on the only thing that moves is the habit, and that is what the bar in band 4
and `/pet`'s `mark` row measure.

| Level | XP | Stretch | What you are |
| --- | --- | --- | --- |
| 1 | 0 | — | larva — `spark`, no feet yet |
| 2 | 60 | 60 | temperament — how you work |
| 3 | 180 | 120 | trade — what you are good at |
| 4 | 400 | 220 | the same trade, settled |
| 5 | 2000 | **1600** | mark — plus the habit condition, or a secret already earned |
| 6 | 4500 | 2500 | title — the branch's final form |

### Why level 4 is so long

The tree **only forks at levels 2, 3 and 5**. Level 4 hands out nothing: it is the
same trade, settled. That makes it the only stretch where the habit that decides the
mark keeps moving and **can still change its mind** — a `bughunter` who starts
reproducing bugs before fixing them leans towards `bloodhound`, one who chains green
suites leans towards `exterminator`, and either can overtake the other for as long
as the level lasts.

It used to last 500 XP. Measured against a real day of meals — about 12 XP a meal,
on the order of 60 an hour of effective work — that is **eight hours**: one long
session, and the fork decided before the habits had had a week to say anything. At
1600 the stretch is around **twenty-five hours** of work.

Through that stretch band 4 says `bughunter` and nothing else: the XP bar is the one
showing the progress, and which of the two marks is being earned is not announced —
it is still in dispute until the end, which is exactly what makes level 4 long.
`/pet`'s `mark` row does keep the count of both habits.

### The variant you are wearing: `bughunter[bloodhound]`

Band 4 writes **the mark the pet is wearing** in brackets, with the trade it is a
variant of outside:

```
bughunter[bloodhound] level 5 │ fresh ✦
```

It reads whole, as one name: *a bughunter, in its bloodhound form*. The tree forks
at levels 2, 3 and 5, and the mark is the level-5 fork — so the bracket appears
there and nowhere else:

| Level | Band 4 | Why |
| --- | --- | --- |
| 4 | `bughunter` | there is no variant to name yet |
| 5 | `bughunter[bloodhound]` | the fork, and which way it went |
| 6 | `wolf` | the title is the end of the branch and competes with nothing |

**The bracket used to say the opposite, and it misled.** It wrote the mark the pet
was *heading for*, so a level 4 read `bughunter[bloodhound]` without being a
bloodhound and with no guarantee of becoming one. The intent was for the brackets to
be the tense — a name says *is*, a bracket says *is heading for* — but that only
works if they can be seen: they were painted in `Rule`, the separator bar's colour,
which gives **1.54:1** against the background against the **11.8:1** of the two
words it sits between. What reached the eye was two bright words stuck together with
nothing in between, and it read as one compound word. The punctuation carrying the
whole meaning was the only part invisible. They are `Dim` now.

It is the first thing the band drops when it runs out of columns, after the speech
bubble. Losing the bracket costs something real — which of the two forms took the
trade — but the half left standing is still **true**: a bloodhound is a bughunter,
so a bare `bughunter` is less precise and is not false.

And two secrets off the tree: **phoenix** (reach hunger 10 and climb back to 0 in
the same session, only from `feral` or `marathon`) and **chimera** (two temperaments
tied on the way up to level 4; it inherits the eyes of one and the body of the
other).

Both are **level 5** forms and wait for 2000 XP like any other. The condition is met
earlier — the chimera's at level 4 — and in between the panel says what you are
heading for: `488 to chimera`. Handing it over on the spot skipped the whole of
level 4 and put a "level 5" next to 412 XP.

### A secret earns its own rung, not the one above

And it was the end of the road. `walk` returned the secret **before** walking the
tree, so the pet stayed on rung 5 for ever: no mark, no title, and the branch it was
on stopped meaning anything the day the secret landed. Three quarters of a branch in
exchange for a pretty form. A chimera is a level 5 form, not a headstone.

The tree is walked first now and the secret is only put on top if the tree returned
rung 5 **or lower**. The title is rung 6 and beats it, so it stays something a
chimera can turn into:

| Rung the tree gives | What it wears | Why |
| --- | --- | --- |
| trade (3) or mark (5) | the secret | it is rarer, and it is its rung |
| title (6) | the title | it is above, and it is paid for with the habit |

The marks skipped along the way are not a loss: the secret already occupies that
rung, and the title behind it asks for **the same habit**, more of it. The habit is
still the door; what changes is the form you wear while you go through it. The panel
says so — `22/50 to wasp` under `phoenix` — where before it was correctly empty
because there was nothing to point at.

And once it is on, the title is not given back: `tradeOf` of a secret is `""`, which
the floor reads as "I do not know which branch this comes from" and therefore keeps
the highest rung it stood on. A streak falling over does not take you from wasp back
to phoenix.

## A form does not go down a rung

The form is recomputed from the counters on **every refresh** and is not recorded,
so what the pet *is* can change from one line to the next. What it cannot do is go
down the tree.

Two habits go to zero when you blow the context — `test_streak` and `diff_streak`,
the two clean streaks — and without a floor that was a fall: a level 6 `wasp` came
back as `bughunter`, a level 3 form, with "level 6" written beside it. `pet.json`
now keeps the highest rung stood on in `form_seen`, and `pet.Save` writes it down
**on every write**, so no path can persist a pet and forget where it is.

The rule is that a form **does not fall**: it moves sideways, upwards, or across
branches.

| from | goes to | why |
| --- | --- | --- |
| `exterminator` | `bloodhound` | same rung, the other habit |
| `exterminator` | `wasp` | upwards, its title |
| `exterminator` | `bughunter` | **never**: that would be going down a rung |
| `wasp` | `bloodhound` | **never**: same trade, that is the streak falling over |
| `wasp` | `surgeon` | yes: a different trade, and that mark is earned |

The last two rows look like the same drop from 6 to 5 and are not, so the floor asks
**why** it came out lower. If the trade is the same, a habit has fallen and that is
exactly what the floor is there to stop. If the trade has changed, the pet has moved
branch and the new branch's mark is paid for: showing a title from a branch it no
longer stands on says less than showing what it is today. Below rung 5 nothing
happens in either direction — a walk that returns a bare trade is xp falling, or a
branch with nothing earned yet.

Two consequences worth knowing:

- **Changing temperament does change the form**, as soon as the new branch has a
  mark earned — and also if you already wear a title. What it does not do is leave
  you on the bare trade while nothing is earned there.
- **The level can go down**, and the form cannot. They are two different facts: the
  form is a watermark and the level is today's xp, which falls when the context
  blows (−15) and while the pet goes hungry. So `wasp level 5` is a pairing you can
  see, and it is not a bug.

## The food

| Meal | XP | Hunger | Cap |
| --- | --- | --- | --- |
| green suite | **+15** | −4 | one an hour, and with a change behind it |
| commit made | **+12** | −3 | — |
| `/compact` | **+8** | −3 | — |
| plan task closed | **+6** | −1 | — |
| `/feed` | **+3** | −2 | one every 4 h |
| context at 100% | **−15** | — | — |
| every hour at hunger 10 | **−1** | — | — |

And on session close, with no XP involved: a peak under 60% adds `ctx_low` (towards
`tidy`), a peak over 85% adds `impulsive` (towards `ember`).

**Why the green suite has a brake.** It was the biggest meal on the table and the
only one with no cap at all, so it was the one thing worth farming: running the suite
in a loop gave +15 every few seconds — 120 XP in eight minutes, measured in a real
session — and with that the ceiling and the drain were decoration. Nothing that
repeats in nine seconds can be worth a fifteenth of a level.

There are two brakes, because they solve different things:

- **One an hour.** Not an arbitrary number: the canvas budgeted level 5 at "a week
  of normal use", that is about 128 XP a day, and eight green suites in a working day
  is exactly that. The brake is still calibrated there; what changed is the
  destination, not the pace — level 5 moved further away on purpose so the habit's
  fork has time to decide itself.
- **And with a change behind it.** A suite that passes without you having edited
  anything is not work, it is the same suite again. The hook already knew which tools
  are used, so remembering whether there has been an `Edit` since it last paid is
  enough. The bloodhound's red → green cycle is booked either way, even when the
  suite does not pay: reproducing a bug counts on its own.

Every meal carries **its own clock** (`meals` in `pet.json`). There used to be one,
`fed_at`, which was enough while `/feed` was the only one with a wait; two meals with
brakes would have gagged each other.

**Hunger** climbs +1 an hour without food, up to 10. From 7 the eyes go out and the
pet asks for food in the statusline. At 10 it stops being a warning and **starts
costing 1 XP an hour**, which is the only way the pet has of losing ground on its
own. **It never dies**: at the bottom it stays a larva, which is a form, not a grave.

Blowing the context costs 15 XP and breaks the clean streaks.

### Why the level does go down

The original design said the level never goes down, and under that rule a pet that
reached the top stayed there for ever: nothing to gain and nothing to lose. The
ladder ended and the tamagotchi stopped being one.

Two changes correct it without touching the tree, because the machinery for going
down was already whole — `LevelFor` follows the XP in both directions, and the form
comes down with the level — what was missing was something that actually subtracted:

- **The XP has a ceiling**: `XPCeiling`, the last threshold plus one level-1
  stretch. Without it the XP was a moat. At 1641 points with the top at 900 it took
  fifty blown contexts to lose a level, so any penalty drowned in the buffer before
  it meant anything.
- **Hunger at the cap drains.** Half a day away costs nothing; at **two and a half
  days** you lose the last level. That number does not move when the ladder is
  touched, because the ceiling is defined *relative* to the last threshold: it is the
  60 hours of the level-1 stretch, always.

  Falling all the way, on the other hand, does scale with the ladder, and it is now
  **190 days** — it used to be about 80. The doc said "six weeks" and had not been
  true for a while. If 190 days looks too forgiving, the thing to move is `StarveXP`,
  not the ceiling.

The figures live in `StarveXP` and `XPCeiling`, and there is a test
(`TestTheCostOfNeglectIsWhatWeMeantItToBe`) that argues with whoever moves them.

## What feeds each counter

The hook (`ccpet hook`) and the statusline itself turn what you do into counters.
**All 41 evolutions are reachable**: the root, the three temperaments, the seven
trades, the fourteen marks, the fourteen titles and the two secrets. And they stay
reachable with the pet already grown, which is what
`TestEveryFormIsReachableFromAVeteran` pins: at every fork the winner is the habit
that has got furthest *relative to what it asks for* — its threshold at level 5, its
scale at levels 2 and 3 — and not the first to cross a line, so no door closes behind
you.

| Counter | Filled by | Who sees it |
| --- | --- | --- |
| `methodical` | commits and `/compact` | hook |
| `inquisitive` | tests and plan tasks | hook |
| `impulsive`, `ctx_maxed` | the session's context peak (85 / 95) | statusline → `SessionEnd` |
| `diffs`, `diff_streak` | commits (the streak breaks on a blow-up) | hook |
| `tests`, `test_streak` | green suites (same) | hook |
| `widest_commit` | the highest `N files changed` | hook |
| `longest_plan` | the longest plan closed all the way | hook |
| `plans_before_code` | a plan of 3+ tasks written before editing anything | hook |
| `single_tool_tasks` | tasks closed using one single tool | hook |
| `repro_before_fix` | a red test followed by a green one | hook |
| `docs_days` | days running with a docs or cleanup commit | hook + `git show --numstat` |
| `bypass_turns` | new prompts with permissions on bypass | statusline + transcript |
| `ctx_low`, `sessions_under_40` | the session's context peak | statusline → `SessionEnd` |
| `short_sessions`, `sessions_15min`, `long_sessions`, `sessions_4h` | the session's duration | same |
| `ctx100_sessions` | touching 100% of context (the third notch) | same |
| `same_repo_days` | days running closing a session in the same repo | same |

Three facts are **seen only by the statusline**, because they reach no hook: the
context usage, the permission mode and the token pace. It leaves them in its scratch
file and the `SessionEnd` hook turns them into counters.

The **permission mode** deserves a note: it does not come in the statusline's
payload, but it does come in the transcript, whose path does arrive. The tail of the
file is read (32 KB, 0.02 ms) looking for the last `permissionMode`. It is the only
way for `gremlin` — "30 turns with permissions on bypass" — to be reachable.

## How the things the CLI does not say are worked out

Two meals and four marks come from **inference**, not from anything the CLI exposes.
The rule while writing them has always been the same: **rather miss a meal than
invent one.**

**Green suites.** Three layers, hardest to softest:

1. An `is_error` from the CLI beats everything: it is the exit code, and the only
   hard fact there is.
2. The red patterns are looked for **in the last twelve lines only**, which is where
   the summary goes. Looking for them in the whole output made a test called
   `test_login_failed` paint a green suite red.
3. No green pattern is needed. If the command was a runner and it came out fine, it
   counts — which is how runners not on the list get in.

To recognise the runner there is a long list (pytest, jest, vitest, go, cargo,
phpunit, rspec, mvn, gradle, dotnet, swift, flutter, mix, make/just/task…), a last
resort by the **executable's name** (`run-tests.sh`, `testear.sh`, `bin/spec` count;
shell's `test` does not, being a file comparison), and `PET_TEST_RUNNERS` for a regex
of your own.

And most importantly: the runner has to be **in command position**, not anywhere in
the text. The command is split on the shell operators (`;`, `&&`, `||`, `|`) and each
piece is looked at from its start, skipping environment assignments and `sudo`.
Without that, an `echo "run pytest"` or a `grep -rn "go test"` counted as a green
suite — the same defect commit detection already had, fixed here the same way.

**Commit made.** `git … commit` has to be at the start of the command or behind a
shell operator. Without that anchor, a `grep -rn "git commit"` counted as a commit.

**The four inferred marks.** None of them guesses at intent: they all look at a
checkable fact that resembles it closely.

| Mark | What the design asks for | What is actually looked at |
| --- | --- | --- |
| `bloodhound` | repro before the fix | a red test followed by a green one in the same session |
| `oracle` | plans written before touching code | a `TodoWrite` of 3+ tasks before the first `Edit`/`Write` |
| `gardener` | docs and cleanup two days running | `git show --numstat` of the commit: mostly `.md`/`docs/`, or a big deletion |
| `sniper` | tasks closed with one single tool | distinct tools used between two closed tasks |

They are approximations and they get it wrong: a red from a network failure and a
green afterwards count as repro→fix even if you fixed nothing. But they get it wrong
**on the low side** almost always, and none of them invents a fact that did not
happen.

## The controls

```bash
pet                    # the panel: level, evolution, xp, hunger, streak and today's food
pet feed               # +3 xp, hunger −2, one every four hours
pet count <c> [n]      # add to a behaviour counter
pet record <c> <v>     # keep a counter's maximum
```

Installed as `/pet` and `/feed` by `scripts/install.sh`.

To start from scratch: `rm ~/.claude/pet.json`. To release just the defended
branches and let them be recomputed: remove the `branch` key.

---

See also the [README](../../README.md) for the bands and the palette, and
[vitals.md](vitals.md) for the other layer, the one of the moment.
