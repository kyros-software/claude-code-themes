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

## The creature keeps its crest, and it is at the bottom

It is `pet.DrawCard`: four rows of nine cells, the crest over the three compact
ones. It is by a distance the biggest thing on the field, and that is the point
of the whole exercise - the swarm is the arcade's, and the thing shooting back at
it is your pet.

Three smaller versions were built and thrown away, and the third is the one worth
recording.

Cropping the compact form to two rows lost little and gained little. Then the
fair question: can the sprite not just be **scaled**? It can, and it is
arithmetic rather than guesswork - the block glyphs really are pixels, each one a
2x2 patch, so a 9x3 sprite is an 18x6 image and halving it is a downsample. It
was tried. Four of the forty-one came out as the same five glyphs and every one
of them lost its eyes. **A cell is the floor of what a terminal can draw, and the
compact form is already standing on it**: anything smaller has to be drawn, not
derived.

So a five-cell version was drawn - the middle of the crest over the eyes - and it
worked, in the sense that it fitted and told 31 of the 41 apart. It was still the
wrong answer, and finding out why is what settled the design: what the creature
has to keep is its **crest**, the antennae and horns that say which of the
forty-one it is, and the crest is precisely the thing `DrawCompact` throws away
to get down to three rows.

Hence the card, which is the compact form with the crest put back on. The
direction was down and the answer was up.

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

## One glyph an invader, and why it took three tries

The swarm is the arcade's three - squid, crab, octopus - drawn in **one cell
each**. Two bigger versions were built and thrown away first, and both were
wrong in a way that only shows when you look at a screen:

| try | what it was | why it went |
| --- | --- | --- |
| forty, off the design canvas | 5x3 cells, `> <` eyes, eight stages | inventive, and it did not look like Space Invaders - the one thing a game called invade has to do |
| the arcade's own pixel grids | 11x8 pixels packed into 12x4 cells with half blocks | looked exactly right and was **enormous**: one invader three times the creature you steer |
| one glyph | `Ψ` `Ж` `Щ` | the cabinet's own arithmetic fits: eleven columns by five rows, fifty-five on screen |

At one cell the silhouette is gone and what is left is a glyph picked for its
shape: antennae, arms out, legs down. What it buys is the formation - the real
one, the one everybody pictures - and a creature at the bottom that is plainly
the biggest thing in the game, which is the joke.

Three species and not forty because the arcade had three. What a later wave
changes is the **colour and the price**: the stage climbs every four waves
through the canvas's eight palettes, and a kill pays its species' arcade value -
30, 20, 10 - multiplied by the stage. So the top row is worth three of the bottom
one at every depth, which is the arcade's own reason to shoot the squids first.

One thing had to be given back. A one-cell target hit by a one-cell bullet is not
the arcade's game, it is a coin toss: the cabinet's aliens are eight to twelve
pixels across and its bullet is one. So an invader is **three cells wide to a
bullet** and one to the eye. On a five-cell pitch that leaves two columns of
clear air between neighbours, so aiming still means something and a miss is still
yours.

## A shot has to hit every row it crosses

A bullet travels 1.1 rows a tick, which means it does not land on every row: over
a flight it steps clean over about one row in eleven. Testing only the row it
landed on therefore misses, at random, whichever row the arithmetic happens to
skip - and the row it skipped most visibly was the top one, at field row zero,
because the next step takes the bullet off the field, where it was thrown away
before anything was tested against it.

From the outside that is a top row you cannot kill until the block drops a step,
which is how it was reported after five minutes of play. It is also the kind of
bug that hides: every other row worked, and the one that did not moved around
with the height of the terminal.

So a shot now resolves against every row between where it was a tick ago and
where it is, lowest first, and it is culled after that rather than before. The
regression test fires at a lone invader on every row of five different terminal
heights; against the old code it fails thirteen times.

## Moving and firing at once, which a terminal does not want to allow

The creature **latches**: an arrow sets it going and it keeps going until you
point it the other way or press down to stop.

This is not how the cabinet felt and it is the only thing that works. A terminal
has no key-up event and no way to say two keys are down at once. Hold left and
the operating system streams left; press fire and it starts repeating *that*
instead, and the left never comes back until you let go and press it again. So
the first version stopped dead the moment you shot.

Momentum was tried next - keep sliding for three tenths of a second after the
last arrow - and it is not enough either: hold the fire key and you coast to a
halt just the same. Reported twice, from actual play, before the shape of the
problem was clear.

Latching costs precision, so the down arrow is a brake and a wall is a stop -
leaving the creature latched against one would mean the next key you press is a
key you did not know you had to press. And draining the key channel prefers an
action over a direction, for the same reason: a dropped direction costs nothing
because the latch carries it, while a dropped shot is a press the game ignored.

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

## The arena: the game opens itself while Claude works

`ccpet arena on`, and a turn is a round. Submitting a prompt opens the game in a
window of its own and hands it the keyboard; the `Stop` that ends the turn pauses
the game and puts the focus back in the tab you typed in. You play while Claude
works and you stop where you were when it is done, without touching either
window yourself.

Three rules, and they are what the tests are about.

**One game, however many Claudes are open.** The running game writes
`~/.claude/ccpet-invade.live` once a second; a prompt reads its mtime, and the
session that finds it stale is the only one that opens a window. Everyone else
raises the window that is already playing.

