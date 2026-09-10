# `ccpet invade`

Space Invaders played with the pet you already have. The creature sits at the
bottom of the screen and runs along the floor; the swarm comes down from the top
in a block that walks sideways and steps down at the walls. Your form decides the
weapon, its mark refines it and its level scales it. A boss stands at every fifth
wave, and there is no last one.

You aim it and you fire it. The first draft did neither - the gun fired by
itself and tracked the nearest enemy, and the player's only verb was to move -
which left nothing to be good at. Now moving IS the aiming: a shot leaves the
middle of the creature and goes straight up.

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

## The creature is small, and it is at the bottom

It is `pet.DrawCompact`: three rows of nine cells, the same small form the
statusline uses. The five-row card is the pet's portrait; this is its cannon, and
a cannon that took a quarter of the screen would leave nowhere to dodge to.

Everything else follows from the block having to fit above it. The floor is
**60x18**, and inside it:

| | how it is derived | at 80x24 |
| --- | --- | --- |
| formation columns | `(cols-2)/6`, clamped to 4..11 | 11, the arcade's number |
| formation rows | `(rows-ship-4)/4`, clamped to 2..5 | 3 |
| the creature | three rows on the floor, nine wide | 3x9 |

Both are derived rather than fixed so that the game is the same shape of problem
in a narrow window as in a wide one, and so the block always starts clear of the
creature. Below the floor it refuses and says both pairs of numbers rather than
drawing a mess.

## Two ways to lose, and the second one is the clock

Bombs whittle you down: one life each, two from a boss, and you can dodge them.
That is the slow way.

The fast way is that **they land**. When the block reaches the creature's row the
run is over, whatever life you had left. It is the arcade's own rule and it is
what stops a slow gun from simply waiting a wave out - without it, a patient
player with a weak kit could clear any wave eventually, and the descent would be
scenery.

The block walks sideways at the wave's own pace and steps down a row every time
it reaches a wall, and it **quickens as it empties**: the last three come down
fast. That is the arcade's most famous accident, kept on purpose, because an
almost-cleared screen must not be a slow one. Clearing a flank also widens the
block's runway, since the edges are taken from the members that are still alive -
which is what makes shooting the outside columns first a tactic rather than a
habit.

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
Stage(n)   = min(1 + (n-1)/4, 8)          // which of the canvas's eight it draws from
HP(n)      = 1 + (n-1)/10                 // hit points per member; no ceiling
Step(n)    = clamp(26 - n/2, 5, 40)       // ticks between sideways steps
Drop(n)    = clamp(70 - n, 14, 70)        // ticks between bombs
Boss(n)    = n % 5 == 0
BossHP(n)  = 30 + 20*(n/5)                // no ceiling
```

**Capped**: how fast the block walks and how often it bombs, because past a point
faster is not harder, it is unreactable; and the size of the block, which is the
terminal's business and not the wave's.

**Uncapped**: the hit points of a member and of a boss. A kit tops out at level
6, so there comes a wave your damage cannot clear before the block lands. That is
the ending, at a point nobody typed.

## The rule that makes it a game rather than a hose

Only two of your presses may be in the air at once.

The arcade allowed exactly one shot on the screen, and that is what turns every
press into a decision: miss, and you wait for it to reach the top before you may
try again. Two is the concession to a terminal, where a frame is fifty
milliseconds and one would feel like lag rather than like discipline.

It is not a detail. Measured with an autopilot, without any cap at all a
level-six title cleared **forty-five waves in three minutes** - four seconds a
wave - because nothing limited how much lead was in the air. With it the same
creature takes ten to fifteen seconds a wave, and a run is a session:

| form | level | wave reached | seconds per wave |
| --- | --- | --- | --- |
| `spark` | 1 | 5 | 40 |
| `refactor` | 3 | 10 | 35 |
| `bughunter` | 4 | 25 | 29 |
| `marathon` | 4 | 50 | 16 |
| `wasp` | 6 | 65 | 14 |
| `phoenix` | 5 | 160 | 10 |

A person plays better than the autopilot, so these are floors. The spread across
forms is the point: `marathon` is the cannon family, and a shot that pierces
three deep is worth more against a block eleven wide than a faster gun is.

One volley always fits under the cap whatever the ceiling says, and that is not
pedantry: a `loom` at level six fires seven projectiles at a time, and with a
flat cap of six it could never fire at all - every press refused, for the length
of the run. Its own playability test caught it.

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

## Forty off the canvas, mixed by stage

The bestiary is not invented here. It comes off the design canvas as
`Bichitos por Stage`: **forty troop sprites in eight stages**, three rows of five
cells with two leg frames, plus a seven-tone ramp and a points value per stage.
The bigger ones come from `Sprites Marcianitos v2`: **thirty-five in five ranks**,
five rows of nine cells, the last four of which the canvas calls *jefes*.

They are drawn the way the canvas says to draw them: the body in the stage's own
tone and the three cells of eyes in the light one. That is the only thing that
breaks the flat colour, and it is what makes a screen of thirty-three readable at
a glance.

A wave is **not** a stage, though the canvas groups them that way. It mixes the
stages up to the one it has reached, deepest at the top and one shallower each
row down, so a later wave looks like an army rather than like a colour swatch.
The line-up is deterministic in the wave number and not in the seed, which means
wave twelve is the same twelve every time you reach it - and a wave you can learn
is worth more than a wave that is fresh.

A guard test measures every one of the seventy-five sprites: each row exactly as
many cells wide as its grid claims, eyes on every one of them, and two leg frames
that differ. They arrive as pipe-separated strings pasted out of a canvas, which
is a format that loses a character quietly.

## The bosses climb with the stages

A boss is one of the thirty-five, and which one is tied to the **stage the troops
have reached** rather than to a count of bosses. Walking the roster in order
spends nine bosses - forty-five waves - on the rank the canvas calls larvae, and
almost nobody gets that far, so the four called *jefes* would never be seen. Tied
to the stage they turn up around wave thirty, while a run is still going:

| wave | stage | rank | boss |
| --- | --- | --- | --- |
| 5 | 2 | 1 | mota |
| 10 | 3 | 2 | orbe |
| 20 | 5 | 3 | lancero |
| 25 | 7 | 4 | acechador |
| 30 | 8 | 5 | reina |

It moves side to side, quickens as it is worn down, leans a row lower every few
seconds, and fires three bombs at a time. Clearing it heals you to full, which is
what makes every fifth wave a rhythm rather than a countdown.

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

The frame counter is divided by eight before it reaches `pet.DrawCompact`,
because the walk cycle is `step%12 < 4`, calibrated for a statusline that
refreshes once a second. Handed a raw twenty-a-second counter, the feet strobe.

The creature is the one thing not painted cell by cell: `pet.DrawCompact` hands
back a whole painted row, so it claims nine columns of the grid and the assembly
steps over them. Re-implementing its painter to get cells would be a second copy
of the one thing `internal/pet` is for.

One bug here is worth recording because three width tests all passed over it.
`theme.Truncate` counts the bytes of an escape sequence as visible width and cuts
wherever it lands, so the Spanish HUD in a 76-column terminal printed a literal
`[38` where the life bar should have been - three cells wide, so every test
asserting "no wider than the terminal" was satisfied. Looking at an actual frame
found it in a second. The HUD drops whole parts now, most important first, and
never cuts; plain text is truncated before the escapes go on. `theme.Truncate` is
left alone, because it is right for the plain text the statusline hands it.
