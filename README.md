# claude-code-themes

Three colour themes for [Claude Code](https://claude.com/claude-code) and a
**statusline** that takes the foot of the window: four bands of data on the left
and a pet on the right that reflects the state of the session and **evolves with
how you work** — 41 forms in a six-level tree, with xp, hunger and a streak.

Installable as a plugin. The runtime is a Go binary with no dependencies: no
`python3`, no `node`, no `jq`. It costs 1.5 ms a refresh.

```
──────────────────────────────────────────────────────────────────────────────────────────────────────────────
 Opus 5  ██████░░░░░░░░░░ 36% · 1M ctx │ xhigh │ 5h 41%  7d 13% │ 98% cache                           ▚╲   ╱▞
claude-code-themes (main) │ +184/−37 │ $28.29 │ 1h 12m                                                ▗▟███▙▖
explanatory                                                                                          ▐█ > < █▌
bughunter level 4 │ lively                                                                            ▖▖▀▀▀▗▗
```

## Installing

```
/plugin marketplace add kyros-software/claude-code-themes
/plugin install claude-code-themes
/pet-statusline
```

The first brings the themes, the commands and the hooks that feed the pet. The
third turns the statusline on — it takes a step of its own because `statusLine`
is not a plugin component and the key goes into `~/.claude/settings.json`, with a
backup and an atomic write. After that, `/theme` → Terminal.

Without the plugin:

```bash
scripts/install.sh            # themes + statusline + pet + /pet and /feed
scripts/install.sh --hooks    # and wires up the hooks that feed it
scripts/install.sh --uninstall
```

The **hooks are a separate step on purpose**: they live in the global
`settings.json`, so they run in every one of your repos. Without them the pet
exists and shows up, but it only eats through `/feed`.

## The four bands

- **1 · the engine** — model, context, the two limits, effort and pace: what
  changes every turn. The quotas are painted with the same colour ladder as the
  pet, so a `5h` at 95% comes out indigo even with an empty window.
- **2 · the work** — repo, branch, diff, cost and the session's clock.
- **3 · where, and with what judgement** — the directory (just the directory; at
  the root of the repo it disappears) and the active output style, **checked
  against the disk** before it is painted: the payload sends the configured name,
  not the loaded one.
- **4 · the pet** — trade, the mark in brackets, level, how it is doing, the bar
  and the speech bubble. `bughunter[bloodhound]` reads whole, as one name: *a
  bughunter, in its bloodhound form*.

Below **100 columns** band 4 keeps the trade and nothing else; below **55**, the
pet disappears.

The why behind every decision is in [statusline.md](docs/design/statusline.md).

## The pet

Nine columns. The silhouette **and the colour** are chosen by the evolution; the
eyes, the feet and the step of the ramp are chosen by the state. Every branch has
its own hue and keeps it across the seven states, which is what makes 41
silhouettes tellable apart in the rows there are.

### Seven states

One number decides state, eyes, feet and colour:
`context_window.used_percentage`. It moves **four independent signals**, in this
order — first the eyes, then the step, then the head sinks, and at the end the
silhouette lies down. At a glance you can tell *tired* from *drowning* without
reading the label.

| Usage | Label | Eyes | Head | Feet |
| --- | --- | --- | --- | --- |
| ≤22% | fresh ✦ | `>` `<` | up | walking |
| ≤45% | lively | `>` `<` | up | walking |
| ≤63% | easy | `o` `o` | up | walking |
| ≤78% | sluggish | `▬` `▬` | up | still |
| ≤89% | tired | `_` `_` | **sunk** | still |
| <100% | drowning | `x` `x` | sunk | still |
| 100% | k.o. | `x` `x` | sunk | **lying down** |

It is **the context only**, on purpose. The 5h and 7d quotas belong to the
account, not to the session: with them in the sum, every open window read the
same number and the pet stopped talking about the session it lives in. They are
still in band 1, with their number and their colour, but with no face. The whole
reasoning is in [vitals.md](docs/design/vitals.md).

### The tree

You do not pick the form: it comes out of how you work. Commits and `/compact`s
lead down the **methodical** branch, tests and plans down the **inquisitive**
one, and working with the context high down the **impulsive** one.

```
        level 1   2           3             5              6
        spark ─┬─ ember ───┬─ marathon ──┬─ ox ─────────── mammoth
               │           │             └─ mole ───────── worm
               │           ├─ feral ─────┬─ gremlin ────── devil
               │           │             └─ kraken ─────── leviathan
               │           └─ sprinter ──┬─ sniper ─────── falcon
               │                         └─ bolt ───────── storm
               ├─ pattern ─┬─ tidy ──────┬─ gardener ───── forest
               │           │             └─ monk ───────── abbot
               │           └─ refactor ──┬─ surgeon ────── scalpel
               │                         └─ weaver ─────── loom
               └─ probe ───┬─ architect ─┬─ cartographer ─ atlas
                           │             └─ oracle ─────── sphinx
                           └─ bughunter ─┬─ exterminator ─ wasp
                                         └─ bloodhound ─── wolf
```

1 root + 3 temperaments + 7 trades + 14 marks + 14 titles = **39**, plus two
secrets off the tree: `phoenix` and `chimera`. Level 4 does not fork.

Every row of the design canvas is one form in the seven states. **The mark on top
and the number of feet identify the form and never change**; the state fills in
the eyes, moves the step, flattens the body from *tired* on, lays it down at
*k.o.* without losing the count of feet, and walks the colour down that branch's
ramp.

![Level 1: spark](assets/formas-nivel-1.png)
![Level 2: the three temperaments](assets/formas-nivel-2.png)
![Level 3: the seven trades](assets/formas-nivel-3a.png)
![Level 3: the seven trades, continued](assets/formas-nivel-3b.png)

The marks and the titles inherit their trade's ramp — a bloodhound is blue like
the bughunter it comes from, and what tells them apart is the body — and that is
why **ten ramps are enough for 41 forms**. They all come out of
`internal/pet/testdata/ATLAS.json`, which is in the repo and which four tests
compare the Go against: the 41 names, the 10 ramps, the tree's parents and the
287 silhouettes row by row.

### How it eats

| Event | xp | Hunger | Brake |
| --- | --- | --- | --- |
| green suite | **+15** | −4 | once an hour, and only if you changed something |
| commit | **+12** | −3 | — |
| compact | **+8** | −3 | — |
| a plan's task closed | **+6** | −1 | — |
| `/feed` | **+3** | −2 | one every four hours |
| context at 100% | **−15** | — | breaks the streak |

The levels land at 60, 180, 400, 2000 and 4500 xp.

**And it goes down.** Hunger climbs +1 an hour without food, capped at 10; past
that every hour costs 1 xp. The xp has a ceiling — the last threshold plus one
level-1 stretch, `4500 + 60` — because without it the buffer you have built up
swallows any penalty. Hence the two figures: 60 hours — **two and a half days** —
to lose the level above, and 4560 hours, **about six months**, to go back to a
larva. It never dies: at the bottom it stays a `spark`, which is a form, not a
grave.

**A form does not fall.** It moves sideways, upwards or across branches: an
`exterminator` becomes a `bloodhound` or a `wasp`, but never a bare `bughunter`
again. It goes down a rung in one case only — you change branch and you already
have a mark earned there — because that is not falling, that is having moved. The
level can go down even when the form does not, so `wasp level 5` is legitimate.

The whole tree, and what feeds each counter, is in
[evolution.md](docs/design/evolution.md).

### `/pet`

Shows the **22 counters that decide the tree**, not just the four the statusline
carries. The colour means one thing only: whether that counter is taking you
somewhere you can still get to.

```
  bughunter   level 4
  inquisitive › probe › bughunter

  level  █░░░░░░░░░░░░░░░  533/2000 xp
  hunger ██░░░░░░░░  2
  streak ███░░░░  3 days · best 3

  the level 5 mark
    ✓ bloodhound     reproduced before fixing   36/10
      exterminator   days running in the green  2/15
```

### `ccpet invade`

A shmup where the ship is **the creature you have right now**. Its trade decides
the weapon, its mark refines it, its level scales it - so feeding the pet is how
you get a better gun, and the forty-one forms all play differently.

```
oleada 12 · ♥ ██████░░ · puntos 8400 · cazabugs n4 rastreador · habilidad ████░░

 ▚╲   ╱▞                       ▪            ▰▰
 ▗▟███▙▖      ·        »              ·
▐█ > < █▌  ·         ·           ◆          ▬▬▬
 ▝▀▀▀▀▀▘      ·                             ▪
 ▝▝   ▘▘                  ∘

↑↓ mover · espacio habilidad · p pausa · q salir
```

The gun is automatic and the **ability** is what you time - terminal key repeat
is too uneven across emulators for holding a key to feel like anything. Every
fifth wave a **rival** turns up: one of the forty-one forms you are not, drawn
with its own sprite and firing the kit that form would fly. There is no last
wave; a run ends when the swarm outgrows your kit, and the record is how far you
got.

It needs its own terminal - at least 60x18 - because a statusline refreshes once
a second and cannot read a keypress. Inside tmux, `ccpet invade --split` opens it
in a pane beside Claude.

**Losing costs the creature a level.** Not the shape: you stay whatever you
evolved into, but the kit drops a step until you feed it back up. It never costs
more than a day's feeding, and quitting is not losing.

## The themes

| Theme | Accent | Look |
| --- | --- | --- |
| **Terminal** | `#4dd6c1` turquoise | one colour per kind of data |
| **Blood Red** | `#ff5c47` coral | warm: coral, terracotta, wine |
| **Electric Blue** | `#2e8bff` blue | cold: cyan, azure, deep blue |

![Electric Blue on the left and Blood Red on the right](assets/preview.png)

*The statusline showing in the corner of that screenshot is an old one; the CLI's
colours, which are what it is there to show, are still these.*

**Terminal** is the one that pairs with the statusline: a kind of data always
carries the same colour, so you do not have to read to know what you are looking
at.

| Role | Hex | |
| --- | --- | --- |
| Paths, files, repos | `#4DD6C1` | turquoise |
| Identifiers, code, additions | `#57E389` | green |
| Urls, branches, links | `#6FB6FF` | light blue |
| Numbers, money, warnings | `#E8C46A` | amber |
| CLI modes and settings | `#B07CF0` | violet |
| Deletions, errors, risk | `#F2777A` | salmon |
| Emphasis in prose | `#ECEFF4` | near white |
| Separators, units | `#6B7683` | grey |

All three cover the **72 tokens** Claude Code knows about, not just the dozen you
see at a glance.

## Language

The theme speaks **Spanish or English**: the pet, the panel, the statusline, the
install messages and the help. Spanish by default, which is what it spoke before
it spoke two languages — upgrading does not reword the foot of your window.

```bash
ccpet lang            # says which it speaks and who decided that
ccpet lang en         # English, from now on
ccpet lang es         # Spanish
ccpet lang auto       # whatever your locale says (LC_ALL, LC_MESSAGES, LANG)
```

It is kept in `~/.claude/ccpet.json`, beside `pet.json`, and it honours
`CLAUDE_CONFIG_DIR` like everything else. For one command, without touching the
setting:

```bash
ccpet --lang en                 # the panel in English
CCPET_LANG=en ccpet             # the same, through the environment
```

The **ids never change**: `pet.json` holds `bughunter`, `bloodhound` and `fresh`
in both languages, so switching does not touch the pet's life or cost you a
streak. All that changes is what you read — and in English the tree's names *are*
the ids, because the tree was written in English already:
`bughunter[bloodhound]` reads in Spanish as `cazabugs[sabueso]`.

The three colour themes are the same in both languages: colour does not talk.

## Settings

| Variable | Effect |
| --- | --- |
| `STATUSLINE_PET=0` | turns the pet off, leaves the four bands |
| `STATUSLINE_PET_WALK=1` | walks on every refresh instead of now and then |
| `STATUSLINE_BACKGROUND=0` | drops the footer's background |
| `STATUSLINE_RULE=0` | drops the rule on top and saves a row |
| `STATUSLINE_RIGHT_PAD` | right margin, `6` by default |
| `PET_TEST_RUNNERS` | extra regex to recognise your test runner |
| `CLAUDE_CONFIG_DIR` | moves `~/.claude`; the pet and the statusline honour it |
| `CCPET_LANG` | `es`, `en` or `auto` for a while; wins over the saved setting |

**Truecolor.** The themes use 24-bit colour, and Windows Terminal, WSL and
`docker run` do not export `COLORTERM`. Without it, close shades collapse into
one:

```bash
export COLORTERM=truecolor                                  # .zshrc / .bashrc
docker run -e COLORTERM=truecolor -e TERM=xterm-256color ...
```

The pet does have a plan B: it quantises to the real 256 cube, so it looks the
same with fewer shades.

## Migrating from the Python version

**There is nothing to do.** `scripts/install.sh` deletes the old launchers and
`pet.json` is translated by itself the first time it is written. The pet keeps
its xp, streak, counters and secret form. What you read on screen is in whichever
language you have set; the file holds the ids in English, because renaming them
would rewrite every life file out there.

The one thing by hand is the environment variables: the ones that were in Spanish
are no longer read, and are now `STATUSLINE_PET`, `STATUSLINE_PET_WALK`,
`STATUSLINE_BACKGROUND` and `STATUSLINE_RULE`.

## Deeper in

- [statusline.md](docs/design/statusline.md) — the four bands: why each piece of
  data is where it is, and what is verified before it is painted
- [vitals.md](docs/design/vitals.md) — the layer of the moment: from fresh to k.o.
- [evolution.md](docs/design/evolution.md) — the permanent layer: xp, food and the
  41 forms
- [language.md](docs/design/language.md) — Spanish or English: what is translated,
  what is not, and why `pet.json` is the same file in both
- [runtime.md](docs/design/runtime.md) — why Go, where the time goes, the
  `pet.json` lock and why the binaries are in the repo
- [invaders.md](docs/design/invaders.md) — `ccpet invade`: why the ship is the
  creature you already have, one kit per form, thirty-five enemies composed from
  seven bodies and five traits, and the one rule the game breaks
- [audit-log.md](docs/audit-log.md) — history: the audit of the Python version
- [thresholds.md](docs/design/thresholds.md) — **unimplemented**: the design
  canvas's 97-form tree, why its rule puts 27 of the 42 marks out of reach, and
  the fix that gives them back. The gates live in `testdata/PUERTAS-97.json` and
  `go test ./internal/pet/ -run NinetySeven` checks them

## Licence

[MIT](LICENSE).
