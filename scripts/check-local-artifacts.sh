#!/usr/bin/env bash
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
failed=0
# Inspect staged additions/changes, including force-added ignored files.
# Historical committed evidence is retained; never remove disk data.
while IFS= read -r -d '' file; do
  case "$file" in
    .run/*|.verify/*|.visual/*|verify-output/*|visual-output/*|test-results/*|playwright-report/*|screenshots/*|*/screenshots/*|docs/PRD|docs/PRD/*|docs/PRD/**)
      printf 'local artifact in index: %s\n' "$file" >&2
      failed=1 ;;
  esac
done < <(git diff --cached --name-only --diff-filter=ACMR -z)
if ((failed)); then
  echo 'Unstage local artifacts; retain the source files on disk.' >&2
  exit 1
fi
echo 'local-artifact index gate passed'
