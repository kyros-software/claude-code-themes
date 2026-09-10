# `ccpet invade`

A terminal shooter played with the pet you already have. A representation of your
creature sits at the bottom and runs along the floor; a fleet of ten kinds of
enemy ship comes down from the top, each falling and drifting and shooting on its
own clock. Your form decides the weapon, its mark refines it and its level scales
it. Asteroids fall on everybody, health kits fall for you, the score buys
upgrades, a boss off the design canvas stands at every fifth wave, and there is no
last one.

You aim it and you fire it. The first draft did neither - the gun fired by itself
and tracked the nearest enemy, and the player's only verb was to move - which left
nothing to be good at. Now moving IS the aiming: a shot leaves the middle of the
ship and goes straight up.

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

## The player is a representation of the creature, not the creature

For a while the cannon WAS the pet: `pet.DrawCard`, four rows of nine cells,
crest and antennae and its own colour ramp. It is the best drawing in the repo
and it was the wrong thing to fly.

| | the creature | the representation |
| --- | --- | --- |
| size | 9x4 | 5x3 |
| drawn by | `internal/pet` | `ship.go`, in the fleet's own line art |
| identity | the whole sprite | colour, silhouette and eyes |
| against a 5-cell enemy | twice its width | its width |
| where it may go | the floor, left and right | half the field, all four ways |

Three things carry the identity now, and each of them says something the player
already knows:

- the **colour** is the form's own ramp, so a bughunter is the bughunter's green
- the **silhouette** is the weapon's family, so the thirteen fly visibly apart
- the **eyes** are the health: `o o`, then `- -`, then `x x` when the run is over

```
   \ /        -+-        [^]         |          ^-v
  <o o>      [o o]      |o o|       =o o=      {o o]
   /^\        /_\        /|\         /_\         /v\
  homing     steady     turret      cannon     chimera
```

Getting here took four goes at the player and three at the enemy, and the order
matters: the creature stopped being the ship the moment the enemies stopped being
one block. Nine cells of pet against fifty-five one-glyph invaders was a joke
that read; nine cells of pet against a fleet of five-cell ships was just a wide
target.

What was NOT done, twice tried: shrinking the sprite. Cropping the compact form
to two rows lost the crest, and a true downsample - the block glyphs are 2x2
pixel patches, so halving a sprite is arithmetic rather than guesswork - turned
four of the forty-one into the same five glyphs. The answer was not a smaller
creature. It was a different drawing that means the same thing.

## Two ways to lose, and neither of them is the clock

Bombs whittle you down: one life each, two from a boss, and you can dodge them.
A ship that reaches the floor costs one more, or two if it comes down on top of
you, and then it is gone. Letting one through is a mistake, not a defeat: a fleet
is individuals.

The fast way is a **boss landing**. When one reaches your row the run is over
whatever life was left, and that is the fight lost rather than a slip. It is the
last thing left of the arcade's "they land and it is over" rule, and it is kept
for exactly one reason: without it a patient player with a weak gun could stand
under a boss for as long as it took, and the descent would be scenery.

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

## Three axes are capped and one is not

That asymmetry is the whole of the difficulty design.

```
Stage(n)   = min(1 + (n-1)/4, 8)          // how deep into the fleet it may draw
Count(n)   = clamp(6 + n, 6, 30)          // ships released over the wave
Every(n)   = clamp(60 - 2n, 12, 60)       // ticks between releases
Pack(n)    = clamp(1 + n/12, 1, 3)        // ships per release
Tough(n)   = (n-1)/6                      // hit points added to each; NO ceiling
Haste(n)   = min(1 + n/50, 2)             // multiplier on fall and drift
Boss(n)    = n % 5 == 0
BossHP(n)  = 45 + 40*(n/5)                // no ceiling
```

**Capped**: the pace of the releases, the speed of the ships, and how many are
allocated at once. The first two because past a point faster is not harder, it is
unreadable; the third because `release` cannot ask for memory without a limit off
a wave number that came out of a file.

**Uncapped**: the hit points. A kit tops out at level 6 plus whatever upgrades a
run has bought, so there comes a wave whose ships you cannot clear before they
are on top of you. That is the ending, at a point nobody typed.

Measured with the autopilot, which is a poor player on purpose - it does not
dodge, it does not pick off what is about to land, and it spends its ability the
moment it is ready:

| form | level | waves reached |
| --- | --- | --- |
| `spark` | 1 | 5 |
| `pattern` | 2 | 5 |
| `bughunter` | 4 | 12-13 |
| `architect` | 4 | 15-18 |
| `marathon` | 5 | 13-14 |
| `wasp` | 6 | 13-18 |
| `leviathan` | 6 | 22-23 |

Two things to read off that table. The pet's level is worth roughly four waves a
rung, which is what makes feeding it worth doing; and the first boss is a real
gate at level one, which is intended - a larva is meant to lose to it.

