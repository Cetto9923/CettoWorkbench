# 第 2 批变更记录

所有变更位于独立优化分支，没有执行迁移、生产写入或部署。

源码快照验证：`GOPROXY=off go test ./...` 通过（隔离数据库并发测试未运行）。

## 文件净行数

| 文件 | 净行数 |
|---|---:|
| `db/install.sql` | 54 |
| `db/upgrade_workbench_safety.sql` | 71 |
| `docs/quality/workbench-write-exceptions.md` | 56 |
| `internal/bootstrap/bootstrap.go` | -13 |
| `internal/module/agileteam/repo_basic_atomic.go` | -16 |
| `internal/module/agileteam/repo_basic_atomic_test.go` | 15 |
| `internal/module/agileteam/repo_confirm.go` | 8 |
| `internal/module/agileteam/service_adjustment_test.go` | 4 |
| `internal/module/agileteam/service_basic_test.go` | -8 |
| `internal/module/metrics/repo.go` | 7 |
| `internal/module/po/home_stage_repo.go` | -24 |
| `internal/module/po/owner.go` | -17 |
| `internal/module/po/repo_account_display.go` | -25 |
| `internal/module/po/sidebar_badges.go` | -3 |
| `internal/module/profile/atomic_test.go` | 41 |
| `internal/module/profile/repo.go` | -30 |
| `internal/module/profile/service.go` | -21 |
| `internal/module/schedule/repo.go` | -51 |
| `internal/module/schedule/repo_scheduling.go` | 23 |
| `internal/module/schedule/repo_scheduling_date.go` | -8 |
| `internal/module/schedule/repo_scheduling_write.go` | -1 |
| `internal/module/schedule/repo_window_scope_test.go` | 2 |
| `internal/module/schedule/repo_window_stats.go` | 74 |
| `internal/module/schedule/safety_test.go` | 20 |
| `internal/module/schedule/service_capacity.go` | 48 |
| `internal/module/schedule/service_scheduling.go` | -9 |
| `internal/module/schedule/service_window.go` | -15 |
| `internal/pkg/database/database.go` | -3 |
| `internal/pkg/database/safety_test.go` | 106 |
| `internal/pkg/database/schema.go` | 51 |
| `internal/pkg/render/render.go` | 2 |
| `internal/pkg/render/sidebar_badges.go` | 1 |
| `internal/pkg/sqllog/sqllog.go` | 2 |
| `tests/unit/quality/zentao-writes.test.py` | 26 |
| `tools/check-zentao-write.sh` | 16 |
| `tools/zentao-write-allowlist.txt` | 10 |

## 新增函数 / 方法声明

- `func updateTeamgroupFields(tx *gorm.DB, table string, id uint, name, slogan, declaration, logo string, history *History) error`
- `func teamgroupRowError(err error) error`
- `func expectTeamgroupObserved(mock sqlmock.Sqlmock, id, parent uint)`
- `func expectBasicTargetLock(mock sqlmock.Sqlmock, id, parent uint)`
- `func TestBasicNoChangeSucceedsWithoutHistory(t *testing.T)`
- `func (s *Service) SidebarBadges(ctx context.Context, actor *model.User) (SidebarBadges, error)`
- `func TestProfilePreferenceFailureRollsBackContact(t *testing.T)`
- `func (r *Repo) SaveSelfProfile(ctx context.Context, actor *model.User, req UpdateReq, preferred []string) error`
- `func (r *Repo) GetStoriesTasks(ctx context.Context, storyIDs []uint) (map[uint][]ZtTaskItem, error)`
- `func (r *Repo) GetProductsProjects(ctx context.Context, productIDs []uint) (map[uint][]ZtProjectOption, error)`
- `func (r *Repo) GetProjectsExecutions(ctx context.Context, projectIDs []uint) (map[uint][]ZtExecutionOption, error)`
- `func (r *Repo) projectOptions(ctx context.Context, query string, ids []uint) ([]projectOptionRow, error)`
- `func (r *Repo) windowsStats(ctx context.Context, ids []uint64) (map[uint64]windowStats, error)`
- `func TestWindowStatsUsesThreeQueriesForMultipleWindows(t *testing.T)`
- `func countCalendarDays(start, end time.Time, holidaySet, workingSet map[string]struct`
- `func (s *Service) windowCapacities(ctx context.Context, windows []model.VersionWindow) (map[uint64]int, error)`
- `func windowStart(w model.VersionWindow) time.Time`
- `func windowIDs(windows []model.VersionWindow) []uint64`
- `func orderedDates(start, end time.Time) (time.Time, time.Time)`
- `func TestSQLLoggerLeavesSensitiveParametersOut(t *testing.T)`
- `func TestSchemaCheckUsesInstallColumnsWithoutDDL(t *testing.T)`
- `func installColumnRows(source []byte, missing string) *sqlmock.Rows`
- `func assertInstalledSchema(t *testing.T, missing string)`
- `func CheckSchema(db *gorm.DB) error`

未添加第三方依赖；没有移动生产目录、压行或生成代码来掩盖增长。已有超过 500 行文件在本批均不得净增长。CSS 选择器变化在最终总报告单列。
