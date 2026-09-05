#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
baseline="scripts/quality-baseline/architecture.tsv"
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

record() {
  printf '%s\t%s\t%s\n' "$1" "$2" "$(hash_file "$2")" >>"$current"
}

while IFS= read -r file; do
  if grep -Eq 'gorm\.io/gorm|"database/sql"|[[:alnum:]_]\.db\b|[[:alnum:]_]\.DB\b' "$file"; then
    record HANDLER_DATABASE_ACCESS "$file"
  fi
done < <(git ls-files --cached --others --exclude-standard -- 'internal/module/**/*handler*.go')

while IFS= read -r file; do
  if grep -Eq 'github\.com/gin-gonic/gin|\*gin\.Context' "$file"; then
    record REPO_HTTP_DEPENDENCY "$file"
  fi
done < <(git ls-files --cached --others --exclude-standard -- 'internal/module/**/*repo*.go')

while IFS= read -r file; do
  if grep -Eq '[[:alnum:]_]\.db\.(WithContext|Table|Model|Raw|Exec)|\*gorm\.DB' "$file"; then
    record SERVICE_DATABASE_ACCESS "$file"
  fi
done < <(git ls-files --cached --others --exclude-standard -- 'internal/module/**/*service*.go')

sort -u -o "$current" "$current"
awk '!/^[[:space:]]*(#|$)/' "$baseline" | sort >"$expected"
if ! diff -u "$expected" "$current"; then
  echo "architecture regression: fix the boundary violation; update baseline only when removing resolved debt" >&2
  exit 1
fi

count=$(wc -l <"$current" | tr -d ' ')
echo "architecture boundary regression gate passed (existing debt: $count file(s))"