## The magazine, which replaced a cap on shots in the air

Something has to make a press cost something, or the game is a hose.

The arcade's answer was **one shot on the screen at a time**: miss, and you wait
for it to reach the top. This had it as two - fifty milliseconds a frame makes one
feel like lag rather than like discipline - counted in projectiles so that a
seven-shot volley still fits.

The reference's answer is a **magazine**, and for a fleet it is the better one.
What limits you is how much you can shoot before you have to stand still and
reload, not how far your last shot has travelled - and standing still is a real
cost when six ships are coming down at their own angles.

| | rounds | cadence | empties in | reloads in |
| --- | --- | --- | --- | --- |
| `spark` at level 1 | 9 | 12 | 108 ticks | 70 |
| `bolt` at level 6 | 14 | 2 | 28 ticks | 20 |
| `marathon` at level 5 | 13 | 8 | 104 ticks | 48 |

A press spends one round whatever the volley fires, so the capacity does not
depend on `Shots`. That was a bug for about ten minutes: tying it to the volley
width made a level-four form - where `Shots` goes up - come out with a *smaller*
magazine than it had at level three, which is a level that makes you weaker.

The reload is six shots' worth of cadence rather than a flat number, and that is
the second thing that was wrong here. At a flat 64 ticks a `bolt` - four ticks a
shot - spent two thirds of its life reloading while a `marathon` barely noticed.
Two thirds of the whole magazine was the next try and it wobbled: integer
division drifted the ratio a point either way between levels, which showed up as
a level that fires slower over a magazine than the level below it. Six times the
cadence has neither problem, and the guard is
`TestNoLevelEverMakesAFormWeaker`, which measures the SUSTAINED rate - rounds
per tick counting the reload - and not the reload on its own.

Firing on an empty magazine starts the reload rather than doing nothing, and it
does so ahead of the cadence check: whether the gun reloads itself must not
depend on exactly which tick you pressed on.

## Three upgrades, and a menu that stops the world

The score buys them, at 300 points and then further apart. The three are the
reference's own - **power**, **rate**, **magazine** - and they are the run's own
gun rather than the pet's: the kit that comes out of `KitFor` is the base, and
`boost` is applied on top.

The field freezes while the menu is up. A choice made with a bomb in the air is
not a choice, and the alternative - a menu over a running game - is a menu you
learn to answer with the same key every time to get back to the fight.

They are persisted **by kind** in `invaders.json` (`up_power`, `up_speed`,
`up_mag`) and replayed through `boost` on the way back in. That is not
over-engineering: the arena pauses and resumes a run every single turn, and
coming back after twenty waves with the pet's bare kit would read as the game
having forgotten what you built.

## Kits and rocks: two things that fall on their own

A **health kit** every minute, which is the reference's rate. It falls, you catch
it by flying into it, it goes in the hold - up to three - and `e` spends one for
a third of your maximum life. It is not spent on a full hull: a kit thrown away
for nothing is a kit somebody swears at.

An **asteroid** every twenty-two seconds, and it is on nobody's side. It does not
shoot and it does not aim; when it breaks it throws six meteoroids that hurt the
first thing they touch, which is as often one of theirs as it is you. Breaking
one scores nothing, deliberately: if it paid, the safest way to farm the game
would be to stand still and shoot rocks.

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

## Ten ships, and why the block went

The enemy has been four things. The order is worth keeping because each version
was a real answer to the version before it.

| | what it was | why it went |
| --- | --- | --- |
| 1 | the canvas's forty sprites, 5x3 | did not read as the genre at all |
| 2 | the arcade's pixel grids in half blocks, 12x4 | read perfectly, and one of them was three times the size of the player |
| 3 | one glyph an invader, 11x5 marching as a block | read perfectly and was the arcade - and a block is ONE decision on screen at a time |
| 4 | ten line-art ships, flying free | what the reference plays like, which is what was asked for |

Three is not a worse game than four. It is a different one, and the difference is
worth naming: a block moves as one thing, so there is exactly one thing to read
and the skill is in choosing an order to shoot it in. A fleet is ten kinds of
ship arriving in ones and twos, each with its own fall, its own drift and its own
gun, so what is in front of you is never the same twice.

```
 _^_      \_ _/      ^        ^^^      /\ /\     \__ __/    .-----.
 <o>      <ooo>     /o\      <-o->     (-o-)     -<ooo>-    |o-o-o|
                     v        v v      \/ \/     /  v  \    '--v--'
zángano   avispa    lanza    arpía    tejedora   cazador    yunque
```

Ten and not forty. Forty of anything is forty things nobody can balance and
forty rows of a table that all look alike after the fifteenth; ten is enough that
a wave is a mix and few enough that each one can be given a job. They come out
stage by stage - `Wave.Unlocked` - so wave one is drones and wasps and by stage
eight the whole zoo is out.

