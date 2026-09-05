# 性能基线与结果等价契约报告 (Phase P0)

- **采集时间**: 2026-09-05 14:00 CST
- **测试环境**: Apple M5 / macOS (Darwin arm64)
- **基线输入 HEAD**: `546caf8de325148ab5e50a32d9630bd4aec8ed44`
- **环境状态**: `WB_TEST_MYSQL_DSN` 未注入（由于禁止直连真实业务库 `zentaopms` 进行写操作，本次建立标准化合成基线 Synthetic Baseline，真实隔离 MySQL 运行测试标记为 DATA REQUIRED）。

---

## 一、性能瓶颈真源与当前行为分析

依据治理计划确证事实：
1. **通知列表 (E31-N)**: `internal/module/po/reponotice.go`
   - `findNoticeRows` 一次性拉取当前用户名下全部历史通知（包括大文本列 `n.data`）。
   - 在应用层内存中连续执行两次 `filterNoticeRows`：一次用于统计全分类计数，一次用于截取 `[start:end]` 分页数据。
   - 随用户历史通知积累，内存分配与延迟呈 $O(N)$ 线性增长。
2. **我的待办 (E31-T)**: `internal/module/po/servicetodo.go` 与 `repotodo.go`
   - `FindTodoItems` 聚合需求、任务、Bug 的所有候选 ID，并在内存中做 7 维过滤。
   - `sortTodoItems` 对全部候选结果在内存中执行全量排序后截断，计算复杂度为 $O(N \log N)$。
3. **版本窗口 (E32)**: `internal/module/schedule/service_window.go`
   - `ListWindowCards` 获取全部窗口列表后，使用 `for` 循环针对每个窗口执行 `CalcCapacity`、`GetWindowConsumedHours`、`GetWindowDemandCount`。
   - 查询数量随窗口数量 $N$ 呈 $1 + 2N$ 线性膨胀（N+1 查询扇出）。

---

## 二、合成基准测试数据与增长检测

通过 `tests/integration/performance_baseline_test.go` 测量得出的当前实现开销基线：

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
# 1. 运行性能基线确定性与隔离测试
GOCACHE="$PWD/tmp/gocache" go test -tags=integration -count=1 ./tests/integration -run PerformanceBaseline

# 2. 运行列表内存基准测试
GOCACHE="$PWD/tmp/gocache" go test -tags=integration -run '^$' -bench WorkbenchLists -benchmem ./tests/integration
```
