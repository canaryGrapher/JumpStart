#!/usr/bin/env bash
# Publish docs/wiki/*.md to the GitHub Wiki for canaryGrapher/JumpStart.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/docs/wiki"
OWNER_REPO="${OWNER_REPO:-canaryGrapher/JumpStart}"
WIKI_URL="https://github.com/${OWNER_REPO}.wiki.git"
TMP="${TMPDIR:-/tmp}/jumpstart-wiki-publish-$$"

if [[ ! -d "$SRC" ]]; then
  echo "missing $SRC" >&2
  exit 1
fi

TOKEN="$(gh auth token)"
AUTH_URL="https://x-access-token:${TOKEN}@github.com/${OWNER_REPO}.wiki.git"

cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

if ! git ls-remote "$AUTH_URL" HEAD &>/dev/null; then
  cat >&2 <<EOF
Wiki git repo does not exist yet.

Open https://github.com/${OWNER_REPO}/wiki
Click "Create the first page", save any placeholder, then re-run:
  $0
EOF
  exit 1
fi

git clone --depth 1 "$AUTH_URL" "$TMP"
# Replace wiki content with docs/wiki (keep .git)
find "$TMP" -maxdepth 1 -type f -name '*.md' -delete
cp "$SRC"/*.md "$TMP"/
# README.md in docs/wiki is for the repo docs folder, not the wiki Home.
rm -f "$TMP/README.md"

cd "$TMP"
git add -A
if git diff --cached --quiet; then
  echo "Wiki already up to date."
  exit 0
fi

git -c user.name="JumpStart Wiki Bot" -c user.email="wiki@users.noreply.github.com" \
  commit -m "docs: sync developer wiki from docs/wiki"
git push origin HEAD
echo "Published to https://github.com/${OWNER_REPO}/wiki"
