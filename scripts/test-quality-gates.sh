#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"

echo "=== Running quality gate regression test suite (Phase G1) ==="
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

failed=0
pass_count=0

assert_pass() {
  local desc="$1"
  shift
  echo -n "Testing $desc... "
  if "$@"; then
    echo "PASS"
    pass_count=$((pass_count + 1))
  else
    echo "FAILED (expected success, got exit code $?)"
    failed=$((failed + 1))
  fi
}

assert_fail() {
  local desc="$1"
  shift
  echo -n "Testing $desc... "
  if "$@" >/dev/null 2>&1; then
    echo "FAILED (expected failure, got exit code 0)"
    failed=$((failed + 1))
  else
    echo "PASS (correctly failed)"
    pass_count=$((pass_count + 1))
  fi
}

# -----------------------------------------------------------------------------
# Scenario 1: EmptyBaseline - check-go-vet with empty/comment-only baseline
# -----------------------------------------------------------------------------
empty_baseline="$tmpdir/empty_baseline.txt"
touch "$empty_baseline"
comment_baseline="$tmpdir/comment_baseline.txt"
printf '# only comments\n# in baseline\n' >"$comment_baseline"

# 1.1 Clean tool output with empty baseline -> must PASS (exit 0)
assert_pass "go-vet with empty baseline and clean output" \
  env GO_VET_BASELINE="$empty_baseline" GO_VET_MOCK_CMD="true" bash scripts/check-go-vet.sh

# 1.2 Clean tool output with comment-only baseline -> must PASS (exit 0)
assert_pass "go-vet with comment-only baseline and clean output" \
  env GO_VET_BASELINE="$comment_baseline" GO_VET_MOCK_CMD="true" bash scripts/check-go-vet.sh

# -----------------------------------------------------------------------------
# Scenario 2: HeaderOnly vs RealDiagnostic
# -----------------------------------------------------------------------------
# 2.1 Only package header output -> normalized to empty, matches empty baseline -> must PASS
assert_pass "go-vet with only package header output" \
  env GO_VET_BASELINE="$empty_baseline" GO_VET_MOCK_CMD="printf '# workbench/internal/foo\n'" bash scripts/check-go-vet.sh

# 2.2 Real diagnostic output against empty baseline -> must FAIL (exit 1)
assert_fail "go-vet with real diagnostic against empty baseline" \
  env GO_VET_BASELINE="$empty_baseline" GO_VET_MOCK_CMD="printf 'foo.go:10:2: missing tag\n'" bash scripts/check-go-vet.sh

# 2.3 Real diagnostic matching baseline -> must PASS (exit 0)
diag_baseline="$tmpdir/diag_baseline.txt"
printf 'foo.go:10:2: missing tag\n' >"$diag_baseline"
assert_pass "go-vet with matching diagnostic in baseline" \
  env GO_VET_BASELINE="$diag_baseline" GO_VET_MOCK_CMD="bash -c 'printf \"foo.go:10:2: missing tag\n\"; exit 1'" bash scripts/check-go-vet.sh

# -----------------------------------------------------------------------------
# Scenario 3: ToolFailure - tool exits non-zero without diagnostics -> must FAIL
# -----------------------------------------------------------------------------
assert_fail "go-vet tool failure (exit 2) without diagnostics must not be swallowed" \
  env GO_VET_BASELINE="$empty_baseline" GO_VET_MOCK_CMD="bash -c 'exit 2'" bash scripts/check-go-vet.sh

# -----------------------------------------------------------------------------
# Scenario 4: EmptyBaseline on check-gofmt, check-architecture, check-secrets
# -----------------------------------------------------------------------------
# 4.1 Empty baseline parsing does not fail on grep pipefail
assert_pass "gofmt baseline parser handles empty file" \
  awk '!/^[[:space:]]*(#|$)/' "$empty_baseline"

assert_pass "architecture baseline parser handles empty file" \
  awk '!/^[[:space:]]*(#|$)/' "$empty_baseline"

assert_pass "secrets baseline parser handles empty file" \
  awk '!/^[[:space:]]*(#|$)/' "$empty_baseline"

# 4.2 New violation against empty baseline must FAIL (not false pass)
assert_fail "secrets check fails on empty baseline when existing debt exists" \
  env SECRETS_BASELINE="$empty_baseline" bash scripts/check-secrets.sh

assert_fail "architecture check fails on empty baseline when existing debt exists" \
  env ARCHITECTURE_BASELINE="$empty_baseline" bash scripts/check-architecture.sh

# -----------------------------------------------------------------------------
# Scenario 5: EmptyLengthBaseline - zero-byte & comment baseline in file-length awk
# -----------------------------------------------------------------------------
zero_byte_baseline="$tmpdir/zero_length.tsv"
touch "$zero_byte_baseline"

current_over="$tmpdir/current_over.tsv"
printf 'new_over_file.go\t501\n' >"$current_over"

current_under="$tmpdir/current_under.tsv"
printf 'clean_file.go\t500\n' >"$current_under"

current_empty="$tmpdir/current_empty.tsv"
touch "$current_empty"

awk_check() {
  local base="$1"
  local cur="$2"
  awk -F '\t' '
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
  ' "$base" "$cur"
}

# 5.1 Zero-byte baseline + 501-line file -> MUST FAIL (old NR==FNR leaked this)
assert_fail "file-length with zero-byte baseline and 501-line file must FAIL" \
  awk_check "$zero_byte_baseline" "$current_over"

# 5.2 Comment-only baseline + 501-line file -> MUST FAIL
assert_fail "file-length with comment-only baseline and 501-line file must FAIL" \
  awk_check "$comment_baseline" "$current_over"

# 5.3 Zero-byte baseline + 0 over-limit files -> MUST PASS
assert_pass "file-length with zero-byte baseline and 0 over-limit files must PASS" \
  awk_check "$zero_byte_baseline" "$current_empty"

# 5.4 Exact match baseline -> MUST PASS
exact_baseline="$tmpdir/exact.tsv"
printf 'legacy_large.go\t600\n' >"$exact_baseline"
current_exact="$tmpdir/current_exact.tsv"
printf 'legacy_large.go\t600\n' >"$current_exact"
assert_pass "file-length with matching baseline must PASS" \
  awk_check "$exact_baseline" "$current_exact"

# 5.5 File grew over baseline -> MUST FAIL
current_grew="$tmpdir/current_grew.tsv"
printf 'legacy_large.go\t601\n' >"$current_grew"
assert_fail "file-length with file growing over baseline must FAIL" \
  awk_check "$exact_baseline" "$current_grew"

echo ""
echo "=== Test Summary: $pass_count passed, $failed failed ==="
if ((failed > 0)); then
  exit 1
fi
echo "All quality gate regression tests PASSED."
exit 0
