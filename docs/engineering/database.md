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
