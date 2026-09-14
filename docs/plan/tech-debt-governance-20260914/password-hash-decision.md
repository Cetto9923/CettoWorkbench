# Password hash decision (ZenTao MD5 compatibility)

**Date:** 2026-09-14
**Status:** DECIDED — keep MD5 for `zt_user.password` until product/security authorizes migration
**Governance wave:** Wave 1 (decision package only; **no storage algorithm change**)

## Relation to ADR 0001

[`docs/engineering/decisions/0001-password-md5-exemption.md`](../../engineering/decisions/0001-password-md5-exemption.md)
already accepted MD5 isolation for **profile** change-password (2026-09-10).
This Wave 1 package **extends the same isolation policy** to all remaining
Workbench paths that read/write ZenTao `zt_user.password` (`login` verify,
`user` create/batch/reset). It does **not** authorize algorithm migration.

## Decision

1. **Continue MD5 write + verify** for Workbench paths that share ZenTao’s `zt_user.password` column. ZenTao clients and existing rows expect hex MD5 of the plaintext password.
2. **Compatibility surface is closed:** all MD5 usage for passwords must go through `internal/pkg/encode` (`encode.MD5`). Call sites today:
   - Write: `user.Service` create / batch create / reset; `profile.Service` change password
   - Verify: `login.Service` login; `profile.Service` old-password check
3. **New Workbench-owned auth stores** (if ever introduced) MUST NOT copy MD5; use a modern KDF (e.g. Argon2id / bcrypt) and never invent a parallel weak hash.
4. **Silent migration is forbidden:** no dual-write, force-reset, or algorithm swap without explicit product + security approval and a cutover plan.

## Why MD5 remains

- Shared DB with ZenTao; password field ownership and format are ZenTao-shaped.
- Changing write algorithm without coordinated ZenTao / reset UX breaks login for all existing accounts.

## Migration options (not authorized yet)

| Option | Summary | When to choose |
|--------|---------|----------------|
| A. Stay MD5 forever while co-resident with ZenTao | Lowest risk; keep isolation + scanner advisory | Default until ZenTao exits or ZenTao upgrades |
| B. Dual-verify (MD5 or new hash); dual-write on successful login | Gradual; needs column or version flag | Soft cutover with long overlap |
| C. Force reset + modern hash | Cleanest crypto; high UX cost | Only with org-wide reset window |
| D. External IdP / SSO; stop storing password hashes | Removes local hash problem | Larger identity project |

Recommended next authorization ask: **Option A documented + advisory retained**, or **Option B design spike** if security requires a timeline.

## Code isolation checklist (already / required)

- [x] Single helper: `encode.MD5` with comment that it is ZenTao-compatible only
- [x] Call sites limited to login / user / profile (see `WEAK_PASSWORD_HASH` in `patterns.tsv`)
- [x] File-level banner on `encode.go`: compatibility-only
- [ ] CI remains advisory on `WEAK_PASSWORD_HASH` until migration closes the hits

## Explicit non-goals of this wave

- Changing hash algorithm or pepper
- Batch password resets
- Promoting `user/` to Golden Reference
- Removing `WEAK_PASSWORD_HASH` advisories by weakening the scanner
