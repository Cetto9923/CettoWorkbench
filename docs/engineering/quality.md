# Quality gates and completion

`make check` is the single local and CI entry. It executes:

1. tracked-Go `gofmt` verification;
2. `go test ./...`;
3. `go vet ./...` with an exact diagnostic baseline;
4. `git diff --check`;
5. the 500-line first-party file limit with exact non-growth baseline;
6. hard-pattern regression gating plus advisory pattern scanning;
7. secret scanning without printing values; and
8. Handler/Repo architecture-boundary scanning.

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
underlying rule. Only `SQL_WILDCARD` is currently a hard pattern: its match maps
directly to the unconditional `SELECT *` prohibition. The following are
advisory/debt detectors because text matching cannot establish the necessary
context or exception:

- `DIRECT_GIN_SCALAR`, `DIRECT_PAGE_FETCH`, `LOCAL_ESCAPE_HTML`, and
  `LOCAL_PAGINATION` flag local consistency/reuse review;
- `NEW_WINDOW` requires a product reason and safe opener handling, so the token
  alone is not a violation;
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
Existing tracked credentials are debt; Phase 1 fingerprints without editing or
displaying them. Rotation/migration requires separate authorization.

Only all-green required gates plus task-specific acceptance permit `done`,
`verified`, `ready`, `complete`, or `可交付`. A failure is `failed`/`partial`; a
baseline impediment is `BLOCKED BY EXISTING BASELINE`. Never relabel after failure.

UI acceptance includes authenticated interaction, responsive/visual inspection,
Console/network review, and loaded-asset proof. HTTP 200 or a login page alone is
not browser verification.
