# The runtime

Why this is a Go binary, where the time goes, and the two things that had to be
fixed for the numbers above to be true.

## Why Go

It was in Python and it worked. The problem was not the code — measured, it did
its job in 1.5 ms — but what it costs for Python to turn up: 5.4 ms of
interpreter plus 12.9 of imports, of which 10 were `subprocess` and `re` with
everything they drag along. That toll was paid **once a second** in the
statusline and **on every tool call** in the hook.

| | Python | Go |
| --- | --- | --- |
| statusline (once a second) | 22.4 ms | **1.5 ms** |
| hook, slow path (`Bash`, `Edit`, `TodoWrite`) | 21.3 ms | **1.7 ms** |
| hook, fast path (everything else) | 2.6 ms | **1.5 ms** |
| `/pet` panel | 14.7 ms | **1.4 ms** |

The hook is the one that matters: it hung 21 ms off every `Bash` and every
`Edit`.

> That table is the migration's measurement, both columns taken the same way.
> Reproducing it today is only half possible — the Python half no longer exists —
> and it is the order of magnitude that holds, not the decimal: 200 runs of
> `ccpet statusline` back to back on WSL2 give **1.7–1.9 ms of wall clock**, of
> which about **0.45 ms is the cost of starting any process at all** — a
> `/bin/true` measured in the same loop. WSL2 measures with noise: one round of
> nine came back with a negative delta. Measure it on your own machine:
>
> ```bash
> for i in $(seq 200); do COLUMNS=116 ./bin/ccpet-linux-amd64 statusline < payload.json >/dev/null; done
> ```

Two things that existed only to make Python cheaper went away with Go: the hook's
bash prefilter (starting the interpreter cost 15 ms, so it had to be avoided) and
the manual purge of `sys.path` — a `python3 -c` puts the current directory on the
import path, and any `json.py` in a repo you happened to have open hijacked the
statusline. Verified: it really did happen.

What is left is a twenty-line `bin/ccpet` in bash that picks the binary for the
platform, because `hooks.json` needs one fixed path. It uses `$OSTYPE` and
`$MACHTYPE`, which bash fills in itself: `uname` would be two forks in something
that runs on every call. And it is not even on the hot path — `ccpet link` leaves
two stable links to your machine's binary, and both the hook and the statusline
go straight there. The shim is the fallback, and it repairs the links when a
plugin update leaves them dangling.

## Half the statusline was `git`

The statusline read 3.5 ms in the first measurement and 1.5 now, and the
difference is not that Go runs faster:

| | |
| --- | --- |
| reading the branch out of `.git/HEAD` | **0.9 µs** |
| `git status` to know whether the tree is dirty (one fork) | **1.1 ms** |
| that same fact, already cached | **4.2 µs** |
| the rest of the refresh: parse, measure, compose the four bands | **3.4 µs** |

With `refreshInterval: 1` that was one `git` fork a second per open session to
redraw something that has almost never changed. So the two halves are split: **the
branch comes from reading `.git/HEAD`** — no fork, always exact, and it gets
detached HEADs, worktrees and a repo with no commits right along the way — and
only the "dirty tree" asterisk goes through `git`, **with three seconds of cache
per repo and session**.

That is the only thing in the footer that can lag: you make a commit and the ✳
takes up to one long refresh to go out. In exchange, two refreshes out of three
fork nothing.

## One `pet.json`, many windows

`~/.claude/pet.json` is **one file** for all your sessions and all your repos, and
the hook touches it on **every tool call**. Since Claude Code launches tools in
parallel, two writes at once is not the rare case: it is the normal one.

It was written with `rename`, which is atomic, and that solves a different problem
from the one there was: it guarantees nobody reads half a json, and it does not
stop two writers from stepping on each other. Since every write dumps the
**whole** state, the one that arrives late puts back everything it read and undoes
the other.

```
100 meals in series      800 xp
100 meals in parallel     72 xp    ← 91 % lost
```

Every modification now goes through a lock (`pet.json.lock`, beside it, empty):
the read and the write happen inside it, so the 100 meals leave all 800 points. It
is a kernel `flock` — `LockFileEx` through kernel32 on Windows, no dependencies —
and not a sentinel file, so a process that dies mid-write releases it by itself
and leaves nothing stuck.

If the lock is not taken within two seconds the write happens anyway without it:
losing an xp point now and then is a scratch, and a hung hook blocks the tool
behind it.

The same lock protects the session's tool log, which used to be read and cleared
in two steps and lost whatever arrived in between — **17,508 names out of 24,000
under load**. That fed `sniper`, which counts how many distinct tools you use
between two closed tasks, and made it see one-tool tasks that were not.

Two guard rails run over the code and fail if anybody goes back to the old
pattern.

## The binaries are in the repo, and CI checks them

The plugin is installed by cloning this repo, so the five builds live in `bin/`.
That only works if they are current, which is why CI recompiles and compares.

For that comparison to mean anything the build has to be **reproducible**:
`scripts/build.sh` passes `-trimpath -buildvcs=false`. Without the second, Go
stamps the commit and a `vcs.modified` flag with the state of the tree into every
binary — and since `bin/` is itself tracked:

```
clean tree      -> build -> vcs.modified=false baked into the binary
bin/ written    -> the tree is now dirty
next build      -> vcs.modified=true -> a different binary, the same source
```

Two builds of the same source never gave the same binary, and the CI job would
have failed for ever saying "bin/ is stale" with `bin/` perfectly current. None of
that stamp is wanted here: the version comes in through `-X`.

`.github/workflows/ci.yml` runs gofmt, `vet`, `go test -race` on ubuntu and macos,
the scripts' syntax and the json's validity. It uses `go-version-file: go.mod` and
not `stable`: a Go build is reproducible with the *same* toolchain, not with any
toolchain.
