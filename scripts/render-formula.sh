#!/bin/bash
# Prints the Homebrew formula for a release.
#   render-formula.sh <version> <sha:darwin-arm64> <sha:darwin-amd64> <sha:linux-arm64> <sha:linux-amd64>
# <version> has no leading "v".
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${1:?version}"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "not a version: $VERSION" >&2; exit 1; }
for sha in "${@:2:4}"; do
  [[ "$sha" =~ ^[0-9a-f]{64}$ ]] || { echo "not a sha256: $sha" >&2; exit 1; }
done
[ "$#" -eq 5 ] || { echo "usage: $0 <version> <4 sha256 values>" >&2; exit 1; }

sed -e "s/__VERSION__/$VERSION/" \
    -e "s/__SHA_DARWIN_ARM64__/$2/" \
    -e "s/__SHA_DARWIN_AMD64__/$3/" \
    -e "s/__SHA_LINUX_ARM64__/$4/" \
    -e "s/__SHA_LINUX_AMD64__/$5/" \
    scripts/formula.rb.tmpl
