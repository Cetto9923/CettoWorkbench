#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
baseline="scripts/quality-baseline/patterns.tsv"
raw=$(mktemp)
current=$(mktemp)
expected=$(mktemp)
details=$(mktemp)
new_hard=$(mktemp)
new_advisory=$(mktemp)
trap 'rm -f "$raw" "$current" "$expected" "$details" "$new_hard" "$new_advisory"' EXIT

sources=()
while IFS= read -r file; do
  [[ "$file" == web/static/vendor/* ]] && continue
  sources+=("$file")
done < <(git ls-files --cached --others --exclude-standard -- '*.go' '*.js' '*.html')

# SQL files are fed exclusively into SQL_WILDCARD so the hard detector can
# cover CREATE TABLE / view definitions; other advisory scanners keep their
# existing scope and do not start matching *.sql.
sql_sources=()
while IFS= read -r file; do
  sql_sources+=("$file")
done < <(git ls-files --cached --others --exclude-standard -- '*.sql')

scan() {
  local severity=$1
  local rule=$2
  local regex=$3
  shift 3
  local matches
  local evidence
  matches=$(grep -EnHi -- "$regex" "$@" 2>/dev/null || true)
  while IFS=: read -r path line rest; do
    [[ -z "$path" ]] && continue
    evidence=$(printf '%s' "$rest" | sed -E 's/[[:space:]]+/ /g; s/^ //; s/ $//')
    printf '%s\t%s\t%s\t%s\t%s\n' \
      "$severity" "$rule" "$path" "$line" "$evidence" >>"$raw"
  done <<<"$matches"
}

if ((${#sources[@]} > 0)); then
  scan advisory DIRECT_GIN_SCALAR 'c\.(Query|PostForm)[[:space:]]*\(' "${sources[@]}"
  scan advisory WEAK_PASSWORD_HASH '(md5\.|encode\.MD5[[:space:]]*\()' "${sources[@]}"
  scan advisory INIT_FUNCTION 'func[[:space:]]+init[[:space:]]*\(' "${sources[@]}"
  scan advisory AD_HOC_PRINT '(fmt|log)\.Print(f|ln)?[[:space:]]*\(' "${sources[@]}"
  scan advisory UNSAFE_TEMPLATE_HTML 'template\.HTML' "${sources[@]}"
  scan advisory NEW_WINDOW '(target[[:space:]]*=[[:space:]]*"_blank|window\.open[[:space:]]*\()' "${sources[@]}"
  scan advisory LOCAL_ESCAPE_HTML 'function[[:space:]]+escapeHtml[[:space:]]*\(' "${sources[@]}"
  scan advisory LOCAL_PAGINATION 'function[[:space:]]+renderPagination[[:space:]]*\(' "${sources[@]}"
  scan advisory IN_MEMORY_PAGINATION '\[[[:space:]]*start[[:space:]]*:[[:space:]]*end[[:space:]]*\]' "${sources[@]}"
  scan hard SQL_WILDCARD 'SELECT[[:space:]]+([A-Za-z_][A-Za-z0-9_]*\.)?\*([[:space:],]|$)' "${sources[@]}"
  scan hard SQL_WILDCARD 'SELECT[[:space:]]+([A-Za-z_][A-Za-z0-9_]*\.)?\*([[:space:],]|$)' "${sql_sources[@]}"
fi

fetch_sources=()
for file in "${sources[@]}"; do
  case "$file" in
    web/static/js/app.js|web/static/js/ui.js|web/static/js/schedule/schedulefetch.js) ;;
    *.js) fetch_sources+=("$file") ;;
  esac
done
if ((${#fetch_sources[@]} > 0)); then
  scan advisory DIRECT_PAGE_FETCH '(^|[^A-Za-z0-9_])fetch[[:space:]]*\(' "${fetch_sources[@]}"
fi

handler_files=()
while IFS= read -r file; do
  [[ "$file" == internal/module/login/* ]] && continue
  handler_files+=("$file")
done < <(git ls-files --cached --others --exclude-standard -- 'internal/module/**/*handler*.go')
if ((${#handler_files[@]} > 0)); then
  routes=$(grep -EnH '\.(GET|POST|PUT|DELETE)[[:space:]]*\(' "${handler_files[@]}" 2>/dev/null || true)
  while IFS=: read -r path line rest; do
    [[ -z "$path" ]] && continue
    [[ "$rest" == *RequirePerm* ]] && continue
    evidence=$(printf '%s' "$rest" | sed -E 's/[[:space:]]+/ /g; s/^ //; s/ $//')
    printf '%s\t%s\t%s\t%s\t%s\n' \
      advisory ROUTE_PERMISSION_REVIEW "$path" "$line" "$evidence" >>"$raw"
  done <<<"$routes"
fi

while IFS=$'\t' read -r severity rule path line evidence ordinal; do
  fingerprint=$(printf '%s\t%s\t%s\t%s' "$rule" "$path" "$evidence" "$ordinal" | git hash-object --stdin)
  printf '%s\t%s\t%s\t%s\n' "$severity" "$rule" "$path" "$fingerprint" >>"$current"
  printf '%s\t%s\t%s\t%s\t%s\n' \
    "$severity" "$rule" "$path" "$line" "$fingerprint" >>"$details"
done < <(
  sort -t $'\t' -k1,1 -k2,2 -k3,3 -k5,5 -k4,4n "$raw" |
    awk -F '\t' 'BEGIN { OFS=FS }
      { key=$1 FS $2 FS $3 FS $5; occurrence[key]++; print $0, occurrence[key] }'
)
sort -u -o "$current" "$current"
sort -u -o "$details" "$details"

if [[ "${1:-}" == "--emit-baseline" ]]; then
  cat "$current"
  exit 0
fi

{ grep -Ev '^[[:space:]]*(#|$)' "$baseline" || true; } | sort >"$expected"
if ! awk -F '\t' 'NF != 4 || ($1 != "hard" && $1 != "advisory") { exit 1 }' "$expected"; then
  echo "invalid pattern baseline: expected severity, rule, path, fingerprint" >&2
  exit 1
fi

while IFS=$'\t' read -r severity rule path line fingerprint; do
  key=$(printf '%s\t%s\t%s\t%s' "$severity" "$rule" "$path" "$fingerprint")
  grep -Fxq "$key" "$expected" && continue
  if [[ "$severity" == hard ]]; then
    printf '%s\t%s\t%s\t%s\n' "$rule" "$path" "$line" "$fingerprint" >>"$new_hard"
  else
    printf '%s\t%s\t%s\t%s\n' "$rule" "$path" "$line" "$fingerprint" >>"$new_advisory"
  fi
done <"$details"

hard_count=$(awk -F '\t' '$1 == "hard" { count++ } END { print count+0 }' "$current")
advisory_count=$(awk -F '\t' '$1 == "advisory" { count++ } END { print count+0 }' "$current")
new_advisory_count=$(wc -l <"$new_advisory" | tr -d ' ')

if [[ -s "$new_advisory" ]]; then
  echo "advisory pattern findings not in the recorded inventory:" >&2
  cat "$new_advisory" >&2
fi
echo "advisory pattern scan completed ($advisory_count finding(s), $new_advisory_count new)"

if [[ -s "$new_hard" ]]; then
  echo "hard-pattern regression: new exact finding(s):" >&2
  cat "$new_hard" >&2
  echo "fix new findings; do not add a baseline entry just to make CI green" >&2
  exit 1
fi

echo "hard-pattern non-growth gate passed ($hard_count existing finding(s))"
