# `ccpet invade`

A shmup played with the pet you already have. Your current form is the ship, its
trade decides the weapon, its mark refines it and its level scales it. Waves come
in from the right, a rival stands at every fifth one, and there is no last one.

## Why this is not the statusline

The first ask was to put the game in the grey gap of the footer. It cannot go
there. Checked against the installed CLI (`2.1.266`), not from memory:

| the game needs | the statusline gives |
| --- | --- |
| 20 frames a second | **one.** `statusLine.refreshInterval` is `min(1)` in the binary's own settings schema, in whole seconds |
| keypresses | **none.** It is a command that receives a payload on stdin, prints, and exits, once per refresh - no process is alive between frames and stdin is already spent |
| a key to swap between the prompt and the game | **no surface.** The ten hook events are `PreToolUse`, `PostToolUse`, `Notification`, `UserPromptSubmit`, `Stop`, `SubagentStop`, `PreCompact`, `SessionStart`, `SessionEnd`, `PermissionRequest`. None carries input, and nothing lets a plugin take the prompt line or paint over it |

Measured for the record: the gap in the footer is about **45 columns by 3 rows**
at 116 columns, and it shrinks as the bands fill up. Even as a non-interactive
animation that is a lane defender, not a shmup.

So: a real TUI in its own terminal, sharing the pet. **The statusline is not
touched by this feature at all.**

What that costs, plainly: switching between Claude and the game is the terminal
emulator's own tab key, because the two own different terminals. Inside tmux it
can be one keystroke instead, which is what `ccpet invade --split` is for - ten
lines, guarded on tmux being present, and never a dependency.

## The ship is the creature, so the field had to grow

The ship is `pet.Draw`'s five rows, the real sprite with the real ramp, not a
glyph that looks like one. Its hitbox is the **middle three** of those five: the
crest and the feet are cosmetic and things pass through them, which is what lets
a creature-shaped ship be a fair one.

That is what killed the 60x14 field the design started with:

| rows | field | places the ship can stand | lanes | share the ship's body blocks |
| --- | --- | --- | --- | --- |
| 14 | 12 | 8 | 8 | **37%** |
| 18 | 16 | 12 | 12 | 25% |

At 14 rows the player is standing in more than a third of the playfield with
nowhere to go when three lanes fire at once. **The floor is 60x18**, which still
fits inside a 24x80 terminal with room for a shell prompt. Below it the game
refuses and says both pairs of numbers, rather than drawing a mess.

One more rule falls out of the same arithmetic: **enemies only ever spawn in
lanes the ship's face can reach**. An enemy one row above the topmost face
position is unkillable, and unkillable means a life gone every wave for ever -
a bleed no amount of skill touches.

## There is no last wave

The design had 99 of them with a final boss. That is gone. Waves keep coming
while you can hold them, a rival stands on every fifth one as a checkpoint, and
the only thing that ends a run is reaching zero life. The record is the wave you
got to.

This is not a smaller design, it is a more honest one: a run ends when the ladder
outgrows you, at a point nobody typed. And it is testable in a way a number is
not - `TestARunPlaysItselfUntilTheLadderOutgrowsIt` plays a run to its end and
fails if the autopilot survives, because a ladder that never wins is a game with
no ending.

## Two axes are capped and two are not

That asymmetry is the whole of the difficulty design.

```
Seconds(n)  = min(15 + 3*(n-1), 180)         // 15 s at wave 1, +3 s each, flat at 3 min from wave 56
Squad(n)    = min(1 + n/10, lanes) * lanes/12 // how many arrive together
SquadEvery  = 40 ticks (2 s)
Count(n)    = Squad * Seconds * 20 / SquadEvery
HP(n)       = 1 + (n-1)/8                    // no ceiling
Speed(n)    = min(0.45 + 0.006*(n-1), 0.75)  // cells per tick
Boss(n)     = n % 5 == 0
BossHP(n)   = 40 + 25*(n/5)                  // no ceiling
```

**Capped**: the length, because a half-hour wave is not a wave; the speed,
because faster than three quarters of a cell per tick is not harder, it is
unreactable; and how many arrive at once, because a squad places at most one
enemy per lane and that is what keeps every allocation in the package bounded by
the size of the terminal rather than by a number read off disk.

**Uncapped**: the HP of the swarm and of the rivals. A kit tops out at level 6,
so there comes a wave your damage cannot clear, and that is the ending.

The duration is the primary dial and the count follows from it. Deriving it the
other way round - which is what the design did - gave 20 seconds at wave 1 and 52
at wave 99: one length with a rounding error, not short-to-long.

### The number that was wrong, and how it was found

The base speed was 0.18 cells a tick. At that speed one of the swarm takes
**twenty-two seconds** to cross eighty columns - longer than the whole of wave
one - so they piled up thirty-three deep before the first wave had finished
arriving. Playing all forty-one forms with an autopilot said every single-lane
family died before wave 3 whatever its level, which would have made most of the
tree unplayable.

At 0.45 the crossing is nine seconds and wave one holds four or five at a time.
Measured after the change, with an autopilot that dodges:

