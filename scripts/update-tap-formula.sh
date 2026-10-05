#!/usr/bin/env bash
# Point the tap's spx formula at a release.
# Usage: update-tap-formula.sh <tag> <checksums.txt> <tap checkout>
#
# Rewrites the tag in each download URL and the four sha256 values, and nothing else. The
# formula has no `version` line: Homebrew reads the version from the URLs. Checks every
# checksum before it writes, so a missing one leaves the formula untouched. Safe to run
# again with the same input.
set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "usage: $0 <tag> <checksums.txt> <tap checkout>" >&2
  exit 2
fi

tag=$1
checksums=$2
formula=$3/Formula/spx.rb

[[ $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "error: tag must look like v1.2.3: $tag" >&2; exit 1; }
[ -f "$checksums" ] || { echo "error: no such file: $checksums" >&2; exit 1; }
[ -f "$formula" ] || { echo "error: no such file: $formula" >&2; exit 1; }

for asset in spx-macos-arm64 spx-macos-amd64 spx-linux-arm64 spx-linux-amd64; do
  sum=$(awk -v name="$asset" '$2 == name { print $1 }' "$checksums")
  [[ $sum =~ ^[0-9a-f]{64}$ ]] || { echo "error: no sha256 for $asset in $checksums" >&2; exit 1; }
done

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

awk -v tag="$tag" '
  NR == FNR { sum[$2] = $1; next }
  /^ *url "https:\/\/github.com\/sschlesier\/spx\/releases\/download\// {
    asset = $0
    sub(/^.*\//, "", asset)
    sub(/".*$/, "", asset)
    sub(/download\/[^\/]*\//, "download/" tag "/")
    print
    next
  }
  asset != "" && /^ *sha256 "/ {
    sub(/"[^"]*"/, "\"" sum[asset] "\"")
    asset = ""
    print
    next
  }
  { print }
' "$checksums" "$formula" > "$tmp"

! grep -q '^  version "' "$tmp" || { echo "error: formula has a version line; Homebrew reads it from the URLs" >&2; exit 1; }
[ "$(grep -cF "/download/$tag/spx-" "$tmp")" -eq 4 ] || { echo "error: expected four download URLs in the formula" >&2; exit 1; }
for asset in spx-macos-arm64 spx-macos-amd64 spx-linux-arm64 spx-linux-amd64; do
  sum=$(awk -v name="$asset" '$2 == name { print $1 }' "$checksums")
  grep -qF "sha256 \"$sum\"" "$tmp" || { echo "error: formula has no sha256 line for $asset" >&2; exit 1; }
done

cat "$tmp" > "$formula"