Three seconds of staleness, which is three missed beats: one is a busy machine
and three is a window that has been closed. The alternative - a pid file, checked
with `kill(pid, 0)` - does not work here at all. Under gnome-terminal every
window belongs to one `gnome-terminal-server` process, so the pid the launcher
gets back is a dbus client that has already exited, and the game is a grandchild
of a process that outlives every game. The heartbeat is written by the only party
that knows, which is the game.

The claim is taken **before** the window is opened and under `lockfile`, and the
file's contents are the beating process's pid so only that process can drop it.
Both matter. Two sessions answering in the same instant would otherwise both
look, both see nothing playing, and both open a window; and a second game started
by hand and quit would otherwise free a slot the arena's game is still holding.

**Off until somebody says otherwise.** The switch is `~/.claude/ccpet-arena`, a
file whose existence is the setting. Not a key in `ccpet.json`, which is read by
a hook on every prompt of every session and would have to be parsed and merged to
be written: a stat is cheaper, it cannot lose somebody's language setting to a
botched write, and it can be found with an `ls` by somebody wondering why
terminals keep opening. A theme that starts opening windows because it was
installed is a theme that gets uninstalled.

**Never an error the turn can see.** No X, no window manager, an emulator nobody
has heard of, a machine over ssh: every piece of this degrades to doing nothing,
which is the same as the arena being off. `ccpet arena on` is where that is said
out loud, because it is the only moment somebody is listening - the alternative is
turn after turn of nothing happening and no way to tell why.

### The protocol is two mtimes

| file | touched by | read by | means |
| --- | --- | --- | --- |
| `ccpet-stop` | `Stop` | the game, 4×/s | Claude has answered: pause |
| `ccpet-play` | `UserPromptSubmit` | the game, 4×/s | a turn has started: play |
| `ccpet-invade.live` | the game, 1×/s | every prompt | a game is running, and whose |

Two files rather than one with a word in it: the signal is the mtime, and a mtime
is something both sides compare without reading, parsing or locking anything.
Which of the two moved last is the whole protocol, and `fileWatch` is nine lines
because both halves are the same nine lines.

A pause the *player* asked for with `p` is not lifted by the next prompt, and that
is what the banner is checked for: somebody who stopped the game to go and read
something did not ask for it back. Claude only undoes Claude's pause.

### A window, not a tab, and a title rather than a pid

The focus is moved with `wmctrl -i -a <id>` and read with
`xprop -root _NET_ACTIVE_WINDOW`. Both are X11, both are looked up rather than
depended on, and Wayland answers neither.

The game gets its **own window** and not a tab beside Claude, because no terminal
emulator has a command line that selects a tab: `gnome-terminal --tab` opens one
and then nothing can bring it forward again. A window is a thing `wmctrl` can
raise. Which is also why the game names its window - `\033]0;ccpet invade\007` on
the way in, an empty title on the way out - since a title is the only handle a
window manager offers for "the window with the game in it".

The window the focus goes **back** to is the one that had it when the prompt was
submitted, which is the tab you typed in, because typing is what gives a window
the focus. It is read first, before the game is raised over it, and it is
remembered per session in `$TMPDIR` with the rest of the session's scratch - not
in one file like the pause, because two sessions waiting on two answers go back
to two different tabs. A prompt that arrives from the game's own window has
nowhere to hand anything back to and says so.

### What the wiring costs

Five edit points, the same shape as `Stop` and with one more trap.
`hookEvents`, `hooks.json`, `hook.Run` and `HooksWired`, as before - and the trap
is that **whatever a `UserPromptSubmit` hook writes to stdout is appended to the
prompt**. A stray line from this code would arrive as something the user said, so
the hook prints nothing at all, and `openArena` gives the emulator no stdin,
stdout or stderr: a hook holding its pipe open until the game is over is a turn
that never starts.

`--split` stays as it was, for tmux. The arena is what the same idea looks like
when there is no multiplexer to lean on, which is most desktops.

## The keyboard was dead, and every test passed

Worth writing down, because the shape of it will happen again.

Raw mode's obvious setting is VMIN 0 with VTIME 1: *come back in a tenth of a
second with whatever there is*. It does not work here. `os.File` turns a read
that returns nothing into `io.EOF`, so the goroutine reading keys saw an EOF a
tenth of a second after the game started, took it for a closed terminal and
returned. Not one keypress reached the game for the rest of the run.

Every test in the package passed, because they all drive the loop through a fake
key channel and none of them goes near a terminal. What found it was a pty: fork
one, run the binary in it, write bytes at it and read the frames back. That is
also how the diagnosis got sharp - `a` and the arrows behaved identically, which
ruled out the decoder and pointed straight at the reader.

Three things came out of it. Raw mode blocks for a key now (VMIN 1, VTIME 0), and
the flags it computes are a pure function so there is something to assert. The
reader tolerates a budget of empty reads before it gives up, so getting the
termios wrong again is a game that keeps playing rather than one nobody can
steer. And both are regression tests that fail against the old code.

## Drawing

Full-frame repaint, one buffer, one flush, one write. Cursor home and an
erase-to-end on every row, never `\033[2J` - that is the flash. The
synchronised-output pair around it is ignored by terminals that lack it. No
diffing: sixty by eighteen is about ten kilobytes a frame and two hundred a
second, which is nothing, and a shadow buffer buys a whole class of stale-cell
bugs for no measurable gain.

Colour is emitted per RUN and not per cell. A row of eleven identical sprites is
one colour and sixty-odd glyphs; wrapping each of them cost about ten kilobytes a
frame and two hundred a second, which is fine on a local terminal and is not fine
down an ssh connection.

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
