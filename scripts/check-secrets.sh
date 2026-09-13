#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
baseline="scripts/quality-baseline/secrets.tsv"
current=$(mktemp)
details=$(mktemp)
expected=$(mktemp)
trap 'rm -f "$current" "$details" "$expected"' EXIT

hash_text() {
  if command -v shasum >/dev/null 2>&1; then
    printf '%s' "$1" | shasum -a 256 | awk '{print $1}'
  else
    printf '%s' "$1" | sha256sum | awk '{print $1}'
  fi
}

all_files=()
while IFS= read -r file; do
  case "$file" in
    scripts/check-secrets.sh|scripts/quality-baseline/*|web/static/vendor/*) continue ;;
  esac
  # A tracked path whose worktree copy is gone (e.g. a pending deletion) has no
  # content to scan; skip it explicitly instead of letting awk emit an open error.
  [[ -f "$file" ]] || continue
  all_files+=("$file")
done < <(git ls-files --cached --others --exclude-standard -- \
  '*.yaml' '*.yml' '*.json' '*.toml' '*.env' '.env*' \
  '*.go' '*.js' '*.ts' '*.sh' '*.md' '*.pem' '*.key')

for file in "${all_files[@]}"; do
  case "$file" in
    *.yaml|*.yml|*.json|*.toml|*.env|.env*)
      while IFS=$'\t' read -r line_no line; do
        [[ -z "$line_no" ]] && continue
        printf '%s\t%s\t%s\n' SECRET_VALUE "$file" "$(hash_text "$line")" >>"$current"
        printf '%s %s:%s\n' SECRET_VALUE "$file" "$line_no" >>"$details"
      done < <(awk '
        /^[[:space:]]*#/ { next }
        {
          lower = tolower($0)
          if (match(lower, /^[[:space:]]*[a-z0-9_.-]*(password|passwd|secret|token|api[_-]?key|private[_-]?key)[a-z0-9_.-]*[[:space:]]*[:=]/)) {
            value = tolower(substr($0, RLENGTH + 1))
            gsub(/[[:space:]"]/, "", value)
            sub(/#.*/, "", value)
            if (value != "" && value !~ /\$\{/ && value !~ /placeholder|redacted|changeme|example|dummy/) {
              print FNR "\t" $0
            }
          }
        }
      ' "$file")
      ;;
  esac
done

if ((${#all_files[@]} > 0)); then
  signatures=$(grep -EnHi -- '-----BEGIN .*PRIVATE KEY-----|AKIA[0-9A-Z]{16}|ghp_[0-9A-Za-z]{20,}|github_pat_[0-9A-Za-z_]{20,}|sk-[0-9A-Za-z]{20,}' "${all_files[@]}" 2>/dev/null || true)
  while IFS=: read -r file line_no line; do
    [[ -z "$file" ]] && continue
    printf '%s\t%s\t%s\n' SECRET_SIGNATURE "$file" "$(hash_text "$line")" >>"$current"
    printf '%s %s:%s\n' SECRET_SIGNATURE "$file" "$line_no" >>"$details"
  done <<<"$signatures"
fi
sort -o "$current" "$current"

if [[ "${1:-}" == "--emit-baseline" ]]; then
  cat "$current"
  exit 0
fi

awk '!/^[[:space:]]*(#|$)/' "$baseline" | sort >"$expected"
if ! diff -u "$expected" "$current"; then
  echo "secret regression detected; findings (values suppressed):" >&2
  cat "$details" >&2
  exit 1
fi

count=$(wc -l <"$current" | tr -d ' ')
echo "secret regression gate passed (existing debt: $count fingerprint(s); values suppressed)"
