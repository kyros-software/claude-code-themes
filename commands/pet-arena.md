---
description: Let the game open itself while Claude works, or stop it opening (on | off | status)
argument-hint: "[on|off|status]"
allowed-tools: ["Bash(~/.claude/ccpet/bin/ccpet arena:*)"]
---
Run `~/.claude/ccpet/bin/ccpet arena $ARGUMENTS` with Bash and show its output as
it comes. With no argument it only says whether the arena is on; it never turns
it on by itself, because what it switches is a program opening windows on
somebody's desktop.

`on` makes every prompt open the game in a window of its own (or bring forward the
one already playing) and every finished answer pause it and hand the focus back.
`off` stops that: nothing opens any more. A game that is already open is left as
it is - quitting it is the player's call, with `q`.

If `on` also prints a line saying there is no terminal emulator to open the game
in, pass that on as it is: the arena is switched on, and it will do nothing until
there is one.

If the binary is missing, the runtime link has not been made yet: it is a
SessionStart hook, so opening a new session is enough. Say that rather than
guessing at paths.
