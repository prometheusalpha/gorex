#!/usr/bin/env bash
# Builds GoRex, signs it with the local "GoRex Local" certificate and puts it
# in /Applications. A stable signature keeps the folder permissions macOS
# gave GoRex; an ad-hoc one changes every build and macOS asks again.
set -euo pipefail

cd "$(dirname "$0")/.."
app=build/darwin-arm64/GoRex.app

go tool mygo build
# Not `mygo build -sign`: it adds the hardened runtime, whose library
# validation refuses libghostty-vt.dylib, and no session opens.
codesign --force --deep --sign "GoRex Local" "$app"

if [[ "${1:-}" == "--no-install" ]]; then
	echo "Built $app"
	exit 0
fi

# pgrep misses GoRex while it runs, so look through ps.
if ps -axo command | grep -q "[/]Applications/GoRex.app"; then
	echo "GoRex is running: quit it (⌘Q) and run this again to install." >&2
	exit 1
fi
ditto "$app" /Applications/GoRex.app
echo "Installed /Applications/GoRex.app"
