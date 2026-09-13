#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
failed=0

# Untracked files directly under the repository root pollute every `--others`
# scan (check-patterns, check-secrets, check-file-length) and are the classic
# "AI additive bias" artifact: a report/dump dropped next to the code instead of
# under docs/. Reject them here so the root stays a clean entry surface.
while IFS= read -r -d '' file; do
  case "$file" in
    */*) continue ;;
  esac
  printf 'untracked file at repository root: %s\n' "$file" >&2
  failed=1
done < <(git ls-files --others --exclude-standard -z)

if ((failed)); then
  echo 'Move root-level artifacts under docs/ (or remove them); they must not sit at the repository root.' >&2
  exit 1
fi

echo 'root-artifact gate passed (no untracked files at repository root)'
