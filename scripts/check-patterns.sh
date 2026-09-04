#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
baseline="scripts/quality-baseline/patterns.tsv"
raw=$(mktemp)
current=$(mktemp)
expected=$(mktemp)
trap 'rm -f "$raw" "$current" "$expected"' EXIT

sources=()
while IFS= read -r file; do
  [[ "$file" == web/static/vendor/* ]] && continue
  sources+=("$file")
done < <(git ls-files --cached --others --exclude-standard -- '*.go' '*.js' '*.html')

scan() {
  local rule=$1
  local regex=$2
  shift 2
  local matches
  matches=$(grep -EnHi -- "$regex" "$@" 2>/dev/null || true)
  while IFS=: read -r path line rest; do
    [[ -z "$path" ]] && continue
    printf '%s\t%s\t%s\n' "$rule" "$path" "$line" >>"$raw"
  done <<<"$matches"
}

if ((${#sources[@]} > 0)); then
  scan DIRECT_GIN_SCALAR 'c\.(Query|PostForm)[[:space:]]*\(' "${sources[@]}"
  scan WEAK_PASSWORD_HASH '(md5\.|encode\.MD5[[:space:]]*\()' "${sources[@]}"
  scan INIT_FUNCTION 'func[[:space:]]+init[[:space:]]*\(' "${sources[@]}"
  scan AD_HOC_PRINT '(fmt|log)\.Print(f|ln)?[[:space:]]*\(' "${sources[@]}"
  scan UNSAFE_TEMPLATE_HTML 'template\.HTML' "${sources[@]}"
  scan NEW_WINDOW '(target[[:space:]]*=[[:space:]]*"_blank|window\.open[[:space:]]*\()' "${sources[@]}"
  scan LOCAL_ESCAPE_HTML 'function[[:space:]]+escapeHtml[[:space:]]*\(' "${sources[@]}"
  scan LOCAL_PAGINATION 'function[[:space:]]+renderPagination[[:space:]]*\(' "${sources[@]}"
  scan IN_MEMORY_PAGINATION '\[[[:space:]]*start[[:space:]]*:[[:space:]]*end[[:space:]]*\]' "${sources[@]}"
  scan SQL_WILDCARD 'SELECT[[:space:]]+([A-Za-z_][A-Za-z0-9_]*\.)?\*([[:space:],]|$)' "${sources[@]}"
fi

fetch_sources=()
for file in "${sources[@]}"; do
  case "$file" in
    web/static/js/app.js|web/static/js/ui.js|web/static/js/schedule/schedulefetch.js) ;;
    *.js) fetch_sources+=("$file") ;;
  esac
done
if ((${#fetch_sources[@]} > 0)); then
  scan DIRECT_PAGE_FETCH '(^|[^A-Za-z0-9_])fetch[[:space:]]*\(' "${fetch_sources[@]}"
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
    printf '%s\t%s\t%s\n' ROUTE_WITHOUT_PERMISSION "$path" "$line" >>"$raw"
  done <<<"$routes"
fi

awk -F '\t' '{ key=$1 FS $2; count[key]++ } END { for (key in count) print key FS count[key] }' "$raw" | sort >"$current"

if [[ "${1:-}" == "--emit-baseline" ]]; then
  cat "$current"
  exit 0
fi

grep -Ev '^[[:space:]]*(#|$)' "$baseline" | sort >"$expected"
if ! diff -u "$expected" "$current"; then
  echo "forbidden-pattern regression: fix new findings; do not add baseline entries just to make CI green" >&2
  exit 1
fi

count=$(awk -F '\t' '{sum += $3} END {print sum+0}' "$current")
echo "forbidden-pattern regression gate passed (existing debt: $count finding(s))"
