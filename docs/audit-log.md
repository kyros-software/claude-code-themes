> **Historical document.** This audits the **Python** implementation, which was
> the project's up to version 2.0.0. The runtime is a Go binary now and the
> numbers here no longer describe what runs on your machine: the statusline went
> from 22.4 ms to 3.5 and the hook from 21.3 to 1.6. It is kept because the
> measurements and the reasoning are still true about what they measured, and
> because they explain **why** the language was changed in the end: our own code
> cost 1.5 ms and the rest was Python turning up. The file and identifier names
> are the ones of the time (`statusline.sh`, `ESTADOS`), and so is the Spanish
> they are written in.

# Statusline audit

A pass over `statusline.sh`'s performance and bugs, measured on commit `e94ac98`
on 1 September 2026. Python 3.12.3, WSL2, git 2.43.

Everything here is **measured, not estimated**. What I could not reproduce is in
[Discarded](#discarded) so that nobody chases it again.

> **State as of 1 September 2026.** Bugs 1-4 are **fixed** and `python3 -S` is
> **applied**. Design points 5 and 6 are closed now too: see "The two design
> points" at the end. The measurements below are the ones from *before* anything
> was fixed: they are the baseline, and I leave them as they are so they can be
> compared. There is a section with the after figures at the end.

## How to reproduce the measurements

```bash
J='{"session_id":"bench","model":{"display_name":"Opus 5 (1M context)"},
    "workspace":{"current_dir":"'$PWD'"},"effort":{"level":"xhigh"},
    "cost":{"total_cost_usd":28.29,"total_lines_added":184,"total_lines_removed":37,
            "total_duration_ms":4320000},
    "context_window":{"used_percentage":36,"context_window_size":1000000},
    "prompt_cache":{"hit_ratio":0.98},
    "rate_limits":{"five_hour":{"used_percentage":41},"seven_day":{"used_percentage":13}}}'

# time per invocation
t0=$(date +%s%N); for i in $(seq 30); do echo "$J" | COLUMNS=116 ./statusline.sh >/dev/null; done
t1=$(date +%s%N); echo $(( (t1-t0)/30/1000000 )) ms

# breakdown of the imports
python3 -X importtime -c 'import sys,json,os,re,subprocess,time' 2>&1 | sort -t'|' -k2 -rn | head
```

---

## Performance

### Where the time goes

| Component | Cost | % |
| --- | --- | --- |
| **Total per invocation** (this repo) | **25.3 ms** | 100% |
| ↳ CPython startup (`python3 -c pass`) | 11.3 ms | 45% |
| ↳ ↳ *of which the `site` module* | *4.7 ms* | *19%* |
| ↳ imports: `json` 5.1 · `re` 4.0 · `subprocess` 3.8 (+`threading`, `selectors`) | ~6 ms | 24% |
| ↳ compiling the 10 KB of `PY_SRC` | 2.0 ms | 8% |
| ↳ the two `git` calls | 2.0 ms | 8% |
| ↳ `$(cat <<EOF)` + bash subshell | 0.8 ms | 3% |

In a big repo (`avanzapi-frontend`, ~840 files) it climbs to **30.7 ms**, because
`git status --porcelain` goes from 1.1 to 6.0 ms.

**The neck is not git: it is the interpreter starting.** 70% of the time goes
before the first useful line runs. Optimising the git calls is polishing 8% of the
problem.

### The real frequency

`settings.json` declares `refreshInterval: 1`, but the measurement does not give
1 Hz. Sampling the mtimes of `/tmp/claude-statusline-*` at 20 Hz for 15 s:

```
active sessions:  3 of 12
per session:      ~0.7 Hz
aggregate:        2.2 invocations/s  ->  ~6% of a core, continuously
```

Not a fire, but permanent for as long as the CLI is open, and it scales with the
number of sessions open at once.

### Prototyped optimisations

Both verified as producing output **byte for byte identical** to the original.

| Change | Result | Verdict |
| --- | --- | --- |
| `exec python3 -S -c` (skip `site`) | 25.3 → **23.3 ms** (−8%) | **Apply.** Zero risk: the script only uses the stdlib. |
| \+ move git to bash and drop `subprocess` | → **21.1 ms** (−17%) | **Do not apply.** See below. |

The second sounds good and is not worth it: bash does not have the `cwd`, which
comes in the JSON on stdin, and stdin is only read once. You would have to trust
`$PWD` assuming Claude Code invokes the statusline inside the workspace — an
undocumented assumption, in exchange for 2 ms.

**The real lever is not micro-optimising, it is `refreshInterval`.** The only thing
that demands a refresh every second is the feet's animation. At
`refreshInterval: 5` the cost falls fivefold.

---

## Bugs

| # | Severity | What happens | State |
| --- | --- | --- | --- |
| 1 | **High** | A non-numeric `used_percentage` leaves the statusline blank | **fixed** — every number from the JSON goes through `num()` |
| 2 | Medium | Rewrites the state file on every refresh even when it has not changed | **fixed** — only writes on a change |
| 3 | Low | Orphan files in `/tmp`, one per session, never cleaned | **fixed** — sweeps the ones over a day old, and `SessionEnd` deletes its own |
| 4 | Cosmetic | A file descriptor that is never closed | **fixed** — `finally: os.close(fd)` |

### 1 · Crash on a non-numeric `used_percentage`

**High**, because the failure does not degrade: the whole statusline disappears.

```
$ echo '{"context_window":{"used_percentage":"abc"}}' | ./statusline.sh
Traceback (most recent call last):
  File "<string>", line 194, in <module>
ValueError: cannot convert float NaN to integer
```

| Value | Result |
| --- | --- |
| `"NaN"`, `"abc"`, `[]` | **crash**, empty output, exit 1 |
| `"36"`, `true`, `-5`, `150` | ok |

The cause is that `float(pct)` and `round(float(pct))` in band 1 are **the only
point in the script with no `try/except`**. Every other numeric field — `cache`,
`coste`, `ctxsz`, `durms`, `rl5`, `rl7` — is guarded. It is an inconsistency, not a
decision.

Fix: normalise `pct` to `float` or `None` once, alongside the other reads, and stop
reconverting at each use.

### 2 · Redundant state write

The per-session file keeps the previous label so that one refresh can be bolded on
crossing a threshold. It is written **always**, changed or not:

```
attempt 1: mtime=10:53:55.511667035  content=8 bytes
attempt 2: mtime=10:53:55.538067034  content=8 bytes   <- same content
attempt 3: mtime=10:53:55.564467033  content=8 bytes   <- same content
```

One `write()` plus filesystem metadata per invocation, per session, to change
nothing. The block already reads the previous value: writing only when it differs is
enough.

### 3 · Rubbish in `/tmp`

One file per `session_id`, always created, never deleted. On the test machine:
**12 files, 9 of them from dead sessions**. They are 8 bytes each, so the problem is
not the space but that it grows with no ceiling and `/tmp` is not always cleaned on
reboot (WSL among them).

Fix: when writing, sweep the `claude-statusline-*` with an mtime over a day old.

### 4 · Unclosed descriptor

```python
return os.get_terminal_size(os.open("/dev/tty", os.O_RDONLY)).columns
```

`os.open` returns an fd nobody closes. Harmless — the process dies right after —
and it only runs on the fallback branch, when `COLUMNS` is missing. It is here for
hygiene, not for impact.

---

## Design

Not bugs: decisions worth taking knowingly.

### 5 · The k.o. is practically unreachable

With all three usages present, the `k.o.` state demands `usage > 99.999`, and being
a weighted average that means all three pinned at 100%:

| ctx | 5h | 7d | usage | state |
| --- | --- | --- | --- | --- |
| 100 | 100 | 100 | 100.000 | k.o. |
| 100 | 100 | 99 | 99.800 | drowning |
| 100 | 90 | 90 | 95.000 | drowning |
| 100 | — | — | 100.000 | k.o. |

It is consistent with what [vitals.md](design/vitals.md) documents, and it still
deserves saying plainly: **the k.o. sprite, the one that took the most work, is
almost never going to be seen.** It only appears if the context is the one piece of
data available. If it is to be reachable, the threshold has to come down (to 97,
say) or the k.o. has to fire off the maximum of the three rather than the average.

### 6 · The animation runs flat out by default

The code read a calm environment variable and, only if it was set, limited the step
to four refreshes out of twelve:

```python
anda = bool(E.get("anda")) and (paso % 12 < 4 if _calma else True)
```

Without that variable the feet alternate on **every** refresh, for ever, in
peripheral vision. Calm mode — walking 4 seconds out of every 12 — is the better
default; whoever wants the continuous dance can ask for it with a variable.

---

## Discarded

Checked, and **not** a problem. Written down so the work is not repeated.

- **Shell injection.** There is none. The JSON never goes through the shell and
  `git -C` is invoked with an argument list, no `shell=True`. Tested with
  `display_name: "'; rm -rf /"` → it is painted literally.
- **Double-width characters knocking the pet out of square.** There are none. Every
  non-ASCII glyph in the script is East Asian *Ambiguous*, which paint at one
  column; zero `W`, zero `F`, zero Nerd Font. `vis()` counts correctly.
- **`index.lock` contention from running `git status` in a loop.** It does not
  happen: `.git/index`'s mtime does not change after the `status`.
  `git --no-optional-locks status` would be prophylactic. Mind the syntax: the
  option goes **before** the subcommand, putting it after is an error.
- **`assemble()` is O(n²).** It is, and it does not matter: n ≤ 6.
- **Input robustness.** Invalid JSON, `{}`, a nonexistent `cwd`, `COLUMNS` from 20
  to 200: everything degrades cleanly, exit 0, no wrap and no overflow.

---

## Suggested order

1. Bug 1 — the only one that breaks something visible.
2. `python3 -S` — 8% for free.
3. Bugs 2 and 3 — hygiene, five minutes.
4. Design 6 — invert the animation's default.
5. Design 5 and bug 4 — whenever.

---

See also the [README](../README.md) for the bands and the palette, and
[vitals.md](design/vitals.md) for the pet's state formula.

---

## After fixing it

Same conditions, same repo, same test JSON.

| | before | after |
| --- | --- | --- |
| Time per invocation | 25.3 ms | **24.9 ms** |
| Non-numeric `used_percentage` | crash, empty output | degrades, exit 0 |
| State writes per refresh | 1 always | 0 unless changed |
| Orphans in `/tmp` | grow with no ceiling | swept at 24 h |

The time drops **only 0.4 ms** and that deserves an explanation: `python3 -S` takes
2.0 ms off, but the evolution system adds reading `~/.claude/pet.json` and importing
the module. The net sum is that **the whole tamagotchi came in for free**, not that
the fix did nothing.

Two design decisions came out of this audit:

- **The drawing lives in its own module, not embedded in the `.sh`.** An imported
  module uses the bytecode cache; a `python3 -c` recompiles its source on every
  refresh. That gives back the 2 ms of `compile()` the table above measured.
- **`tempfile` is imported inside `escribir_pet()`**, not at the top. It costs
  2.0 ms and the statusline reads that file on every refresh but **never writes
  it**: only the hooks and `/feed` write.

And one that did not change: `git --no-optional-locks` **is** applied, even though
the audit classified it as prophylactic. It is free and the scenario it avoids — two
sessions fighting over `index.lock` — is real even if I did not reproduce it.

---

## Second round: the evolutions review

A `/code-review` over the evolutions commit turned up **fifteen findings, all
fifteen real**. I reproduced the worst three before touching anything. All fixed.

### The serious one

**Code execution from any repo you open.** `python3 -c` puts the current directory
on `sys.path` as `""`, and the statusline runs with the cwd set to your project.
`sys.path.insert(0, SL_DIR)` pushed the cwd to position 1 rather than removing it,
so **if that module was missing from `~/.claude` — the degradation path the README
itself announces — the one from the open repo was imported**, running it once per
refresh, with the exception swallowed by the import's `try`. Reproduced:
`*** REPO CODE EXECUTED ***`, rc=0, no trace. The cwd is now purged from `sys.path`
before importing.

### The embarrassing one

**Bug 1 from the first round, reintroduced in two new files.** The `num()` that
armours the JSON from stdin was applied neither to `~/.claude/pet.json` nor to the
session file in `/tmp`. A `{"hambre":"mucha"}` left the statusline blank again. Both
files are editable by anyone and one lives in `/tmp`. Every field of both now goes
through a type normaliser.

### The other thirteen

| What | How it showed |
| --- | --- |
| `dict(PET_VACIO)` was a shallow copy | `contadores` aliased the module's dict: one `contar()` poisoned every later read in the process |
| `sesiones_ctx100` counted twice | the kraken was reached in 2 sessions instead of 3 |
| a missing `t0` = epoch 0 | 56-year sessions handing out `ox` |
| `_subio` filtered out by `leer_pet` | the level-up bubble was dead code |
| `/feed`'s daily cap over `hoy[-40:]` | it was skipped as soon as the log rotated |
| `git commit` unanchored | a `grep "git commit"` gave +12 xp |
| `\bok\b` with `re.I` | any output saying "ok" gave +15 xp |
| unvalidated `session_id` in an `open()` | path traversal outside `TMPDIR` |
| `claude-pet-todos-*` markers | a prefix the orphan sweep never reached |
| `json.load` with no `try` in the uninstaller | with `set -e`, a broken settings.json stopped you uninstalling |
| `settings.json` written non-atomically | a failure mid-write emptied your global configuration |
| `alimentar(ahora=…)` only half done | `dia` off the real clock and `ayer` off the parameter |
| `phoenix` and `chimera` unreachable | nobody wrote `secreta`: two templates were dead data |

The last two were fixed by **implementing them**, not by documenting them: the
phoenix asks for touching hunger 10 and coming back to 0 in the same session from
`feral` or `marathon`, and the chimera for two temperaments tied on reaching level
4. The 27 templates are now reachable.

### What it teaches

The four bugs of the first round were all **badly validated external input**.
Twelve of these fifteen too. The difference is that in the first round there was one
input — the JSON on stdin — and in this one there are four: stdin, `pet.json`, the
session file and the hook's JSON. **I armoured the one I already knew about and not
the three new ones.** The lesson is not "validate more": it is that every file you
add is a new trust boundary, and it is worth counting them.

---

## The two design points, closed

### 5 · The k.o. is reachable now

Demanding 100% of the **average** was demanding all three usages at 100% at once:
with ctx, 5h and 7d at 100, 90 and 90 the average gave 95, that is *drowning*. The
sprite that took the most work was never seen.

The k.o. now has **a door of its own**: it fires as soon as the context reaches
100%, without looking at the average. It is consistent with why the average weighs
50/30/20 — the context is the only thing that really stops you — and it touches no
other state.

> **Later note.** That door no longer exists, and this section explains why it was
> needed: it was the symptom, not the disease. The cause was the average, which
> cannot reach 100 unless all three usages do. Usage went back to being the
> **tightest neck** — what the first version measured — which reaches 100 on its
> own, so the door was surplus and went with it. See
> [design/vitals.md](design/vitals.md).
>
> **And a third.** The neck did not stay either. The 5h and 7d quotas belong to the
> **account**, not to the session, so every open window read the same number and the
> pet stopped describing its own. Usage is now the session's context and nothing
> else; the k.o. still needs no door, because the context reaches 100 on its own
> just as the neck did.

### 6 · Calm is the default

The calm variable went from being an optional patch to not existing (it was lost
when the drawing moved into its own module) and then to existing again. It is now
solved the other way round: **by default it walks four seconds out of every
twelve**, and the walk variable gives back the continuous dance the design asked
for. Perpetual motion at the corner of your eye at 1 fps is a permanent attention
cost in exchange for nothing.

## And a lesson that is not about code

While making these changes I discovered that **another Claude session was editing
this same repo at the same time** (`claude-code-themes-84`, redesigning the output
as a footer with a background and a rule). My patches and theirs applied over the
same working tree without colliding out of pure luck: I used string replacement
with anchors that still existed.

That it worked does not make it right. What to do before editing a file in a shared
repo is to look at `git status` **and** at whether there are other live sessions,
not to find out halfway through because the output did not add up.
