# ADR 0001: Password MD5 isolation on shared ZenTao `zt_user.password`

## Status

Accepted — 2026-09-10. Scope-limited to the existing
`internal/module/profile/` write paths that touch `zt_user.password`.
Supersedes the open P0 entry in `docs/engineering/debt.md` ("User
create/reset uses MD5 for password writes") only for the narrow case
described below; the user-create and reset paths in
`internal/module/user/` remain open.

## Context

`AGENTS.md` MUST 7 forbids new password writes using MD5/SHA family
hashes and demands that legacy ZenTao verification compatibility be
isolated from Workbench-owned authentication. The 2026-09-10 debt
snapshot and the GLM scan (`code-review-2026-09-10.html`) both flag
`internal/module/profile/service.go` as currently calling
`encode.MD5` on both the old-password check and the new-password write.

The shared database is ZenTao's `zt_user.password` column, which
historically stores MD5 hex. The profile module is the Workbench endpoint
that lets a logged-in user change their own password. Two facts
constrain the fix:

1. Verifying the old password must round-trip with whatever ZenTao
   currently holds. Right now that is MD5 hex.
2. Writing a non-MD5 hash would break ZenTao's own login against the
   same row, since ZenTao's verification path is also MD5.

## Decision drivers

- **Do not break shared login.** ZenTao's authentication against
  `zt_user.password` must keep working; the Workbench profile endpoint
  must continue to produce values ZenTao can verify.
- **Do not silently broaden the violation.** Anything new outside the
  legacy isolation must not regress to MD5.
- **One bounded change.** The decision must fit inside the current
  debt-cleanup wave (Plan §2 B4) without dragging in a ZenTao-side
  migration.

## Options considered

| ID | Approach | Risk | Verdict |
|---|---|---|---|
| D1 | Isolate the call. Keep using `encode.MD5` here, mark the call site with the exact compat reason, suppress the `WEAK_PASSWORD_HASH` advisory for this one symbol, and bound the file with a structured comment that points back to this ADR. | Permanent compliance debt on this exact file. Acceptable because the file is one user-self-service write path. | **Chosen.** |
| D2 | Stop writing the hash locally. Delegate the change to a ZenTao-supplied change-password API and never persist the hash on the Workbench side. | Removes local hash state but ties availability and latency to a ZenTao endpoint that has not been confirmed to expose this surface, and removes the Workbench-side round-trip check the operator currently relies on. | Rejected for this wave. Keep open as a future option if ZenTao surfaces a verified change-password API. |
| D3 | Dual-write / migrate to a stronger hash on the Workbench side and coordinate the ZenTao side. | Requires ZenTao-side coordination; out of scope; forbidden to take unilaterally by Plan §2 B4 and AGENTS.md MUST 13. | Rejected. |

## Decision

Adopt **D1** for `internal/module/profile/service.go`:

1. Keep `encode.MD5` on the old-password check and the new-password
   write exactly where it is today. The values exchanged with the
   `zt_user.password` column must remain MD5 hex so ZenTao verification
   continues to round-trip.
2. Add a structured comment block above each call site that names the
   shared column, cites AGENTS.md MUST 7, and points at this ADR. The
   comment uses a single, greppable token (`ADR-0001 profile-md5
   isolation`) so future agents can locate every exemption without
   re-deriving the rule.
3. Register the two `encode.MD5` symbols in
   `scripts/quality-baseline/patterns.tsv` as `WEAK_PASSWORD_HASH`
   debt entries that name this ADR, so the gate keeps reporting them
   rather than failing or silently hiding them.
4. Refuse any **new** call to `encode.MD5` for password writes anywhere
   else in the repo. New calls fail `make check` until this ADR is
   amended; the advisory suppression above is **scoped to the two
   existing symbols in `internal/module/profile/service.go`** and does
   not cover user-create/reset in `internal/module/user/`.

## Consequences

- `make check-patterns` continues to report the two exempted symbols as
  documented debt; it does not fail because of them.
- Any future agent that adds a new MD5 password write is forced to
  amend this ADR or to add a fresh scoped entry. The suppression does
  not auto-extend.
- The profile change-password flow remains a single transactional
  write to `zt_user.password`; no schema migration, no ZenTao-side
  coordination, no new env var.
- The `internal/module/user/` user-create and reset paths are
  **explicitly out of scope** for this ADR. They remain on the debt
  list and require their own decision.

## Reversal conditions

This ADR is reversed, in whole or in part, when **all** of the
following hold:

1. ZenTao exposes a verified change-password API (or a verification
   shim) that the Workbench profile endpoint can call instead of
   writing the column directly. Evidence: a documented response shape
   and a working integration test against an isolated ZenTao instance.
2. The ADR owner confirms that the legacy MD5 round-trip in
   `internal/module/profile/service.go` is no longer required, and
   `encode.MD5` for password purposes is removed from the file.
3. The patterns baseline entry for the two symbols is removed in the
   same change.

A partial reversal (e.g. gating the write behind a feature flag while
D2 lands) is allowed but must be a new ADR that references this one.

## References

- `docs/engineering/debt.md` — open P0 "User create/reset uses MD5 for
  password writes" (only the `internal/module/user/` paths remain open
  after this ADR).
- `docs/plan/agent-governance-audit-20260907/REPORT.md` — audit
  evidence for the broader isolation guidance.
- `code-review-2026-09-10.html` §"密码 MD5 维持兼容" — originating
  finding.
- `AGENTS.md` MUST 7, MUST 13.
- Plan §2 B4 "改密 MD5（报告 P1，需决策）" — execution slot.