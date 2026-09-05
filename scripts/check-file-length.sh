#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
baseline="${FILE_LENGTH_BASELINE:-scripts/quality-baseline/file-length.tsv}"
current=$(mktemp)
trap 'rm -f "$current"' EXIT

while IFS= read -r file; do
  [[ "$file" == web/static/vendor/* ]] && continue
  lines=$(awk 'END { print NR }' "$file")
  if ((lines > 500)); then
    printf '%s\t%s\n' "$file" "$lines" >>"$current"
  fi
done < <(git ls-files --cached --others --exclude-standard -- '*.go' '*.js' '*.css' '*.html')
sort -o "$current" "$current"

if ! awk -F '\t' '
  FILENAME == ARGV[1] {
    if ($0 !~ /^[[:space:]]*(#|$)/) allowed[$1] = $2
    next
  }
  {
    seen[$1] = 1
    if (!($1 in allowed)) {
      printf "new over-limit file: %s (%s lines)\n", $1, $2 > "/dev/stderr"
      failed = 1
    } else if ($2 > allowed[$1]) {
      printf "over-limit file grew: %s (%s -> %s lines)\n", $1, allowed[$1], $2 > "/dev/stderr"
      failed = 1
    } else if ($2 < allowed[$1]) {
      printf "file-length debt shrank: %s (%s -> %s); lower the baseline to ratchet it\n", $1, allowed[$1], $2 > "/dev/stderr"
      failed = 1
    }
  }
  END {
    for (path in allowed) {
      if (!(path in seen)) {
        printf "stale file-length baseline: %s (remove/update the entry)\n", path > "/dev/stderr"
        failed = 1
      }
    }
    exit failed
  }
' "$baseline" "$current"; then
  exit 1
fi

count=$(wc -l <"$current" | tr -d ' ')
echo "file-length regression gate passed (existing debt: $count file(s) over 500 lines)"
