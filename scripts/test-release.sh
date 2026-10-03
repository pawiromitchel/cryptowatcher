#!/bin/bash
# Tests the release tooling (formula rendering and the update script) without touching the network.
set -euo pipefail
cd "$(dirname "$0")/.."
ROOT=$(pwd)
fail() { echo "FAIL: $*" >&2; exit 1; }

A=$(printf 'a%.0s' {1..64}); B=$(printf 'b%.0s' {1..64}); C=$(printf 'c%.0s' {1..64}); D=$(printf 'd%.0s' {1..64})

# --- render-formula.sh ---
out=$(./scripts/render-formula.sh 1.2.3 "$A" "$B" "$C" "$D")
grep -q 'releases/download/v1.2.3/cryptowatcher-darwin-arm64.tar.gz' <<<"$out" || fail "version not rendered"
for sha in "$A" "$B" "$C" "$D"; do grep -q "sha256 \"$sha\"" <<<"$out" || fail "sha $sha not rendered"; done
! grep -q '__' <<<"$out" || fail "unreplaced placeholder left in formula"
grep -A1 'darwin-arm64' <<<"$out" | grep -q "$A" || fail "darwin-arm64 sha misplaced"
grep -A1 'linux-amd64' <<<"$out" | grep -q "$D" || fail "linux-amd64 sha misplaced"
if command -v ruby >/dev/null; then ruby -c <<<"$out" >/dev/null || fail "formula is not valid Ruby"; fi

./scripts/render-formula.sh 1.2.3 "$A" "$B" "$C" nothex >/dev/null 2>&1 && fail "bad sha accepted"
./scripts/render-formula.sh v1.2.3 "$A" "$B" "$C" "$D" >/dev/null 2>&1 && fail "v-prefixed version accepted"
./scripts/render-formula.sh 1.2.3 "$A" "$B" "$C" >/dev/null 2>&1 && fail "missing sha accepted"

# --- update-formula.sh against a throwaway remote ---
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
git init -q --bare "$tmp/remote.git"
git clone -q "$tmp/remote.git" "$tmp/work" 2>/dev/null
cd "$tmp/work"
git config user.name test; git config user.email test@example.com
git checkout -q -b main
mkdir scripts && cp "$ROOT"/scripts/*.sh "$ROOT"/scripts/formula.rb.tmpl scripts/
git add -A && git commit -q -m init && git push -q origin main

./scripts/update-formula.sh 1.2.0 "$A" "$B" "$C" "$D" >/dev/null
grep -q 'download/v1.2.0/' Formula/cryptowatcher.rb || fail "formula not created"
git -C "$tmp/remote.git" show main:Formula/cryptowatcher.rb |  grep -q 'download/v1.2.0/' || fail "formula not pushed"

./scripts/update-formula.sh 1.2.0 "$A" "$B" "$C" "$D" | grep -q "already at 1.2.0" || fail "idempotence"
./scripts/update-formula.sh 1.1.0 "$A" "$B" "$C" "$D" | grep -q "newer than 1.1.0" || fail "formula moved backwards"
./scripts/update-formula.sh 1.10.0 "$B" "$A" "$D" "$C" >/dev/null
git -C "$tmp/remote.git" show main:Formula/cryptowatcher.rb | grep -q 'download/v1.10.0/' || fail "1.10.0 must sort after 1.2.0"

echo "release tooling OK"
