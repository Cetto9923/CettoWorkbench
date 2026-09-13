#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
baseline="scripts/quality-baseline/file-length.tsv"
current=$(mktemp)
trap 'rm -f "$current"' EXIT

# No separate anti-loosening guard against HEAD. The awk below already enforces
# the full ratchet (new file / grew / shrank / stale). The former guard compared
# the baseline against HEAD and was the exact cause of the permanent red state:
# it forbade recording a file that was genuinely over limit, blocking the only
# legal reconciliation path. Recording measured truth is not loosening. Use
# `--reconcile` to (re)record the measured truth as the ratchet floor.

while IFS= read -r -d '' file; do
  [[ "$file" == web/static/vendor/* ]] && continue
  [[ "$file" == */testdata/* ]] && continue
  [[ -f "$file" ]] || continue
  case "$file" in
    cmd/*|internal/*|web/templates/*|web/static/js/*|web/static/css/*|tests/*|large.go|new_large.go)
      ;;
    *)
      continue
      ;;
  esac
  lines=$(awk 'END { print NR }' "$file")
  if ((lines > 500)); then
    printf '%s\t%s\n' "$file" "$lines" >>"$current"
  fi
done < <(git ls-files -z --cached --others --exclude-standard -- '*.go' '*.js' '*.css' '*.html')
sort -o "$current" "$current"

if [[ "${1:-}" == "--reconcile" ]]; then
  printf '# path<TAB>current debt line count; every shrink must ratchet this number down\n' > "$baseline"
  cat "$current" >> "$baseline"
  echo "file-length baseline reconciled to measured truth ($(wc -l < "$current" | tr -d ' ') file(s) over 500 lines)"
  exit 0
fi

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
