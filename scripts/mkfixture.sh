#!/bin/sh
# Builds a fake home full of typical junk for testing mr-plow safely.
# Usage: scripts/mkfixture.sh <dir>   then:  HOME=<dir>/home mr-plow
set -eu
D=${1:?usage: mkfixture.sh <dir>}
H="$D/home"
mk() { # mk <file> <MB> [touch-stamp]
  mkdir -p "$(dirname "$1")"
  dd if=/dev/zero of="$1" bs=1048576 count="$2" 2>/dev/null
  [ -n "${3:-}" ] && touch -t "$3" "$1"
  return 0
}
mkdir -p "$H"
# project artifacts
echo '{}' > /dev/null
mkdir -p "$H/Programming/web-app" "$H/Programming/rusty" "$H/Programming/pytool" "$H/Programming/old-2013-site"
echo '{"name":"web"}' > "$H/Programming/web-app/package.json"
mk "$H/Programming/web-app/node_modules/react/index.js" 30
mk "$H/Programming/web-app/node_modules/left-pad/node_modules/inner.js" 2
echo '{"name":"old"}' > "$H/Programming/old-2013-site/package.json"
mk "$H/Programming/old-2013-site/node_modules/grunt/grunt.js" 25 201304011200
touch -t 201304011200 "$H/Programming/old-2013-site/node_modules/grunt" "$H/Programming/old-2013-site/node_modules"
echo '[package]' > "$H/Programming/rusty/Cargo.toml"
mk "$H/Programming/rusty/target/debug/rusty" 20
echo '[project]' > "$H/Programming/pytool/pyproject.toml"
mk "$H/Programming/pytool/.venv/lib/site.so" 12
mk "$H/Programming/web-app/src/precious-source.js" 1
# caches
mk "$H/Library/Caches/com.spotify.client/Data/blob" 80
mk "$H/Library/Caches/com.tiny.app/blob" 1
mk "$H/.npm/_cacache/content/blob" 40
mk "$H/Library/Developer/Xcode/DerivedData/App-abcdef/Build/obj" 50
mk "$H/.cache/pip/http/blob" 60
# tool-managed node_modules: must NOT show up as a project artifact
mk "$H/.nvm/versions/node/v12.22.0/lib/node_modules/npm/npm.js" 25
# trash & downloads
mk "$H/.Trash/old-installer.dmg" 30
mk "$H/Downloads/ubuntu-14.04-desktop.iso" 150 201505051200
mk "$H/Downloads/Zoom.pkg" 60
# personal
mk "$H/Music/Mom/Greatest Hits/track01.mp3" 70 201102031200
mk "$H/Documents/thesis.pdf" 5
mk "$H/Movies/2016-vacation.mov" 90 201608081200
# hard link (counted once) and symlink pointing outside (never followed)
mk "$D/outside/precious-outside.bin" 50
ln -s "$D/outside" "$H/Programming/link-to-outside"
ln "$H/Downloads/Zoom.pkg" "$H/Programming/Zoom-hardlink.pkg"
# unreadable folder
mkdir -p "$H/locked/secret" && chmod 000 "$H/locked"
mkdir -p "$H/Library/Logs" && mk "$H/Library/Logs/app.log" 3
echo "fixture ready: HOME=$H"
