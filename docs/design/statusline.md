# The statusline, band by band

Why each piece of data is where it is, and what is checked before it is painted.
The [README](../../README.md) says what each band carries; this says why.

```
──────────────────────────────────────────────────────────────────────────────────────────────────────────────
 Opus 5  ██████░░░░░░░░░░ 36% · 1M ctx │ xhigh │ 5h 41%  7d 13% │ 98% cache                           ▚╲   ╱▞
claude-code-themes (main) │ +184/−37 │ $28.29 │ 1h 12m                                                ▗▟███▙▖
explanatory                                                                                          ▐█ > < █▌
bughunter level 4 │ lively                                                                            ▖▖▀▀▀▗▗
```

It is a **footer**, not one more line of the thread: a background one shade above
black and a thin rule on top. Five rows — the rule and four bands — with the pet
anchored to the right across all four. Each band groups data that is looked at
together, and drops its lowest-priority elements rather than wrapping, which
knocks the prompt box out of square.

## Band 1 · the engine

The quotas go as a bare number, no bar, and **painted with the same ladder** as
the context bar and the pet: a `5h` at 95% comes out in *drowning*'s indigo, so
the thing about to stop you is the strongest colour on the line even with the pet
in green. It is the answer to "why is it drowning when the window is empty?".

**The `tok/s` is real, not an estimate**, and where it comes from is worth
looking at. The payload's two fields do not measure the same thing:
`total_output_tokens` is what the *last* response produced — it resets every turn,
it is not a counter that climbs — while `total_api_duration_ms` is the session's
*accumulated* API time. The last response's pace is the first over however much
the second has grown. Subtracting two consecutive `total_output_tokens` measures
nothing: they are two different responses, and the result comes out inflated or
negative depending on which was longer.

It goes out by itself after two minutes without moving, and then the cache hit
takes that slot. The two never show at once.

The bar measures the same number that decides the pet, so **bar and pet cannot
contradict each other**. The other two arrangements were tried and read as a bug:
with the bar measuring the context and borrowing the neck's colour, a session at
48% with the 5h quota at 67 drew a half-mast bar next to the word `sluggish`,
which is the reading for 67. Promoting the bar to the neck to close that gap, the
band printed `82% 5h` three columns before printing `5h 82%` again.

## Band 2 · the work

The repo's name comes from the payload's `workspace.repo.name` when there is a
remote, and otherwise from the root directory. The name only: the owner is always
the same and tells you nothing about where you are.

## Band 3 · where, and with what judgement

The directory and the active output style: what barely moves.

Out of the path comes only **the directory you are in**, and if it is named the
same as the repo — that is, you are at its root — it disappears, because band 2
already says so.

The two go bare, with no label, and what tells them apart is the colour: the
directory in grey because it is a place, the style in `Mode`'s purple because it
is a CLI setting. They read in the order *where → who*, and when the band runs
short the style falls first: the band was the directory's to begin with.

### Why the style is lowercased

It is the footer's voice, not a fact about the style. Everything else that takes
that slot arrives lowercase already — `xhigh`, `plan`, `auto-edit`, the pet's
`bughunter` — so a capitalised name would be the one word on the line that shouts.

It happens in the band and not when the payload is read, for two reasons:
`Payload.Style` keeps the real name, and this way `Explanatory` and `Learning`
are covered too, which arrive capitalised and **cannot be renamed**. The directory
beside it is left alone: it has to match what `ls` says.

### Why the name is checked against the disk

The payload sends the **configured** name, not the loaded one. In the CLI they are
two steps and only the first arrives:

```js
let d = Tn()?.outputStyle || "default"
return e[d] ?? null              // e = the styles that loaded
...
output_style: { name: Xe }       // Xe = the config, raw
```

Which means a typo in `settings.json`, or a deleted file, is reported exactly like
a style that works **while the system prompt stays empty**. Painting that name
would be repeating the claim instead of verifying it.

So the band looks it up itself: the built-in styles resolve with no file, and the
rest have to show up in `~/.claude/output-styles/` or in the repo's
`.claude/output-styles/`, under the CLI's own naming rule — the frontmatter's
`name:`, and failing that the filename without `.md`, compared **case and all**,
because on the other side it is an object key. If it does not show up, it is not
painted.