| form | level | wave reached |
| --- | --- | --- |
| `spark` | 1 | 5 |
| `bughunter` | 4 | 10 |
| `chimera` | 5 | 12 |
| `phoenix` | 5 | 31 |
| `wasp` | 6 | 33 |

A person plays better than the autopilot, so these are floors. Two tests keep
them honest: every one of the forty-one has to reach wave 3 at its own tier's
level, and a grown creature has to get more than twice as far as a larva.

## One kit per form, and how that is proved

Three layers, the same shape as the tree that produced them. The family anchor
comes off `pet.Lineage` - the rung-3 trade, or the shallowest form there is when
the lineage does not reach that far - then the mark modifies it and the level
scales it. Forty-one forms play differently without forty-one unrelated
mechanics.

`TestEveryOneOfTheFortyOneFormsFliesDifferently` compares the mechanical half of
a `Kit` - everything that changes how it plays, nothing that only changes what it
is called - across all six levels. Four collisions had to be broken to make it
pass, and three of them are in the design's own tables:

| collision | why | how it was broken |
| --- | --- | --- |
| `bloodhound` on `bughunter` | the mark adds homing and the family already homes | homing **and a point of pierce**: a bloodhound's nose does not stop at the first body |
| `cartographer` on `architect` | the mark adds a turret ability and the family's ability already is a turret | it drops two |
| `gremlin` | "random damage spikes" had **no field in the Kit at all** | `Spike`, a percent chance of a double-damage shot, and it is the only form that sets it |

The fourth is not in the tables, it is in the ladder: subtracting from cadence
per level piles every fast family onto the same floor, and a `bolt` at level 6
came out identical to a bare `sprinter`. Cadence is multiplied now, and no base
is under 8 ticks so the floor never engages.

Two more readings the design could not survive literally. "Every number times
one and a half" for a title would hand it a *slower* gun than its mark, since
cadence is a number where lower is better - so the ones that want to be big grow
and the two that want to be small shrink. And every family gets an ability,
because the space bar is the only thing the player times: the weapon is automatic
on purpose, since terminal key repeat is uneven across emulators and holding a
key to shoot feels broken through no fault of ours.

## Seven bodies by five traits, not thirty-five enemies

An enemy is composed the way a form is: a **body** gives the silhouette, the
width and the multipliers, and a **trait** gives one behaviour.

| body | glyph | width | hp | speed | | trait | what it does |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `mote` | `▪` | 1 | x1 | x1.0 | | `plain` | straight |
| `dart` | `»` | 1 | x1 | x1.6 | | `weaver` | drifts a lane up and down |
| `shard` | `◆` | 1 | x2 | x1.3 | | `darter` | lunges when it gets close |
| `spore` | `∘` | 1 | x1 | x1.2 | | `plated` | one point a hit unless the shot pierces |
| `husk` | `▚▚` | 2 | x2 | x1.0 | | `splitter` | leaves two motes behind |
| `slab` | `▰▰` | 2 | x3 | x0.7 | | | |
| `crawler` | `▬▬▬` | 3 | x4 | x0.5 | | | |

Thirty-five kinds. The glyph says the body and the colour says the trait, so they
are told apart at a glance with no legend. Writing thirty-five out by hand would
have given thirty-five unrelated cases that cannot be balanced and cannot be
proved distinct; this can, and is.

They arrive progressively, like the length: a body every three waves and a trait
every eight, so wave one is one mote going in a straight line and by the thirties
the whole zoo is out.

Two details that bite. `plated` against `Pierce` is the only place the bestiary
and the kits meet, and it is what gives the `cannon` family and the `sniper` mark
a reason to exist. And **a splitter's children never split**, or one lucky wave
is an allocation with no bound.

## The bosses are the creatures you did not become

A rival is not a bigger enemy with a different glyph. It is **one of the
forty-one forms you are not**, drawn with `pet.Draw` using its own sprite and its
own ramp, facing you.

| what a rival needs | where it comes from |
| --- | --- |
| its silhouette and colour | `pet.Draw` and `pet.RampOf` |
| its attack pattern | `KitFor(rival, level)` - forty-one behaviours already proved distinct |
| its phases | `pet.StateFor`: the seven vital states. It droops, its head goes down, and at zero it lies down |
| its name | `pet.NameIn` |

Forty-one bosses with their own identity, and no new drawing or behaviour code
for any of them. The phases come free and are legible: when the rival is worn
down its head drops and its cadence quickens, driven by the same `Vital` the
statusline has used since the beginning. Its hitbox is the middle three of its
five rows, like yours - symmetric, and one sentence to explain.

The roster is the seven trades, then the fourteen marks, then the fourteen
titles, and the two secrets last, which is what makes them rare. It is built by
walking `pet.Tree` in slice order rather than ranging over a map, so it is the
same in every process. It never sends the form you are flying: you do not fight
yourself. Thirty-seven rivals means the roster comes round every hundred and
eighty-five waves, and a rival that comes back comes back with the HP of the
wave it is standing on.

