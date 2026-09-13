# Quality gates and completion

`make check` is the common local and CI regression entry. It executes the gates
enumerated in the `Makefile` `check` target: local-artifact and root-artifact
index checks, tracked-Go `gofmt`, `go test ./...` plus the frontend tests
enumerated in the `Makefile`, `go vet ./...` with an exact diagnostic baseline,
`git diff --check`, the 500-line first-party file limit with exact non-growth
baseline, hard-pattern regression gating plus advisory pattern scanning, secret
scanning without printing values, and Handler/Repo architecture-boundary
scanning.

The **regression gate** blocks new violations, growth, changed secret findings,
stale baseline, failed tests, or unexpected vet diagnostics. The **debt report**
prints/documents existing findings; their presence is not hidden and does not
authorize copying them. A Phase 2 fix updates the precise baseline in the same
change.

Baselines live in `scripts/quality-baseline/`. Pattern entries identify one
finding by severity, rule, path, and a fingerprint of normalized matched source;
line numbers are diagnostics only. A hard-pattern run fails when any current
finding is absent from the baseline, so removing one finding cannot hide a
different addition with the same count. Resolved entries may remain until a
governance change removes them. Directory-wide ignores are forbidden. New hard
entries require an explicit debt decision, not a green-CI excuse.

Pattern severity reflects scanner certainty, not the importance of the
underlying rule. `SQL_WILDCARD` is a hard pattern: its match maps directly to the unconditional
`SELECT *` prohibition. `ZENTAO_HOST_LITERAL` rejects known ZenTao environment
hosts in `internal/`, `web/static/`, and `web/templates/` (excluding vendor and
`*_test.go`); tests and Demo are outside that host rule. This rule has no baseline
escape: URLs must come from configuration or server data. The known-host list
is explicit in `check-patterns.sh`, not a detector for every possible hostname.
`check-local-artifacts.sh` rejects newly staged/modified local runtime, screenshot,
verification and PRD HTML artifacts, including force-added ignored files. Existing
committed historical evidence is retained. This gate runs through `make check`;
it is not an installed Git hook. The following are
advisory/debt detectors because text matching cannot establish the necessary
context or exception:

- `DIRECT_GIN_SCALAR`, `DIRECT_PAGE_FETCH`, and `LOCAL_PAGINATION` flag local
  consistency/reuse review;
- `INIT_FUNCTION` and `AD_HOC_PRINT` require lifecycle/logging context;
- `WEAK_PASSWORD_HASH` requires distinguishing password writes from isolated
  legacy verification compatibility;
- `IN_MEMORY_PAGINATION` requires knowing whether the dataset is proven bounded;
- `UNSAFE_TEMPLATE_HTML` requires knowing the value's trust/sanitization
  contract; and
- `ROUTE_PERMISSION_REVIEW` is only a candidate list. Group and multiline
  middleware are valid and a shell scan is not permission architecture truth.

Advisory findings never fail `make check`; they are reported for review. A
human/Agent still enforces the applicable MUST after examining context.

Passwords, tokens, API keys, secrets, and private keys contain no real values in
tracked files. Examples use unmistakable placeholders; runtime values use env or
an approved secret source. Scanners report only rule/path/line metadata/hashes.
The 2026-09-07 secret scan reported zero existing fingerprints. This does not
prove credential rotation or reduced deployment grants. Historical findings
need dated evidence; rotation/migration needs appropriate authorization.

Only all-green required gates plus task-specific acceptance permit `done`,
`verified`, `ready`, `complete`, or `可交付`. A failure is `failed`/`partial`; a
baseline impediment is `BLOCKED BY EXISTING BASELINE`. Never relabel after failure.

UI acceptance includes authenticated interaction, responsive/visual inspection,
Console/network review, and loaded-asset proof. HTTP 200 or a login page alone is
not browser verification.

## Effective gates and limits

- The frontend Make target enumerates tests explicitly, not every test file.
  Explicitly run all added/affected tests and report exit codes. File existence
  and a green subset are not coverage.
