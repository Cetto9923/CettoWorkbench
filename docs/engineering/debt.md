# Existing engineering debt

Phase 1 does not modify the business implementation below. Severity reflects
risk, not authorization to repair. Machine-readable non-growth baselines live in
`scripts/quality-baseline/`.

| Location | Existing problem | Severity | Phase 2 recommendation |
|---|---|---:|---|
| `configs/config.yaml` | Three tracked password fields contain non-placeholder values; values are not reproduced here. | P0 | Rotate credentials, commit placeholders, verify env/secret injection, then remove fingerprints. |
| `internal/module/user/service.go`, `internal/pkg/encode/encode.go` | User create/reset uses MD5 for password writes. | P0 | Confirm ZenTao auth contract, isolate compatibility, design migration, prohibit new MD5 writes. |
| `internal/model/basemodel.go:17` | Malformed GORM struct tag makes `go vet ./...` fail. | P1 | Correct it in a focused change and remove exact vet baseline. |
| `internal/module/user/handler.go` | POST/form/Flash/redirect conflicts with old JSON contract; file exceeds 500 lines. | P1 | Decide/migrate contracts with browser tests, then split by capability. |
| `internal/module/schedule/handler.go` | Schedule routes have group authentication but no explicit capability permission. | P1 | Add capability permissions, Service object checks, and authorization tests. |
| `internal/module/dept/service.go` | Service reaches through `repo.db` and owns a GORM transaction/callback, crossing the Service -> Repo boundary. | P1 | Move transaction-capable persistence behind Repo methods while leaving business orchestration in Service. |
| `internal/module/po/servicetodo.go`, `internal/module/po/repotodo.go` | Todo loads broad sets, filters/sorts/pages in Go, and reloads account dictionary up to three times. | P1 | Create bounded cross-object query plan; SQL-filter/count/page; load dictionary once. |
| `internal/module/po/reponotice.go` | Notifications load all rows, then filter/count/page in memory. | P1 | Move filter/count/page and bounded aggregates to SQL. |
| `internal/module/po/service.go`, `internal/module/po/servicefollow.go`, `internal/module/po/repokpi.go` | Homepage stage loops and per-KPI queries create page query fan-out. | P1 | Measure query plan, consolidate compatible aggregates/ID sets, add contract tests. |
| `internal/module/schedule/service_window.go` | Window lists query capacity, consumed hours, count/stats per window. | P1 | Batch aggregates/capacity inputs and assemble without per-window SQL. |
| `internal/module/user/service.go` | Batch create checks account existence once per row. | P1 | Fetch conflicts once and rely on verified transaction/uniqueness. |
| `internal/module/user/` | Former Golden Reference violates security, protocol, size, schema-boundary rules. | P1 | Keep non-authoritative; promote only a future all-green candidate. |
| `scripts/quality-baseline/file-length.tsv` entries | Multiple first-party Go/JS/CSS/HTML files exceed 500 lines; two exceed 1,100. | P1 | Split one capability at a time with before/after tests. |
| `web/static/js/po/*.js`, schedule scripts | Escape, pagination, direct fetch/error, and new-window behavior repeat locally. | P2 | Inventory contracts; consolidate only stable identical behavior; add browser coverage. |
| Repository tests | Unit tests are sparse; no explicit integration/E2E suites; SQL/permission contracts largely unprotected. | P2 | Add isolated integration fixtures and focused E2E; prioritize rules/permissions/query contracts. |
| `scripts/quality-baseline/gofmt.tsv` entries | Six tracked Go files have formatting debt. | P2 | Format when deliberately touched and remove exact fingerprint. |
| `go.mod:71` | `replace workbench => /home/wds/repo/workbench` couples module resolution to one developer's absolute filesystem path. | P2 | In Phase 2, confirm why the replacement exists and whether it can be removed or replaced portably. |

The complete >500-line list is stored once in the exact non-growth baseline.
