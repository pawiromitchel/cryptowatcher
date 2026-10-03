#!/bin/bash
# Commits the Homebrew formula for a release to main.
#   update-formula.sh <version> <4 sha256 values> [-- remote branch]
#
# The formula is a generated file, so this always starts from the newest main instead of merging, and it
# never moves the formula backwards (releases finishing out of order). Retries if the push races.
set -euo pipefail
ARGS=("$@")
VERSION="${ARGS[0]:?version}"
SHAS=("${ARGS[@]:1:4}")
REMOTE="${ARGS[5]:-origin}"
BRANCH="${ARGS[6]:-main}"

for attempt in 1 2 3 4 5; do
  git fetch -q "$REMOTE" "$BRANCH"
  git reset -q --hard "$REMOTE/$BRANCH"

  current=$(sed -n 's|.*/releases/download/v\([0-9][0-9.]*\)/.*|\1|p' Formula/cryptowatcher.rb 2>/dev/null | head -n1 || true)
  if [ -n "$current" ] && [ "$current" != "$VERSION" ] \
     && [ "$(printf '%s\n%s\n' "$current" "$VERSION" | sort -V | tail -n1)" = "$current" ]; then
    echo "Formula is already at $current, newer than $VERSION; leaving it alone."
    exit 0
  fi

  mkdir -p Formula
  ./scripts/render-formula.sh "$VERSION" "${SHAS[@]}" > Formula/cryptowatcher.rb
  git add Formula/cryptowatcher.rb
  if git diff --cached --quiet; then echo "Formula is already at $VERSION."; exit 0; fi
  git commit -q -m "chore(brew): update formula to $VERSION"
  if git push -q "$REMOTE" "HEAD:$BRANCH"; then echo "Formula updated to $VERSION."; exit 0; fi
  echo "Push failed (attempt $attempt), retrying from the latest $BRANCH..." >&2
  sleep $((attempt * 2))
done
echo "Could not update the formula." >&2
exit 1
