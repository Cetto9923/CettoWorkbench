#!/usr/bin/env bash
set -euo pipefail

baseline=${QUALITY_BASELINE:-b23bd45de1753fd6020cc08c2a49ab415dc947f7}
git rev-parse --verify "$baseline^{commit}" >/dev/null

status=0
while IFS= read -r file; do
  [[ -f "$file" ]] || continue
  case "$file" in
    .git/*|*/vendor/*|vendor/*|node_modules/*|dist/*) continue ;;
  esac
  case "$file" in
    *.go|*.js|*.css|*.html|*.sh) ;;
    *) continue ;;
  esac

  current=$(wc -l < "$file" | tr -d ' ')
  (( current > 500 )) || continue

  if git cat-file -e "$baseline:$file" 2>/dev/null; then
    previous=$(git show "$baseline:$file" | wc -l | tr -d ' ')
  else
    previous=0
  fi
  delta=$((current - previous))
  printf '%s: %s 行；基线 %s 行；净增 %+d 行\n' "$file" "$current" "$previous" "$delta"
  if (( delta > 0 )); then
    status=1
  fi
done < <(git ls-files --cached --others --exclude-standard)

exit "$status"
