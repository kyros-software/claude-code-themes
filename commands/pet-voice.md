---
description: La voz de la mascota - frases escritas por Haiku según lo que pasa (on | off | status | say | shape on|off)
argument-hint: "[on|off|status|say|shape on|shape off]"
allowed-tools: ["Bash(~/.claude/ccpet/bin/ccpet voice:*)"]
---
Run `~/.claude/ccpet/bin/ccpet voice $ARGUMENTS` with Bash - with no arguments
it reports whether the voice is on - and show its output as it comes.

Each open session already wears the line of the tree its work calls for, read
off what the session has done by fixed rules: no model, nothing leaves the
machine. `shape off` / `shape on` switches that.

The voice is the optional part. With it on, Haiku - through the `claude`
already installed and logged in - writes three lines every half hour of work
about what the session is doing, which the pet says there one every ten
minutes, and settles the shape when the rules fit the session to two or three
lines of the tree and cannot choose. `status` lists what each session wears and
says; `say` asks for lines right now and prints them. It spends the person's
own quota - two or three questions per session an hour, some 4,000 tokens an
hour - which is why it is off until asked for.