## Losing takes a level, and that is the one rule this breaks

The design set itself a hard rule: *a run must not be able to feed or starve the
pet - two systems, one file, no coupling.* Losing a run now costs the creature a
level, so half of that rule is gone, and it is written here rather than left to
be discovered.

What is kept is the half that matters. The game **can only ever take**: never
give XP, at most one level, only on a death, only through `pet.Update`, and never
on a quit, a Ctrl-C or a signal - leaving the game is not losing it. Every one of
those is a test rather than a comment, because they are all silent failures.

Two things make the stake playable rather than punitive.

**The shape does not follow the level down.** `FormSeen` and `floor` already held
the rung and `TestTheShapeHoldsWhileTheLevelIsStillAllowedToFall` already
guarded it. You lose a step of the kit, not the evolution you spent a week on -
and since the kit is `KitFor(form, level)`, the next run is measurably harder
until you feed the pet back up. That is the stake.

**And the drop is capped at a day's feeding.** This is the number that had to be
measured rather than argued. A run at level 4 lasts about three minutes before
the swarm outgrows the kit, and level 4 is deliberately the widest rung on the
tree - 1600 XP, roughly twelve days of real work. Dropping straight to the
threshold charged those twelve days for one bad three-minute run, and then again
three minutes later. With the cap, dying near a threshold still costs the level
and dying deep inside one costs a day and the level holds. That is the right way
round: the player with the most to lose is the one who just got there, not the
one who has been sitting on a full buffer for a fortnight.

A defeat is deliberately **not** a `Food`. `panel.Run` dispatches its meal verbs
by membership of that map, so a food called `defeat` would hand anybody a
`ccpet defeat` that punishes the pet by hand, and would put a row in the panel's
table of what it eats. It gets its label from the catalogue and one branch in the
panel's lookup instead.

## Why the tick is a pure function

`Tick(g Game, in Key) Game` takes a value and returns one. No terminal, no files,
no clock, and no `pet.json`: a run does cost a level, but that happens once, in
`run.go`, when the run is over. A tick that could reach the pet would punish it
twenty times a second. A source scan asserts it.

Randomness is splitmix64 over a `uint64` in the state rather than `math/rand`,
because a `*rand.Rand` is a pointer to mutable state and two copies of a game
would share one stream. Sixty-four bits in the save file, four lines of
arithmetic, and a resumed run plays the sequence it was in.

Everything else in the package leans on this. Two runs from one seed and one key
sequence end in identical states; a whole run plays itself headless with no tty
anywhere; and the wave, life, kit and rival rules are all tested by driving a
value rather than by looking at a screen.

## Why the pause is one file and not one per session

`Stop` fires when Claude finishes answering. The hook touches
`~/.claude/ccpet-stop` and the game watches its mtime four times a second.

One file rather than one per session, because the game has no session id and
cannot enumerate sessions, and *"a Claude I am running has finished"* is exactly
the question it wants answered. A leftover file never pauses a fresh run: the
mtime seen at startup is the baseline and only something newer counts. The game
never deletes it - two games may be watching, and a zero-length file that means
nothing to a new run is not worth the race.

Four times a second and not once every ten ticks: at ten the banner could be half
a second late, which is long enough to lose the wave you were being let out of.

Adding the event was four edit points, and the fourth is worth naming.
`hookEvents` and `hooks.json` are the obvious two, `hook.Run` the third - in the
first switch, before the tool dispatch, so a `Stop` carrying a tool name is not
counted as a tool call. The fourth is `HooksWired`, which spells the events out
one by one in prose, in two languages, with nothing tying it to the list. It is
tied now.

`dropOurHooks` needed nothing: it matches on the `ccpet` marker in the command
string, so uninstall already removes an event it has never heard of.

## Drawing

Full-frame repaint, one buffer, one flush, one write. Cursor home and an
erase-to-end on every row, never `\033[2J` - that is the flash. The
synchronised-output pair around it is ignored by terminals that lack it. No
diffing: sixty by eighteen is about ten kilobytes a frame and two hundred a
second, which is nothing, and a shadow buffer buys a whole class of stale-cell
bugs for no measurable gain.

The alternate screen and the cursor come back on every path, including a panic
and a signal. A game that leaves your terminal with no cursor is worse than one
that crashes.

The frame counter is divided by eight before it reaches `pet.Draw`, because the
walk cycle is `step%12 < 4`, calibrated for a statusline that refreshes once a
second. Handed a raw twenty-a-second counter, the feet strobe.

One bug here is worth recording because three width tests all passed over it.
`theme.Truncate` counts the bytes of an escape sequence as visible width and cuts
wherever it lands, so the Spanish HUD in a 76-column terminal printed a literal
`[38` where the life bar should have been - three cells wide, so every test
asserting "no wider than the terminal" was satisfied. Looking at an actual frame
found it in a second. The HUD drops whole parts now, most important first, and
never cuts; plain text is truncated before the escapes go on. `theme.Truncate` is
left alone, because it is right for the plain text the statusline hands it.
