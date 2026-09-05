# 性能基线与结果等价契约报告 (Phase P0 / P0.1)

- **采集时间**: 2026-09-05 14:20 CST
- **测试环境**: Apple M5 / macOS (Darwin arm64)
- **基线输入 HEAD**: `2b4ed960ed05f7635ba4c9f389d7e53746b409aa`
- **隔离环境**: 本地 Docker 独立 MySQL 8.4 容器 (`127.0.0.1:3307`, Database: `workbench_test_perf`)
- **状态结论**: **COMPLETE** (从原 `PARTIAL / DATA REQUIRED` 升级为 `COMPLETE`)

---

## 一、真实隔离 MySQL 性能与查询数量基线 (Live Isolated MySQL Harness)

在完全隔离的本地 MySQL 8.4 环境中运行真实生产代码链路 (`FindNotices`, `FindTodoItems`, `ListWindowCards`)，通过 GORM Callback Query Counter 精确统计当前老实现的 SQL 查询开销与执行结果：

| 生产链路 | 测试数据集规模 | 返回条目数 | 实际执行 SQL 查询数 | 瓶颈分析 |
|---|---|---|---|---|
| **通知列表** (`poRepo.FindNotices`) | 100 条通知，28 条目标可见 | 20 (分页 Page 1) | **2 次 SQL 查询** | 无论目标通知有多少，首查无分页拉取全量通知行，再在内存中执行 2 次全量过滤和切片。 |
| **待办列表** (`poRepo.FindTodoItems`) | 50 需求 + 50 任务 + 50 Bug + 10 附加待办 | 148 条待办 | **21 次 SQL 查询** | 多表独立拉取 (zt_demand, zt_task, zt_bug 及各类审批/项目/评审等 10+ 表)，聚合所有 ID 后在 Go 内存执行 7 维过滤与 $O(N \log N)$ 排序。 |
| **版本窗口** (`scheduleSvc.ListWindowCards`) | 10 个版本窗口 | 10 个窗口卡片 | **63 次 SQL 查询** | **$O(N)$ 线性查询扇出**: 1 次查询查出 10 个窗口后，在循环中为每个窗口分别执行 `CalcCapacity` (查节假日/工时)、`GetWindowConsumedHours`、`GetWindowDemandCount`，总查询数高达 $3 + 6N = 63$ 次。 |

### 老实现结果基准快照 (Old Implementation Result Fixture)
老实现的输出结果已固化保存在 `tests/integration/testdata/performance/old_impl_results.json`，供后续 P1 与 P2 进行结果等价性比对：
- **Notices**: Total = 28, Unread = 18, Filtered = 28, Returned Page = 20, Categories: all = 28, business = 22, collaboration = 4, approval = 2.
- **Todos**: Total = 148, Filtered = 148.
- **Windows**: Count = 10, Total Queries = 63.

---

## 二、合成基准测试数据与增长趋势 (Synthetic Baseline)

通过 `tests/integration/performance_baseline_test.go` 测量得出的老实现内存计算开销基线：

| 模块与测试场景 | 规模 N | 平均延迟 (ns/op) | 每次分配内存 (B/op) | 每次分配次数 (allocs/op) | 增长趋势 |
|---|---|---|---|---|---|
| **通知内存过滤** (`Notices_Filter`) | 100 | 1,881 ns/op | 18,432 B/op | 1 allocs/op | 基准 |
| **通知内存过滤** (`Notices_Filter`) | 1,000 | 18,643 ns/op | 172,032 B/op | 1 allocs/op | **9.9x 延迟增长，9.3x 内存增长** |
| **待办内存排序切片** (`Todos_SortSlice`) | 100 | 3,740 ns/op | 24,872 B/op | 4 allocs/op | 基准 |
| **待办内存排序切片** (`Todos_SortSlice`) | 1,000 | 35,655 ns/op | 229,672 B/op | 4 allocs/op | **9.5x 延迟增长，9.2x 内存增长** |

---

## 三、P1 / P2 结果等价与正确性合同 (Equivalence Contract)

后续 P1 与 P2 优化必须严格满足以下结果等价性：

### 1. 通知优化 (P1) 结果等价合同
- **分页与排序**: 必须与现有 `createdDate DESC, id DESC` 完全一致。
- **分类计数体系**:
  - `Total`、`Unread`、`Action`、`Abnormal`、`Today` 针对当前用户全部可见通知。
  - `Categories`（全部、审批、需求、项目、代码等）受所有筛选影响，但不受 Category 自身筛选影响。
  - `Filtered` 为应用当前全部过滤后的总条目数。
- **关键字匹配**: 必须保证 `%` 与 `_` 作为字面值匹配，不可直接拼入 SQL `LIKE` 成为通配符。
- **零日期与空值**: `0000-00-00` 或空日期统一规范化，避免解析崩溃。

### 2. 待办优化 (P2) 结果等价合同
- **全局分页**: 必须由数据库执行全局稳定排序后分页，禁止各对象类型提前截断后拼凑。
- **排序规则**:
  - 优先级优先（P0 > P1 > P2 > P3 > P4）。
  - 空截止日期排在最后。
  - 优先级相同时，严格按 `id DESC` 稳定排序。
- **7 维 AND 过滤**:
  - Tab、Focus、Relation、Responsibility、Stage、Action、Keyword 规则与现有 Go 内存过滤严格等价。
- **所有权显示**: 关键字支持对责任人展示名检索。

---

## 四、验证与复现命令

```sh
# 1. 运行隔离 MySQL 性能基准测试（需提供隔离 DB DSN）
WB_TEST_MYSQL_DSN="<user>:<password>@tcp(<host>:<port>)/<isolated_test_db>?charset=utf8mb4&parseTime=True&loc=Local" \
GOCACHE="$PWD/tmp/gocache" go test -v -tags=integration -count=1 ./tests/integration -run PerformanceBaseline

# 2. 运行列表内存基准测试（无需数据库）
GOCACHE="$PWD/tmp/gocache" go test -tags=integration -run '^$' -bench WorkbenchLists -benchmem ./tests/integration
```
