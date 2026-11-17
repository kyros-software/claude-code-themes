#!/bin/bash
# Builds every platform into bin/. Needs Go and nothing else: the binaries are
# static (CGO_ENABLED=0), so there is no libc to match on the far end.
#
#   scripts/build.sh            all five targets
#   scripts/build.sh host       just this machine's, for a quick loop
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "$ROOT"
command -v go >/dev/null || { echo "go is required" >&2; exit 1; }

# The toolchain go.mod names, not whichever Go is installed. CI rebuilds bin/
# with that one (go-version-file) and compares bytes, so a build from go1.26.8
# against go 1.26.11 differs in every binary and goes red as "bin/ is stale".
# Go downloads the pinned toolchain the first time.
GO_PIN=$(sed -n 's/^go \([0-9][0-9.]*\)$/\1/p' go.mod)
export GOTOOLCHAIN="go${GO_PIN:?go.mod has no go line}"

VERSION=$(python3 -c 'import json;print(json.load(open(".claude-plugin/plugin.json"))["version"])' 2>/dev/null || echo dev)
# -trimpath keeps build paths out of the binary; -s -w drop the symbol table and
# DWARF, which is about a third of the size and nothing anyone needs at runtime.
#
# -buildvcs=false is what makes the build REPRODUCIBLE, and without it the
# binaries could not be checked against the source at all. Go stamps the commit,
# its timestamp and a vcs.modified flag into every binary - and bin/ is itself
# tracked, so writing the first build dirties the tree the second one reads:
#
#   clean tree  -> build -> vcs.modified=false baked in
#   bin/ written -> tree is now dirty
#   next build  -> vcs.modified=true -> a different binary, same source
#
# So two builds of identical source never matched, and the CI job that recompiles
# and compares would have failed for ever while reporting "bin/ is stale". None
# of that stamp is wanted here anyway: the version comes in through -X.
FLAGS=(-trimpath -buildvcs=false -ldflags "-s -w -X main.version=$VERSION")

TARGETS=(linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64)
if [ "${1:-}" = "host" ]; then
  TARGETS=("$(go env GOOS)/$(go env GOARCH)")
fi

mkdir -p bin
for target in "${TARGETS[@]}"; do
  os=${target%/*}; arch=${target#*/}
  out="bin/ccpet-$os-$arch"
  [ "$os" = windows ] && out="$out.exe"
  GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build "${FLAGS[@]}" -o "$out" ./cmd/ccpet
  printf '  %-28s %s\n' "$out" "$(du -h "$out" | cut -f1)"
done
echo "version $VERSION"
