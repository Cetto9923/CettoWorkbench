# Quality gates and completion

`make check` is the single local and CI entry. It executes:

1. tracked-Go `gofmt` verification;
2. `go test ./...`;
3. `go vet ./...` with an exact diagnostic baseline;
4. `git diff --check`;
5. the 500-line first-party file limit with exact non-growth baseline;
6. forbidden-pattern regression scanning;
7. secret scanning without printing values; and
8. Handler/Repo architecture-boundary scanning.

The **regression gate** blocks new violations, growth, changed secret findings,
stale baseline, failed tests, or unexpected vet diagnostics. The **debt report**
prints/documents existing findings; their presence is not hidden and does not
authorize copying them. A Phase 2 fix updates the precise baseline in the same
change.

Baselines live in `scripts/quality-baseline/`. They list individual files,
diagnostics, or finding fingerprints/counts; directory-wide ignores are
forbidden. New entries require an explicit debt decision, not a green-CI excuse.

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
