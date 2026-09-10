#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
baseline="scripts/quality-baseline/file-length.tsv"
current=$(mktemp)
trusted=$(mktemp)
trap 'rm -f "$current" "$trusted"' EXIT

find_trusted_ref() {
  if [[ -n "${WB_BASE_REF:-}" ]] && git rev-parse --verify "${WB_BASE_REF}" >/dev/null 2>&1; then
    echo "$WB_BASE_REF"
    return
  fi
  if ! git diff --quiet HEAD -- "$baseline" 2>/dev/null; then
    echo "HEAD"
    return
  fi
  local upstream
  upstream=$(git rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null || true)
  if [[ -n "$upstream" ]] && git rev-parse --verify "$upstream" >/dev/null 2>&1; then
    local mb
    mb=$(git merge-base HEAD "$upstream" 2>/dev/null || true)
    if [[ -n "$mb" ]] && git cat-file -e "${mb}:${baseline}" 2>/dev/null; then
      echo "$mb"
      return
    fi
  fi
  if git rev-parse --verify HEAD^ >/dev/null 2>&1 && git cat-file -e "HEAD^:${baseline}" 2>/dev/null; then
    echo "HEAD^"
    return
  fi
  if git rev-parse --verify HEAD >/dev/null 2>&1 && git cat-file -e "HEAD:${baseline}" 2>/dev/null; then
    echo "HEAD"
    return
  fi
  echo ""
}

trusted_ref=$(find_trusted_ref)
if [[ -n "$trusted_ref" ]] && git cat-file -e "${trusted_ref}:${baseline}" 2>/dev/null; then
  git show "${trusted_ref}:${baseline}" >"$trusted"
  if ! awk -F '\t' '
    FILENAME == ARGV[1] {
      if ($0 !~ /^[[:space:]]*(#|$)/) trusted[$1] = $2
      next
    }
    {
      if ($0 ~ /^[[:space:]]*(#|$)/) next
      if (!($1 in trusted)) {
        printf "baseline expansion rejected: %s (new over-500-line files cannot be added to baseline)\n", $1 > "/dev/stderr"
        failed = 1
      } else if ($2 > trusted[$1]) {
        printf "baseline loosening rejected: %s (cannot increase baseline from %s to %s lines)\n", $1, trusted[$1], $2 > "/dev/stderr"
        failed = 1
      }
    }
    END {
      exit failed
    }
  ' "$trusted" "$baseline"; then
    exit 1
  fi
fi

while IFS= read -r -d '' file; do
  [[ "$file" == web/static/vendor/* ]] && continue
  lines=$(awk 'END { print NR }' "$file")
  if ((lines > 500)); then
    printf '%s\t%s\n' "$file" "$lines" >>"$current"
  fi
done < <(git ls-files -z --cached --others --exclude-standard -- '*.go' '*.js' '*.css' '*.html')
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