Styles that come from a plugin are looked up **loosely**: any copy installed under
`plugins/cache/` will do, without working out which version is live — that is the
whole plugin loader, once a second. A false positive there only means painting a
name that exists somewhere; hiding a style that works would be worse. It costs
0.16 µs if it is built in, 5.9 µs on a hit in the user directory and 28 µs for the
full sweep.

What it does **not** catch: a style that resolves but is not loaded *in this
session* because the config changed after it started. There is no cheap trace that
tells that apart — `/output-style` rewrites that same setting and does apply hot,
so by timestamp the two situations are identical. Reopening fixes it, and the band
does not pretend to know.

With no style set the payload does not send a gap: it sends the word `"default"` —
`output_style: {name: outputStyle || "default"}`, read off the binary, not assumed
— and painting it would spend columns saying there is nothing.

### The band can come out empty

At the root of a repo with no style, which is most sessions. That row is anchored
with a **blank braille** (`U+2800`), because Claude Code trims leading spaces and
without it that row's piece of the pet falls to the edge.

## Band 4 · the pet

```
bughunter[bloodhound] level 5 │ fresh ✦ │ ████░░░░ │ ◗ five days on the trot
```

The bracket writes **the mark the pet is wearing**, with the trade it is a variant
of outside it: it reads whole, as one name, *a bughunter, in its bloodhound form*.
The tree forks at levels 2, 3 and 5, and the mark is the level-5 one, so the
bracket appears there and nowhere else: `bughunter` at level 4,
`bughunter[bloodhound]` at 5, and a plain `wolf` at 6, where the title is the end
of the branch and needs no context.

**It used to say the opposite.** It wrote the mark the pet was *heading for*, so a
level 4 read `bughunter[bloodhound]` without being a bloodhound. The idea was for
the bracket to be the tense — a name says *is*, a bracket says *is heading for* —
and that only works if it can be seen: it was painted in the separator bar's
colour, **1.54:1** against the background against the **11.8:1** of the two words
around it. Two bright words stuck together with nothing visible in between read as
one compound word, which is exactly what it was.

**The state lives here, not crowning the pet.** The canvas draws it twice, but on
a real terminal the same word ends up in the same footer within a few columns of
itself and reads as a bug. Moving it down to the band gave the pet back the row
its crest needs.

The bar measures **this level's stretch**, not the total xp, so it wakes up empty
the day after a level-up. At the top, where there is no ladder left, it changes
currency: it starts measuring the **habit** that opens the next mark, in amber and
with its name beside it. A pet already wearing its own has neither, and then the
band leans on the state.

## Widths

| Columns | What happens |
| --- | --- |
| < 100 (`BubbleMin`) | band 4 keeps **the trade and nothing else** |
| < 55 (`minWidthForPet`) | the pet disappears and the four bands remain |

**The right margin.** The statusline is not told the terminal's width — there is no
field for it in the JSON — so it comes from `COLUMNS`. And Claude Code cuts the
line some 5 columns earlier, so aligning against `COLUMNS-1` truncates the pet or
wraps it. Hence the default margin of 6 (`STATUSLINE_RIGHT_PAD`).

**Leading spaces.** Claude Code trims them. Rows whose left half is empty are just
"spaces + pet": trimmed, the pet falls to the edge and you end up with loose
pieces around the screen. Hence the blank braille.

## What is out of its hands

- The `bypass permissions` line and badges like `/rc active` are painted by Claude
  Code in its own footer. That is why the permission mode comes out as a **mark**
  and not as a word: on bypass, a red `⚡`, instead of spelling out again what is
  already written three lines above. `plan` and `auto-edit` keep their names: they
  have no obvious glyph and an invented one would be a riddle.
- **The animation's ceiling is 1 fps.** It re-runs on events (with a 300 ms
  debounce) and at rest only if you set `refreshInterval`, whose minimum is 1 s.
- The **welcome banner** uses onboarding accents that are not part of the theme
  system: it stays brand pink whatever theme is active.

The permission mode does not come in the payload, but it does come in the
transcript, whose path does arrive. Only the tail of the file is read (0.02 ms).
