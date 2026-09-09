# `ccpet invade` — plan

A shmup played with the pet you already have: your current form is the ship, its
trade decides the weapon, its mark refines it and its level scales it. Waves come
in from the right, a boss every fifth one, ninety-nine of them.

Status: **agreed, not written**. This file is the handover so the next session
starts at the code and not at the argument.

---

## 1 · Why this is a separate binary and not the footer

The first ask was to put the game in the grey gap of the statusline. It cannot go
there. Checked against the installed CLI (`2.1.266`), not from memory:

| the game needs | the statusline gives |
| --- | --- |
| 10–30 fps | **1 fps.** `statusLine.refreshInterval` is `min(1)` in the binary's own settings schema, integer seconds |
| keypresses | **none.** It is a command that receives the payload JSON on stdin, prints, and exits, once per refresh — no process is alive between frames and stdin is already spent |
| a key to swap between the prompt and the game | **no surface.** The ten hook events are `PreToolUse`, `PostToolUse`, `Notification`, `UserPromptSubmit`, `Stop`, `SubagentStop`, `PreCompact`, `SessionStart`, `SessionEnd`, `PermissionRequest`. None carries input, and nothing lets a plugin take the prompt line or paint over it |

Measured for the record: the gap in the footer is about **45 columns × 3 rows** at
116 columns, and it shrinks as bands 2–4 fill up. Even as a non-interactive
animation it is a lane defender, not a Space Invaders.

So: a real TUI, in its own terminal, sharing the pet. **The statusline is not
touched by this work.**

### What that costs, said plainly

"Easy to switch from the text input to the game with the keyboard" is the one
requirement that does not survive. The game owns a terminal; Claude Code owns
another. Switching is the terminal emulator's own tab/pane key. There is **no
tmux, screen or zellij on this machine** — checked — so either:

- a second terminal tab or window, and the emulator's existing shortcut; or
- install tmux, and add `ccpet invade --split` which runs
  `tmux split-window -h 'ccpet invade'` so one keystroke opens it beside Claude.

`--split` is worth building (it is ten lines, guarded on tmux being present), but
it must not become a dependency.

---

## 2 · Scope

In:

- `ccpet invade` — the game.
- Ship = the pet's current form, drawn with its real sprite and ramp.
- One distinct kit per form, scaled by level.
- Waves 1–99, a boss on every multiple of 5 and a final boss on 99.
- HP, enemies that get past the left edge cost life, life recovery.
- Auto-pause when Claude finishes a task, so you go and read what it did.
- Bilingual, like the rest of the theme since `language.md`.

Out:

- Any change to the statusline, the footer, or the pet's XP/counters. A run must
  not be able to feed or starve the pet — two systems, one file, no coupling.

---

## 3 · Layout

```
internal/invaders/
  save.go        the persisted run and the records
  kit.go         form + level -> weapon and ability
  wave.go        wave composition, enemy kinds, bosses
  game.go        the tick: movement, firing, collision, damage, waves
  render.go      grid -> painted lines, using internal/theme and pet sprites
  input.go       key decoding, terminal-independent
  term_unix.go   raw mode via termios ioctl (build tag !windows)
  term_windows.go  stub that refuses politely
  run.go         the loop: ticker, input channel, pause watcher, resize
cmd/ccpet/main.go   dispatch "invade"
internal/i18n/game.go   the game's own catalogue
```

`game.go` must stay free of terminal calls: the tick takes a state and an input
and returns a state. That is what makes waves, bosses and kits testable without a
tty, the same way `internal/pet` is testable without a statusline.

---

## 4 · The field

- Width from the terminal, height `rows - 2` (one HUD row on top, one help row at
  the bottom). Minimum **60 × 14**; below that, refuse with a message rather than
  draw a mess.
- The ship is the pet's 5×9 sprite at the left edge, moving up and down. Enemies
  are 1–3 cells wide, one row tall, entering at the right and marching left.
- Positions are `float64` on x so movement is smooth at 20 ticks/s; rows are ints.
- An enemy that crosses x < 0 is **not** killed: it costs HP and leaves.

## 5 · Controls

| key | |
| --- | --- |
| `↑` `↓`, `w` `s`, `k` `j` | move |
| space | the form's **ability** (the weapon fires by itself) |
| `p` | pause |
| `q`, `Ctrl-C` | quit — the run is saved at the wave you were on |

Auto-fire on purpose: terminal key repeat is unreliable and uneven across
emulators, so holding a key to shoot feels broken through no fault of ours. The
weapon is automatic and the **ability** is what the player times. It also puts
the form's identity in the player's hands, which is the point of the whole thing.

## 6 · The kits

One `Kit` struct, three layers: the **trade** gives the family, the **mark**
modifies it, the **level** scales it. Every one of the 41 forms therefore plays
differently without inventing 41 unrelated mechanics.

```go
type Kit struct {
    Family   string // for the name in the HUD
    Cadence  int    // ticks between volleys
    Damage   int
    Shots    int    // projectiles per volley
    Pierce   int    // enemies a shot passes through
    Homing   bool
    Splash   int    // rows splashed on a hit
    Special  string // what space does
    Cooldown int    // ticks
    MaxHP    int
    Regen    int    // hp per wave cleared
}
```

### Families, by trade

| form | family | what it is |
| --- | --- | --- |
| `spark` | single | one shot. The larva has nothing else |
| `pattern` | steady | slow cadence, higher damage |
| `probe` | seeker | weak homing |
| `ember` | rapid | fast cadence, low damage |
| `refactor` | twin | two shots, converging |
| `tidy` | sweep | ability: a beam that clears a whole row |
| `bughunter` | homing | shots track the nearest enemy |
| `architect` | turret | ability: drops a turret that fires on its own |
| `sprinter` | burst | very fast, very weak |
| `marathon` | cannon | slow, pierces everything in the row |
| `feral` | overload | damage scales with **missing** HP |

