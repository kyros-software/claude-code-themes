# The language

The theme spoke Spanish, and only Spanish. Not by accident: the names in the
tree, the pet's speech bubble and the panel's labels come out of the
«Tema Terminal Claude CLI» design canvas, which is written in Spanish, and half
of the pet's lines are jokes that do not survive a translation («el bug no era el
código, era el jueves» — the bug was not the code, it was thursday).

That was not the problem. The problem was that Spanish was **the only option**:
a public plugin, described in English, with a footer nobody outside this office
can read.

## What is translated and what is not

What a person **reads** is translated. Nothing a file **stores** is.

| | In Spanish | In English |
| --- | --- | --- |
| id in `pet.json` | `bughunter` | `bughunter` |
| what you read | `cazabugs` | `bughunter` |
| counter | `repro_before_fix` | `repro_before_fix` |
| what you read | `reproducir antes de arreglar` | `reproduced before fixing` |

That middle column is why switching languages costs nothing: `pet.json` is the
same file byte for byte, the hooks send the same events, and a nine-day streak is
still nine days. The language is a layer you read through, not a format.

It is also why there is **no English name table**. The ids are English words
already — the tree was written that way — so `Name(id)` in English returns the
id and there is nothing to keep in sync. The one table that is needed is the
counters', because their ids are keys and not words: `sessions_under_40` does not
read.

## Where each piece lives

- `internal/i18n` — which language is spoken and who decided that, plus the
  catalogue of everything that is neither a name nor a line of the pet's: the
  panel's labels, the `setup` messages, the help.
- `internal/pet/names.go` — the names of the tree and of the counters.
- `internal/pet/speech.go` — the pet's voice. `RepertoireEN` is **not a
  translation**: it is the same joke, told again in English. What is kept between
  the two is the *shape* — three lines per trade — because the do-not-repeat
  memory is three deep and the repertoire's restart counts on it.

The catalogue is a `struct` and not a map of keys on purpose: a mistyped field
does not compile, and a language added later cannot quietly forget half of it
without a test saying so.

## The setting

In order, the first one to answer wins:

1. `--lang xx` on the command line — for one command.
2. `CCPET_LANG` — for one terminal session.
3. `lang` in `~/.claude/ccpet.json` — `ccpet lang en` writes it.
4. The default, which is **Spanish**.

`auto` is stored as `auto` and resolved on every start against `LC_ALL`,
`LC_MESSAGES` and `LANG`: somebody who asks for the terminal to decide is asking
for every session, not for the language their locale happened to name that
afternoon. `C` and `POSIX` name no language at all, so they are skipped rather
than answered.

The default is Spanish and not the locale because upgrading must not rewrite a
footer somebody is used to. Anyone who wants the opposite has `ccpet lang auto`,
which is an instruction and not a surprise.

## What is still in Spanish

The pet's own voice, when the theme is set to Spanish — which is the point. And
the design canvas itself, quoted throughout these documents, because it is a
Spanish document and translating a quotation makes it stop being one.
