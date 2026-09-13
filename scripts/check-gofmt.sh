#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
baseline="scripts/quality-baseline/gofmt.tsv"
current=$(mktemp)
expected=$(mktemp)
trap 'rm -f "$current" "$expected"' EXIT

hash_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    sha256sum "$1" | awk '{print $1}'
  fi
}

files=()
while IFS= read -r file; do
  [[ -f "$file" ]] || continue
  files+=("$file")
done < <(git ls-files --cached --others --exclude-standard -- '*.go')

if ((${#files[@]} > 0)); then
  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    printf '%s\t%s\n' "$file" "$(hash_file "$file")" >>"$current"
  done < <(gofmt -l "${files[@]}")
fi
sort -o "$current" "$current"
awk '!/^[[:space:]]*(#|$)/' "$baseline" | sort >"$expected"

if ! diff -u "$expected" "$current"; then
  echo "gofmt regression: format changed files with gofmt; remove fixed fingerprints from $baseline" >&2
  exit 1
fi

count=$(wc -l <"$current" | tr -d ' ')
echo "gofmt regression gate passed (existing debt: $count file(s))"
