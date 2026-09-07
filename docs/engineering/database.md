# Database and performance

## Schema ownership comes first

Every touched table MUST be classified before implementation:

| Ownership | Required treatment |
|---|---|
| Workbench-owned | Follow approved audit fields, timestamps, indexes, and soft-delete model. New schema belongs in the approved Workbench migration/install path. |
| Existing ZenTao | Match deployed schema and semantics exactly, including historical names such as `deleted`, `fromDemand`, and `action`. Do not assume `BaseModel`, `deletedAt`, `createdBy`, or `updatedBy` for style. |

`zt_` does not by itself mean Workbench ownership. Verify the actual source,
schema, and writer. Never change production ZenTao schema as a side effect.

## Query contract

Default list execution is:

```text
SQL filter -> SQL sort -> SQL count -> SQL Limit/Offset
```

Keyset or another bounded strategy MAY replace Limit/Offset when required. A
small, proven-bounded lookup MAY be loaded in memory, but its upper bound and
reason must be documented. Export is a separate contract and still needs an
explicit bound/streaming decision.

The following are MUST-NOT patterns:

- large/unbounded `Find` or `Pluck`, followed by Go filtering or pagination;
- moving filters the database can enforce into Go;
- SQL in a per-row loop or any avoidable N+1;
- reading the same base dictionary multiple times in one HTTP request;
- one SQL call per metric without reviewing the whole page query plan;
- `SELECT *`, alias wildcards, or unnecessary text/blob columns;
- interpolating values into SQL strings; and
- duplicating queries merely to make functions appear independent.

Before a complex aggregation page, record expected SQL count, cardinality,
repeated sources, batch opportunities, and index impact. There is no universal
fixed query-count budget: review the actual page and data.

Service owns transaction and authorization decisions. Dynamic clauses may use
only closed server-controlled fragments; user values stay in placeholders.

## Query acceptance

- Loading all matching IDs is still unbounded loading. Narrow projections do
  not make Go deduplication/count/pagination bounded. Reuse a SQL candidate set
  for count and page, with stable tie-breaking and server-owned scope/filters.
  Do not filter only the current page in JS while reporting database-wide totals.
  If counters intentionally use a broader scope, label and test that contract.
- Batch enrichment must remain bounded as page size grows. A method named
  "batch" with one lookup per ID is still N+1. Include shell/sidebar queries in
  page cost; do not load a whole dashboard to obtain a navigation badge.
- Record before/after SQL count, rows examined/returned, representative plans,
  data cardinality/distribution, latency percentiles and pool wait. Fewer SQL
  calls can still scan more rows. Do not fix fan-out merely with goroutines,
  larger pools, unauthorized caching, or assumed indexes. SQL mocks prove SQL
  construction, not execution performance.
- Performance claims identify engine/version, topology, isolation, dataset,
  concurrency and measurement method. Without environment access report
  UNVERIFIED and provide an isolated validation plan.

## Shared ZenTao database and concurrent writes

These requirements apply to new or materially changed shared writes. Existing
debt is recorded, not permission to copy it. They do not authorize DB access.

1. **Environment/ownership.** A config, login-path name or DSN is not permission
   to connect. Confirm the authorized target and read/write scope. Read actual
   DDL/index evidence for affected tables; identify writers and owned fields.
   Separate live evidence from install/model/query assumptions. After instance
   or engine migration, old evidence is historical. No AutoMigrate, runtime
   CREATE, GRANT, DDL or load tests on a shared business DB as a feature side step.
2. **Atomicity.** Service defines atomic effects; Repo executes them on one
   primary transaction-bound connection. Relationship/authorization rechecks and
   audit writes use that transaction when consistency requires it. No replica
   read or outer Repo handle inside the atomic operation. No HTTP, LLM, file I/O,
   user interaction or sleep while holding DB locks.
3. **Concurrent state.** Validation outside the transaction is not a write-time
   guarantee. Use verified conditional updates with expected state/version/
   ownership, or locked reads and in-transaction rechecks, for shared mutable
   rows. Update only task-owned fields. Define zero affected rows (including
   no-ops), deletion, reassignment and conflict responses; never silently
   overwrite a newer ZenTao edit.
4. **Lock order.** Document table/row/index acquisition order across Workbench
   entries and relevant ZenTao writers. Canonicalize existing-object IDs; reject
   contradictory duplicate actions. Do not lock in arbitrary client order.
   Acquire/revalidate the needed scope/lock set before mutation. Parent locking
   alone does not prove safety across parents or other writers. Do not reorder
   dependent business operations blindly; no blanket table locks.
5. **Finite work.** Freeze request-size, batch-count, transaction-duration,
   query/cancellation, lock-wait and pool budgets in the feature/operations
   contract before release, with units and measured justification. Enforce
   batch bounds server-side before opening the transaction. No universal numeric
   limits without environment evidence; no partial commits that break required
   atomicity. Budget connections across pools/processes and reserve ZenTao capacity.
6. **Retry/uniqueness.** Transactions and SELECT-before-INSERT are not idempotency.
   Verify unique keys and soft-delete semantics; nullable unique columns do not
   automatically enforce one active row. Define duplicate/concurrent requests,
   error 1213, 1205 and uncertain commit outcomes separately. Retry only a
   rolled-back whole idempotent operation with bounded attempts/backoff outside
   locks and fresh validation. Never blindly retry one statement or swallow an
   unrelated duplicate-key error. Do not auto-add schema for idempotency.
7. **Read/write pools.** Two handles do not prove least privilege or replica
   topology. Require deployed grants evidence and defined read-unavailable
   behavior. Mutation decisions must not trust replica lag. Define read-after-
   write consistency for follows, notice reads, permissions and lists before
   using a replica. Do not silently promote failed reads to privileged accounts.
8. **Evidence.** Cover rollback, reverse-order overlapping writes, duplicate
   requests, reassignment/deletion races, cancellation and uncertain commit in
   the existing isolated MySQL suite. Sequential mocks cannot prove deadlock
   safety. Shared/live probing needs explicit scope: collect bounded redacted
   metadata/digests only. Raw PROCESSLIST, InnoDB status and SQL can contain
   business data/credentials; do not publish them or automatically kill sessions.

Ordinary InnoDB consistent reads usually do not take row locks at READ COMMITTED
or REPEATABLE READ; this is not true of every read. Locking reads, writes,
`INSERT ... SELECT`, SERIALIZABLE and metadata locks need separate analysis.
Long snapshots can impede purge; heavy reads can exhaust I/O/CPU/pools without
deadlocks. Isolation changes are not a universal deadlock fix. Validate against
the deployed engine/version; official references:
[statement locks](https://dev.mysql.com/doc/refman/8.0/en/innodb-locks-set.html),
[deadlock handling](https://dev.mysql.com/doc/refman/8.0/en/innodb-deadlocks-handling.html).
