#!/bin/bash
# Installer for the Terminal theme and the pet, without a plugin.
#
#   scripts/install.sh              themes + statusline + /pet
#   scripts/install.sh --hooks      the above, and the feeding hooks
#   scripts/install.sh --uninstall  undoes the hooks and the statusline
#
# IF YOU CAN RUN /plugin, PREFER THAT:
#
#   /plugin marketplace add kyros-software/claude-code-themes
#   /plugin install claude-code-themes
#   /pet-statusline
#
# The plugin carries the themes, the commands and the hooks natively, and it
# updates itself. This script is for anyone who would rather not have a plugin,
# or who wants the runtime pinned to a checkout they control.
#
# The hooks are separate on purpose: they live in ~/.claude/settings.json, which
# is global, so they run in ALL your repos. Without them the pet exists and
# shows up, but it does not feed itself: you feed it with /feed.
set -euo pipefail

SOURCE=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
CLAUDE="${CLAUDE_CONFIG_DIR:-${CLAUDE_DIR:-$HOME/.claude}}"
TARGET="$CLAUDE/ccpet"
HOOKS=0
REMOVE=0

for arg in "$@"; do
  case "$arg" in
    --hooks) HOOKS=1 ;;
    --uninstall) REMOVE=1 ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) echo "unknown option: $arg" >&2; exit 2 ;;
  esac
done

# Files the pre-package layout dropped straight into ~/.claude. Removed on both
# paths so an upgrade does not leave a second, stale copy behind.
#
# The Python runtime's drawing module used to be listed here too and no longer
# is. Anyone upgrading from a 1.x install keeps one inert .py file in ~/.claude
# that nothing loads; naming it here was the only way to delete it.
LEGACY=("$CLAUDE/statusline.sh" "$CLAUDE/pet" "$CLAUDE/pet-hook.sh")

# The binary does the settings.json surgery: it is already here, and one atomic
# writer beats two.
CCPET="$SOURCE/bin/ccpet"
[ -x "$CCPET" ] || { echo "bin/ccpet is missing; run scripts/build.sh" >&2; exit 1; }

# ---------------------------------------------------------------------------
if [ "$REMOVE" = 1 ]; then
  echo "Uninstalling..."
  CLAUDE_CONFIG_DIR="$CLAUDE" "$CCPET" setup uninstall \
    || echo "  settings.json unreadable: carrying on and removing the files"
  rm -rf "$TARGET"
  rm -f "${LEGACY[@]}" "$CLAUDE/commands/pet.md" "$CLAUDE/commands/feed.md" \
        "$CLAUDE/commands/pet-statusline.md"
  rm -rf "$CLAUDE/__pycache__"
  echo "Done. ~/.claude/pet.json stays: delete it yourself to start from scratch."
  exit 0
fi

# ---------------------------------------------------------------------------
echo "Installing into $TARGET"
mkdir -p "$CLAUDE/themes" "$CLAUDE/commands" "$TARGET"

for theme in "$SOURCE"/themes/*.json; do
  cp "$theme" "$CLAUDE/themes/"
  echo "  theme: $(basename "$theme")"
done

# A real directory here, not a symlink: that is exactly how the plugin's
# SessionStart hook knows this copy owns the path and steps aside.
#
# The whole thing goes, not just the parts we are about to write: upgrading
# from the Python era leaves a src/ tree that nothing would ever clean up, and
# a stale module is a module that still gets imported. Nothing of the user's
# lives in here - pet.json is one level up, in ~/.claude.
rm -rf "$TARGET"
mkdir -p "$TARGET"
cp -R "$SOURCE/bin" "$TARGET/bin"
cp -R "$SOURCE/scripts" "$TARGET/scripts"
chmod +x "$TARGET"/bin/* "$TARGET"/scripts/*.sh 2>/dev/null || true
rm -f "${LEGACY[@]}"
rm -rf "$CLAUDE/__pycache__"
echo "  ccpet/bin, ccpet/scripts"

# Copied, not written inline: the plugin ships the same three files, and one
# wording in two places is one wording that drifts.
for command in "$SOURCE"/commands/*.md; do
  cp "$command" "$CLAUDE/commands/"
done
echo "  commands: /pet, /feed, /pet-statusline"

if [ "$HOOKS" = 1 ]; then
  CLAUDE_CONFIG_DIR="$CLAUDE" "$CCPET" setup install-hooks "$TARGET"
else
  CLAUDE_CONFIG_DIR="$CLAUDE" "$CCPET" setup install "$TARGET"
fi

echo
echo "Done. Now pick the theme: /theme -> Terminal"
echo "Open a NEW terminal so COLORTERM reaches it; without that you see it quantised."
