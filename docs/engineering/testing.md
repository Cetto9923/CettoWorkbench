# Testing

| Class | Location | Contract |
|---|---|---|
| Go unit/package test | Beside package as `*_test.go` | Deterministic, isolated from real DB/network, fast under `go test ./...`. |
| Integration | `tests/integration/` or clearly named suite with `integration` build tag | Exercises DB, HTTP server, external protocol, or multiple modules. Setup and isolation are explicit. |
| E2E | `tests/e2e/` | Drives browser journeys against a known environment; keeps artifacts out of production source. |
| Fixture/sample/mock | Nearest `testdata/` | Non-production inputs ignored as package source by Go tooling. |

Do not mechanically move package tests to `tests/`. Do not let a test silently
connect to developer/production data. An integration test must fail or skip with
a precise missing-environment reason, never masquerade as unit success.

Tests SHOULD prioritize business rules, state transitions, object/route
permissions, SQL contracts, observed regressions, and complex algorithms.
Coverage is useful only when assertions protect behavior; do not add low-value
tests to raise a percentage.

Bug work SHOULD start with a reproducing test when isolatable. Refactors require
relevant tests before and after. UI work additionally requires authenticated
browser interaction, visual inspection, Console/network checks, and loaded-asset
confirmation.

`go test ./...` is required. Integration/E2E commands are task-specific until
those suites exist; touching such a boundary makes its command a required gate.

## Regression coverage and honest evidence

- New tests must be reachable from the appropriate runner. For documentation-only
  work, record missing runner coverage without editing it. Source regexes can
  check forbidden syntax but cannot replace real renderer/event/Service calls
  with both allow and deny cases.
- SQL mocks returning empty rows prove empty-result handling, not that a DB
  predicate excludes revoked/deleted records. Assert relevant predicates/args;
  use existing isolated MySQL tests for uniqueness, isolation and concurrency.
- After file extraction, tests must load the production exports/script order.
  Missing helpers and stale DOM mocks are contract failures. Do not delete
  assertions merely to get green tests.
- Permission/action changes need authenticated-but-ungranted and revoked cases,
  plus object/state denial at Service/write boundaries. Nil actor alone is not
  sufficient negative coverage.
- Shared writes need controlled concurrent connections/barriers, duplicate
  requests, rollback, stale state/ownership and cancellation tests. Sleeps and
  sequential mocks cannot prove concurrency safety. Use existing isolated
  integration infrastructure; never shared `zentaopms` fixtures.
- Missing environment is an explicit skip/block. Separate actual executed cases
  from fixture generation, compilation and skipped tests.
