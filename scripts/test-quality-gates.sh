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

# Reconciliation to the measured truth is legal: recording reality is not
# loosening, and the gate re-arms immediately after.
printf 'large.go\t502\n' >scripts/quality-baseline/file-length.tsv
assert_result pass 'file-length reconcile to measured truth' bash scripts/check-file-length.sh

# Fabricated growth above reality is rejected via the shrank check.
printf 'large.go\t503\n' >scripts/quality-baseline/file-length.tsv
assert_result fail 'file-length fabricated growth rejected' bash scripts/check-file-length.sh

# A genuinely new over-500 file may be recorded at its exact measured size.
awk 'BEGIN { for (i=0; i<501; i++) print "// fixture2" }' >new_large.go
printf 'large.go\t502\nnew_large.go\t501\n' >scripts/quality-baseline/file-length.tsv
assert_result pass 'file-length new over-limit file recorded at truth' bash scripts/check-file-length.sh

# A phantom baseline entry (path absent from the tree) is rejected as stale.
printf 'large.go\t502\nnew_large.go\t501\nghost.go\t501\n' >scripts/quality-baseline/file-length.tsv
assert_result fail 'file-length phantom baseline entry rejected' bash scripts/check-file-length.sh
rm new_large.go

# Stale debt: a baseline entry whose file no longer exists is rejected.
printf 'large.go\t502\n' >scripts/quality-baseline/file-length.tsv
rm large.go
assert_result fail 'file-length stale debt rejected' bash scripts/check-file-length.sh

# Ratchet down test: file shrank to <=500 lines and removed from baseline
awk 'BEGIN { for (i=0; i<400; i++) print "// fixture" }' >large.go
: >scripts/quality-baseline/file-length.tsv
assert_result pass 'file-length debt eliminated and baseline ratcheted' bash scripts/check-file-length.sh
rm large.go

# Host literals must fail in production, including untracked and Unicode paths.
: >scripts/quality-baseline/patterns.tsv
mkdir -p internal/example web/static/css web/static/vendor tests docs/Demo web/templates
for host in 127.0.0.1:8080 10.211.55.4 changshu.wrk.oop.cc customer.chandao.net pms.csr.cmbchina.com; do
  printf '// http://%s\n' "$host" >internal/example/host.go
  assert_result fail "production host rejected: $host" bash scripts/check-patterns.sh
  rm internal/example/host.go
done
printf '/* http://10.211.55.4:8080 */\n' >web/static/css/host.css
assert_result fail 'CSS host rejected' bash scripts/check-patterns.sh
rm web/static/css/host.css
printf '<!-- http://10.211.55.4:8080 -->\n' >web/templates/主机.html
assert_result fail 'Unicode template host rejected' bash scripts/check-patterns.sh
rm web/templates/主机.html
for file in internal/example/host_test.go web/static/vendor/host.js tests/host.js docs/Demo/host.html; do
  printf '// http://10.211.55.4:8080\n' >"$file"
done
assert_result pass 'tests and Demo/vendor host fixtures allowed' bash scripts/check-patterns.sh

assert_result pass 'local artifact index clean' bash scripts/check-local-artifacts.sh
mkdir -p .run docs/PRD/fixture/screenshots
printf 'synthetic\n' >.run/result.log
printf '<html>synthetic</html>\n' >docs/PRD/fixture/large.html
printf 'synthetic\n' >docs/PRD/fixture/screenshots/example.png
for file in .run/result.log docs/PRD/fixture/large.html docs/PRD/fixture/screenshots/example.png; do
  git add -f "$file"
  assert_result fail "force-added artifact rejected: $file" bash scripts/check-local-artifacts.sh
  git rm --cached -q "$file"
done
assert_result pass 'unstaged artifacts retained on disk allowed' bash scripts/check-local-artifacts.sh

printf 'Quality gate regression tests: %s passed\n' "$passed"