### Mark modifiers

Each of the fourteen marks adds one property to its trade's family; each title is
its mark one tier up — every number ×1.5 and the ability's cooldown halved.

| mark | adds | | mark | adds |
| --- | --- | --- | --- | --- |
| `surgeon` | +1 damage | | `bolt` | −30% cadence |
| `weaver` | +1 shot | | `sniper` | +2 pierce |
| `monk` | +regen | | `ox` | +50% max HP |
| `gardener` | +max HP and regen | | `mole` | ability: brief invulnerability |
| `bloodhound` | homing | | `gremlin` | random damage spikes |
| `exterminator` | splash 1 | | `kraken` | ability: strikes three rows at once |
| `cartographer` | turret ability | | | |
| `oracle` | −30% ability cooldown | | | |

Secrets: `phoenix` revives once per run at half HP; `chimera` carries both of its
parents' families at once, which is exactly what a chimera is.

### Level

Level 1–6 scales the numbers, so the same form gets stronger as the pet does:
damage `+level`, cadence shrinking, an extra projectile at 4 and at 6, max HP
`+2·level`.

**A test must assert all 41 forms resolve to distinct kits**, the same way
`TestEveryFormIsReachableFromAVeteran` guards the tree. A form that plays like
another is a form nobody has a reason to want.

## 7 · Waves and bosses

- 99 waves. Enemy count, speed and HP climb with the wave number.
- **Boss on every multiple of 5**, and 99 is the final one.
- A boss has HP, moves within the right third and **never crosses to the left
  edge**: it does not leak past you. It damages you by firing at the ship
  directly, so the fight is a duel rather than a leak race.
- Clearing a boss is the run's checkpoint.

## 8 · Life, and getting it back

The recovery rule, since it was left to me:

- Start at max HP for the form's kit.
- An enemy crossing the left edge costs **1 HP**; a boss's shot costs 2.
- **+`Regen` HP per wave cleared**, so a clean wave pays for a sloppy one.
- **A boss cleared heals to full.** That is what makes the every-fifth-wave rhythm
  a rhythm and not a slow bleed.
- At 0 HP the run ends: best wave and score go to the records, the run resets to
  wave 1. `phoenix` gets one revive first.

No XP, no hunger, no counters. **A run cannot touch `pet.json`.**

## 9 · Persistence

Its own file, `~/.claude/invaders.json`, through `config.Dir()` like everything
else. Not inside `pet.json`: that file has strict legacy-compatibility rules and
a game has no business near them.

```json
{ "wave": 12, "hp": 7, "score": 8400, "kills": 611,
  "best_wave": 20, "best_score": 15200, "runs": 4, "paused_at": 0 }
```

Resume granularity is the **start of a wave**. Quit mid-wave and you come back at
the top of it with the HP you had. Forgiving, and it makes the auto-pause below
cheap to implement correctly.

## 10 · The auto-pause

`Stop` fires when Claude finishes answering. Add it to `hookEvents` in
`internal/setup/setup.go` and to `hooks/hooks.json`, and give `hook.Run` a
`case "Stop":` that touches the pause file. The game polls its mtime every ~10
ticks and pauses with a banner saying why.

Two consequences to remember when writing it:

- `HooksWired` in **both** i18n catalogues lists the events by name and will be
  wrong the moment `Stop` is added.
- `Uninstall` and `dropOurHooks` already match on the `ccpet` marker, so removal
  keeps working with no change.

## 11 · Language

The theme speaks two languages as of `language.md`, so the game does too. Add
`internal/i18n/game.go` with its own struct and an `i18n.G()` accessor, and
extend `TestNoCatalogueIsHalfWritten` and `TestTheFormatsAgree` to cover it —
those two tests are what keep a second language from rotting.

Strings needed: HUD (wave, life, score, ability, cooldown), the pause banner and
its reason, game over, the records line, the too-small-terminal refusal, and one
name per ability family.

## 12 · Tests

- every form resolves to a distinct kit
- level scaling is monotonic: no level makes a form weaker
- waves 1–99 all generate, boss on every multiple of 5 and on 99
- a boss never crosses into the left third
- an enemy reaching the left edge costs exactly 1 HP and disappears
- HP never exceeds the kit's max, and a cleared boss heals to full
- `phoenix` revives once and only once
- a run cannot write to `pet.json` — assert the file's mtime is untouched
- the save round-trips, and a corrupt file is a new run rather than a crash
  (the same rule `pet.Load` follows)
- the tick is deterministic given a seed, so the whole thing is testable headless

## 13 · Docs

- `docs/design/invaders.md` — the design, in English like the rest.
- A section in the `README`, after `/pet`.
- No new slash command: a `/invade` that shells out could not be interactive, and
  a command that only prints "now go and type this" is worse than nothing.

---

## Open, for the morning

1. **Ship height.** The sprite is 5 rows in a ~20-row field. It may need to be
   `DrawCompact` (3 rows) to leave room to dodge. Decide by playing it.
2. **`--split`.** Build it, or leave tmux out entirely?
3. **Difficulty at 99.** Nobody is reaching wave 99 in a session. Is the run meant
   to last weeks, like the pet, or should the ladder be shorter and steeper?
4. Whether a run should show anywhere in the footer at all. Agreed as out of
   scope today; worth revisiting once it is playable, because a wave number in
   band 4 is cheap and it is the only part of the footer idea that survives at
   1 fps.