The bosses did not change. They are still the thirty-five off the design canvas,
5x9 block-drawing sprites with two leg frames, and they now carry a life bar in
the HUD: ninety hit points with no bar is a fight you cannot tell you are
winning, and the sprite's own colour ramp says the same thing far too slowly.

## A shot travels 1.1 rows a tick, and nothing it passes is one row tall

This was a bug the user found: *"the top row could not be killed until the block
dropped a step"*. A shot moves faster than a row per tick, so testing only the
row it landed on stepped clean over about one row in eleven - and the row it
stepped over most often was the top one, where the next step took it off the
field and it was thrown away untested.

The fix then was a sweep: check every row the shot crossed, lowest first, and
keep it past the top edge until the hits are resolved. The fix now is
structural - **every craft in the fleet is at least two rows tall**, so a step of
1.1 always lands inside one - and the guard moved with it, into
`TestEveryShipCanFlyAndBeKilled`, which refuses a one-row ship and says why.

The other half is still tested from the outside:
`TestAShotHitsTheShipItPassesThrough` fires at every one of the ten and fails if
any of them can be shot through.

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

Latching costs precision, so there is a brake, and a wall is a stop - leaving the
ship latched against one would mean the next key you press is a key you did not
know you had to press. And draining the key channel prefers an action over a
direction, for the same reason: a dropped direction costs nothing because the
latch carries it, while a dropped shot is a press the game ignored.

### Two axes, and where the brake went

The ship moves up and down as well now, and each axis latches on its own: press
left, press up, and it goes up and left until you say otherwise. That is the only
way a diagonal exists at all down a pipe that reports one key at a time.

The brake used to be the down arrow - the one arrow a ship on the floor had no
use for. It cannot be that any more, so it is `s`, which is the single key where
the two sets people already have in their fingers disagree: wasd wants it for
down, hjkl has nothing there. Down is the arrow and `j`, up is the arrow and `w`
and `k`, and `s` stops both axes at once.

Climbing is three times slower than strafing. A terminal cell is about twice as
tall as it is wide, so a row a tick reads as roughly double the speed of a column
a tick, and at the same cadence the ship crossed its half of the field before you
could let go of the key.

**Half the field, not all of it.** `Field.ShipRoof` is `Rows/2`. The reference
gives its player the whole screen and can afford to, because its enemies come
from everywhere; ours all spawn on row zero and come down, so a ship that could
reach the top would sit on the spawn line and shoot each one before it had drawn a
frame. That is not a harder game or an easier one - it is a game with no descent
in it. Half is nine rows in an eighty-by-twenty-four terminal: room to climb over
a bomb, to meet something before it reaches the floor, or to back away from a
boss.

Three things had to follow the ship up: the hitbox, the muzzle and the ram. The
first two are the same expression with `g.Row` in place of the floor. The third is
new - flying into one of theirs now costs the two life a landing on your head used
to, wherever on the field you did it - and it is what stops climbing from being a
way of taking the fleet out of play. Turrets keep the row they were dropped at,
because a turret that followed the ship would not be a turret.

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

Colour is emitted per RUN and not per cell. A sky of forty stars is one colour and
forty glyphs; wrapping each of them cost about ten kilobytes a frame and two
hundred a second, which is fine on a local terminal and is not fine down an ssh
connection.

The alternate screen and the cursor come back on every path, including a panic
and a signal. A game that leaves your terminal with no cursor is worse than one
that crashes.

Every sprite is painted cell by cell now, the ship included, and that is a
simplification the representation bought. While the cannon was `pet.DrawCard` the
grid needed a second channel - whole painted rows the assembly had to step over -
because `internal/pet` hands back a finished row rather than cells, and
re-implementing its painter here would have been a second copy of the one thing
that package is for. The ship is five cells of line art now, so the channel is
gone and so is the class of bug where something drawn afterwards landed inside
those nine columns.

Blanks in a sprite are transparent, and one is not: the hull's middle row is
`<o o>`, and with a transparent gap a star sailed between the eyes and read as a
hole in the ship. `fill` paints the blanks, `blit` skips them, and the crest and
the tail still let the sky through the way they should.

One bug here is worth recording because three width tests all passed over it.
`theme.Truncate` counts the bytes of an escape sequence as visible width and cuts
wherever it lands, so the Spanish HUD in a 76-column terminal printed a literal
`[38` where the life bar should have been - three cells wide, so every test
asserting "no wider than the terminal" was satisfied. Looking at an actual frame
found it in a second. The HUD drops whole parts now, most important first, and
never cuts; plain text is truncated before the escapes go on. `theme.Truncate` is
left alone, because it is right for the plain text the statusline hands it.
