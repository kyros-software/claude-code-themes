---
description: Turn the four-band statusline on or off (on | off | status)
argument-hint: "[on|off|status]"
allowed-tools: ["Bash(~/.claude/ccpet/bin/ccpet setup:*)"]
---
Run `~/.claude/ccpet/bin/ccpet setup $ARGUMENTS` with Bash - with no arguments it
turns the statusline on - and show its output as it comes.

The statusline is the one piece a plugin cannot install by itself: `statusLine`
is not a plugin component, so the key has to be written into
`~/.claude/settings.json`. The command backs the file up first and writes it
atomically.

If the binary is missing, the runtime link has not been made yet: it is a
SessionStart hook, so opening a new session is enough. Say that rather than
guessing at paths.
