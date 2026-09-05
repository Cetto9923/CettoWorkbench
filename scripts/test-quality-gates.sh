#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/scripts/quality-baseline" "$fixture/bin"
cp "$root"/scripts/check-*.sh "$fixture/scripts/"
cd "$fixture"
git init -q
for name in go-vet.txt gofmt.tsv architecture.tsv secrets.tsv file-length.tsv; do
  : >"scripts/quality-baseline/$name"
done

# Stub only the tool executable in this isolated repository. Production gates
# always invoke go vet and their canonical baselines; there is no eval bypass.
cat >bin/go <<'STUB'
#!/usr/bin/env bash
if [[ "$1" != vet ]]; then exit 99; fi
cat "$REVIEW_VET_OUTPUT"
exit "$REVIEW_VET_STATUS"
STUB
chmod +x bin/go
export PATH="$fixture/bin:$PATH"
export REVIEW_VET_OUTPUT="$fixture/vet-output"
export REVIEW_VET_STATUS=0
: >"$REVIEW_VET_OUTPUT"
passed=0
assert_result() {
  local want="$1" description="$2" status=0
  shift 2
  "$@" >"$fixture/result" 2>&1 || status=$?
  if { [[ "$want" == pass ]] && ((status != 0)); } ||
     { [[ "$want" == fail ]] && ((status == 0)); }; then
    echo "FAIL: $description (exit $status)"
    cat "$fixture/result"
    exit 1
  fi
  echo "PASS: $description"
  passed=$((passed + 1))
}

for gate in go-vet gofmt architecture secrets file-length; do
  assert_result pass "$gate empty baseline, clean tree" bash "scripts/check-$gate.sh"
done
printf '# comment\n' >scripts/quality-baseline/go-vet.txt
assert_result pass 'vet comment-only baseline' bash scripts/check-go-vet.sh
printf '# package/example\n' >"$REVIEW_VET_OUTPUT"
assert_result pass 'vet header-only output' bash scripts/check-go-vet.sh
printf 'foo.go:10:2: diagnostic\n' >"$REVIEW_VET_OUTPUT"
export REVIEW_VET_STATUS=1
assert_result fail 'vet new diagnostic rejected' bash scripts/check-go-vet.sh
cp "$REVIEW_VET_OUTPUT" scripts/quality-baseline/go-vet.txt
assert_result pass 'vet exact diagnostic baseline' bash scripts/check-go-vet.sh
: >"$REVIEW_VET_OUTPUT"
export REVIEW_VET_STATUS=0
assert_result fail 'vet stale baseline rejected' bash scripts/check-go-vet.sh
: >scripts/quality-baseline/go-vet.txt
export REVIEW_VET_STATUS=2
assert_result fail 'vet tool failure without output rejected' bash scripts/check-go-vet.sh
# The old production escape hatch must no longer bypass an actual tool failure.
assert_result fail 'vet ignores obsolete mock override' env GO_VET_MOCK_CMD=true bash scripts/check-go-vet.sh

printf 'package example\nfunc example( ) { }\n' >example.go
assert_result fail 'gofmt real unformatted file rejected' bash scripts/check-gofmt.sh
rm example.go
mkdir -p internal/module/example
printf 'package example\nimport "gorm.io/gorm"\n' >internal/module/example/example_handler.go
assert_result fail 'architecture real layer violation rejected' bash scripts/check-architecture.sh
rm internal/module/example/example_handler.go
printf 'password: synthetic_review_only\n' >fixture.yaml
assert_result fail 'secrets real synthetic finding rejected' bash scripts/check-secrets.sh
rm fixture.yaml

awk 'BEGIN { for (i=0; i<501; i++) print "// fixture" }' >large.go
assert_result fail 'file-length 501 lines with empty baseline rejected' bash scripts/check-file-length.sh
printf '# comment\n' >scripts/quality-baseline/file-length.tsv
assert_result fail 'file-length 501 lines with comment baseline rejected' bash scripts/check-file-length.sh
printf 'large.go\t501\n' >scripts/quality-baseline/file-length.tsv
assert_result pass 'file-length exact legacy baseline' bash scripts/check-file-length.sh
printf '// growth\n' >>large.go
assert_result fail 'file-length growth rejected' bash scripts/check-file-length.sh
rm large.go
assert_result fail 'file-length stale debt rejected' bash scripts/check-file-length.sh
printf 'Quality gate regression tests: %s passed\n' "$passed"
