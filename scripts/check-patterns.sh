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
  [[ "$file" == tests/* ]] && continue
  [[ "$file" == *_test.go ]] && continue
  [[ "$file" == docs/* ]] && continue
  sources+=("$file")
done < <(git ls-files --cached --others --exclude-standard -- '*.go' '*.js' '*.html')

# SQL files are fed exclusively into SQL_WILDCARD so the hard detector can
# cover CREATE TABLE / view definitions; other advisory scanners keep their
# existing scope and do not start matching *.sql.
sql_sources=()
while IFS= read -r file; do
  sql_sources+=("$file")
done < <(git ls-files --cached --others --exclude-standard -- '*.sql')

# Prefer ripgrep when available: the default `grep` here is toybox, which is
# orders of magnitude slower over this many files. Both emit `path:line:match`.
if command -v rg >/dev/null 2>&1; then
  grepper=(rg -n --no-heading --no-ignore -a -i --color never)
else
  grepper=(grep -EnHi)
fi

scan() {
  local severity=$1
  local rule=$2
  local regex=$3
  shift 3
  local matches
  local evidence
  matches=$("${grepper[@]}" -- "$regex" "$@" 2>/dev/null || true)
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
  scan advisory LOCAL_PAGINATION 'function[[:space:]]+renderPagination[[:space:]]*\(' "${sources[@]}"
  scan advisory IN_MEMORY_PAGINATION '\[[[:space:]]*start[[:space:]]*:[[:space:]]*end[[:space:]]*\]' "${sources[@]}"
  # An escape helper must never fall back to returning the unescaped original
  # string. Both rules are deliberately name-agnostic: matching the *shape* of
  # the defect instead of a name (`esc`) is what the 2026-09-13 convergence
  # needed, because three real instances were missed by a name-based scan.
  scan advisory ESCAPE_FAIL_OPEN_CHAIN '\|\| *function *\([a-zA-Z_$]*\) *\{ *return String\(' "${sources[@]}"
  scan advisory ESCAPE_FAIL_OPEN_GUARD '\? *(window\.)?escapeHtml\([^)]*\) *: *String\(' "${sources[@]}"
  scan hard SQL_WILDCARD 'SELECT[[:space:]]+([A-Za-z_][A-Za-z0-9_]*\.)?\*([[:space:],]|$)' "${sources[@]}"
fi

# Production browser and server code must derive ZenTao hosts from configuration
# or server-provided URLs. Tests and Demo fixtures intentionally remain outside
# this scope so environment examples do not become production contracts.
zentao_host_sources=()
while IFS= read -r -d '' file; do
  [[ "$file" == *_test.go || "$file" == */vendor/* || "$file" == docs/Demo/* || "$file" == */Demo/* ]] && continue
  zentao_host_sources+=("$file")
done < <(git ls-files -z --cached --others --exclude-standard -- internal/ web/static/ web/templates/)
if ((${#zentao_host_sources[@]} > 0)); then
  scan hard ZENTAO_HOST_LITERAL '(127\.0\.0\.1:8080|10\.211\.55\.4(:8080)?|changshu\.wrk\.oop\.cc|customer\.chandao\.net|pms\.csr)' "${zentao_host_sources[@]}"
fi

# Only run the *.sql variant of SQL_WILDCARD when SQL files actually exist;
# otherwise grep would be invoked with no path arguments and could read stdin.
if ((${#sql_sources[@]} > 0)); then
  scan hard SQL_WILDCARD 'SELECT[[:space:]]+([A-Za-z_][A-Za-z0-9_]*\.)?\*([[:space:],]|$)' "${sql_sources[@]}"
fi

fetch_sources=()
if ((${#sources[@]} > 0)); then
  for file in "${sources[@]}"; do
    case "$file" in
      web/static/js/app.js|web/static/js/ui.js|web/static/js/schedule/schedulefetch.js) ;;
      *.js) fetch_sources+=("$file") ;;
    esac
  done
fi
if ((${#fetch_sources[@]} > 0)); then
  scan advisory DIRECT_PAGE_FETCH '(^|[^A-Za-z0-9_])fetch[[:space:]]*\(' "${fetch_sources[@]}"
fi

handler_files=()
while IFS= read -r file; do
  [[ "$file" == internal/module/login/* ]] && continue
  handler_files+=("$file")
done < <(git ls-files --cached --others --exclude-standard -- 'internal/module/**/*handler*.go')
if ((${#handler_files[@]} > 0)); then
  routes=$("${grepper[@]}" -- '\.(GET|POST|PUT|DELETE)[[:space:]]*\(' "${handler_files[@]}" 2>/dev/null || true)
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
  [[ "$rule" != ZENTAO_HOST_LITERAL ]] && grep -Fxq "$key" "$expected" && continue
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
