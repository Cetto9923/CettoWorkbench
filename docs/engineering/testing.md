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
