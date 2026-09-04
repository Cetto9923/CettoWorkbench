#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
baseline="scripts/quality-baseline/go-vet.txt"
current=$(mktemp)
expected=$(mktemp)
trap 'rm -f "$current" "$expected"' EXIT

mkdir -p tmp/gocache
status=0
GOCACHE="$root/tmp/gocache" go vet ./... >"$current" 2>&1 || status=$?
# Go 1.25+ prefixes each package with a `# package/path` header. Strip that
# header line during normalization so the exact diagnostic fingerprint is
# independent of the tool version; baseline intentionally stores only real
# diagnostics and is not padded with these tool-emitted headers.
sed "s|$root/||g" "$current" \
  | sed '/^[[:space:]]*$/d' \
  | grep -Ev '^# [^[:space:]]+$' \
  | sort -o "$current"
grep -Ev '^[[:space:]]*(#|$)' "$baseline" | sort >"$expected"

if ! diff -u "$expected" "$current"; then
  echo "go vet regression: diagnostics differ from the exact baseline" >&2
  exit 1
fi
if ((status == 0)) && [[ -s "$expected" ]]; then
  echo "go vet baseline is stale: vet is clean, remove the recorded debt" >&2
  exit 1
fi

count=$(wc -l <"$current" | tr -d ' ')
echo "go vet regression gate passed (existing debt: $count diagnostic(s))"