- The workflow has a separate isolated MySQL integration job. Its existence
  does not certify concurrency, production schema, grants or latency; inspect
  actual executed cases. `make check` does not run tagged DB tests.
- `make check-gates` (which runs `scripts/test-quality-gates.sh`) is required
  for rules/gates/baselines. It uses a temporary fixture repository and tests
  scanners, not semantic authorization, lock order or page query budgets.
- Scanners use text/file-name heuristics. Inspect findings; never move SQL into
  unscanned names to pass. Repo SQL inside a Service file is a placement defect,
  not necessarily a Service receiver executing SQL. Neither is acceptable.
- Run checks on the exact handoff tree. An early failure means later gates did
  not execute; run relevant ones independently. Earlier/cached/other-worktree
  results do not certify changed files.
- Compare starting/final WIP and affected file hashes. Report external changes
  and reconcile before rerunning impacted checks. Do not attribute work to an
  identified model without execution evidence.

## Acceptance versus enforcement

Review changed MUST violations even if advisory. Record location, trigger,
effect, rule and minimal remedy. Adapters cannot force arbitrary Agents to
comply. Required CI, branch protection, independent review and least-privilege
DB grants are additional controls; verify them rather than assuming or enabling
them as a documentation side effect.

Documentation delivery and application readiness are separate. Existing test/
build failures remain **BLOCKED BY EXISTING BASELINE** at an audit handoff;
documenting a defect does not repair the application.

## Review evidence and severity

- Each finding records the observed revision/WIP, file/symbol, trigger, effect,
  applicable rule and evidence limits. Label source-confirmed defects,
  conditional runtime risks and reproduced failures separately. An inspection
  is not a live incident; a risk does not need a production incident to merit repair.
- Trace effective configuration and the assembled middleware/call path. An
  unused YAML field, import, comment or scanner match alone is not proof of a
  disabled control, SQL injection or an authorization bypass.
- Explain severity using reachability, affected data/users, preconditions and
  impact. P0 requires evidence of critical impact and urgency; a layer-placement
  defect alone is insufficient. Confidence and priority are separate dimensions.
- Compare prior findings by root cause and evidence scope before calling one new.
  A stale line number does not make a known defect new or resolved. Apply this
  evidence standard to other Agents' reviews before turning them into tasks.
- Remedies must state schema and business prerequisites. Do not assume an upsert
  supplies a missing unique key, every zero-row update is a conflict, sorting one
  loop prevents every deadlock, or every unused actor requires object restrictions.
  Follow database.md and architecture.md; unresolved business policy stays explicit.

## Runtime log access and lifecycle

These requirements apply to new or materially changed logging. Existing gaps
are debt, not permission to copy them; they do not authorize inspecting raw logs.

- SQL, audit and diagnostic logs MUST minimize data: exclude credentials/tokens,
  request bodies and unnecessary business content. Restrict both filesystem and
  application/debug access to authorized identities; a permission bit alone is
  not a complete confidentiality control.
- On POSIX, new sensitive log files SHOULD use `0600` and dedicated directories
  `0700`. Approved log collectors may use a dedicated group and narrowly scoped
  access such as `0640`/`0750`; record readers and the operational reason. Windows
  or managed storage needs equivalent ACLs. No world access to sensitive logs.
- Verify effective ownership/ACLs, existing files and rotated files. Creation
  mode applies to newly created files; it does not repair an existing file. Do
  not recursively chmod shared directories or change unrelated operational files.
- Before release, define size/age retention, rotation, storage budget and disposal
  for sensitive logs. Preserve required audit retention; never delete evidence to
  hide a failure. Disk/permission/rotation errors must be observable without
  leaking payloads or recursively flooding logs. Audit failure follows its
  explicit business transaction contract, not an invented silent-success fallback.
- Verify with temporary directories and synthetic sensitive values: redaction,
  access modes, rotation and failure handling. Do not read production payloads
  to prove a logging implementation works.
