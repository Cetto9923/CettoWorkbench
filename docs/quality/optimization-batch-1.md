# 第 1 批变更记录

所有变更位于独立优化分支，没有执行迁移、生产写入或部署。

源码快照验证：`GOPROXY=off go test ./...` 通过（隔离数据库并发测试未运行）。

## 文件净行数

| 文件 | 净行数 |
|---|---:|
| `internal/config/config.go` | 10 |
| `internal/config/loader.go` | 10 |
| `internal/middleware/csrf.go` | 1 |
| `internal/middleware/operationlog.go` | 0 |
| `internal/module/agileteam/repo_basic_atomic_concurrency_test.go` | 3 |
| `internal/module/po/gateway.go` | -77 |
| `internal/module/po/repo_deliver.go` | -96 |
| `internal/module/po/repo_home_actions.go` | -35 |
| `internal/module/po/service.go` | -6 |
| `internal/module/po/service_deliver.go` | 5 |
| `internal/module/po/service_deliver_test.go` | 19 |
| `internal/module/po/service_home_actions.go` | 5 |
| `internal/module/schedule/gateway.go` | 309 |
| `internal/module/schedule/repo.go` | -70 |
| `internal/module/schedule/repo_history.go` | -45 |
| `internal/module/schedule/repo_scheduling_scope.go` | 55 |
| `internal/module/schedule/repo_scheduling_status.go` | -61 |
| `internal/module/schedule/repo_scheduling_write.go` | -154 |
| `internal/module/schedule/repo_window_scope_test.go` | 0 |
| `internal/module/schedule/repodemandwindow.go` | -3 |
| `internal/module/schedule/safety_test.go` | 79 |
| `internal/module/schedule/schedule_window_filter_test.go` | 0 |
| `internal/module/schedule/service.go` | 18 |
| `internal/module/schedule/service_scheduling_save.go` | -33 |
| `internal/module/schedule/service_scheduling_scope.go` | 123 |
| `internal/module/schedule/service_scheduling_tostory.go` | 80 |
| `internal/module/schedule/service_story_history.go` | -59 |
| `internal/module/schedule/service_story_tasks.go` | -44 |
| `internal/module/schedule/service_window_authz_test.go` | 22 |
| `internal/pkg/database/database.go` | 4 |
| `internal/pkg/redact/redact.go` | 59 |
| `internal/pkg/sqllog/sqllog.go` | 6 |
| `internal/pkg/zentao/apilog.go` | -13 |
| `internal/pkg/zentao/client.go` | 14 |
| `internal/pkg/zentao/safety_test.go` | 60 |
| `internal/server/csrf_test.go` | 61 |
| `internal/server/server.go` | 2 |

`tools/zentao-write-allowlist.txt` 删除已替代动作的失效登记，净减 12 行。

## 新增函数 / 方法声明

- `func CSRF(secure bool) func(http.Handler) http.Handler`
- `func (r *Repo) RequireDeliverWindow(ctx context.Context, windowID uint) error`
- `func (r *Repo) SaveDeliverWindow(ctx context.Context, id, windowID uint, account string) error`
- `func createProductPlan(ctx context.Context, client *zentao.Client, productID uint, title, begin, end string) (uint, error)`
- `func assignStory(ctx context.Context, client *zentao.Client, storyID uint, assignedTo string) error`
- `func resolveChangedStoryAssigns(req *SaveSchedulingReq, oldByID map[uint]string) []storyAssignItem`
- `func buildZentaoTaskEditBody(taskReq SaveSchedulingTask) map[string]any`
- `func buildZentaoTaskCreateBody(storyID uint, taskReq SaveSchedulingTask) map[string]any`
- `func zentaoTaskCreatePath(executionID uint) string`
- `func createTask(ctx context.Context, client *zentao.Client, storyID uint, taskReq SaveSchedulingTask) error`
- `func updateTask(ctx context.Context, client *zentao.Client, taskID uint, body map[string]any) error`
- `func deleteSchedulingObjects(ctx context.Context, client *zentao.Client, kind string, ids []uint) error`
- `func collectSchedulingDeletes(req *SaveSchedulingReq) (storyIDs []uint, taskIDs []uint)`
- `func collectNewSchedulingStories(req *SaveSchedulingReq) []SaveSchedulingStory`
- `func formatToStoryEstimate(v float64) string`
- `func formatToStoryPlan(planID uint) string`
- `func buildZentaoToStoryBody(in toStoryBodyInput) map[string]any`
- `func zentaoDemandToStoryPath(demandID uint) string`
- `func toStory(ctx context.Context, client *zentao.Client, demandID uint, in toStoryBodyInput) ([]uint, error)`
- `func (r *Repo) SchedulingStories(ctx context.Context, demandID, storyID uint) ([]ZtStory, error)`
- `func (r *Repo) SchedulingTaskOwners(ctx context.Context, ids []uint) (map[uint]uint, error)`
- `func (r *Repo) ExecutionMatchesProduct(ctx context.Context, executionID, productID uint) (bool, error)`
- `func (r *Repo) LockWindow(ctx context.Context, id uint64) error`
- `func TestSchedulingRejectsForeignStoryBeforeWrites(t *testing.T)`
- `func TestSchedulingRejectsForeignTaskBeforeWrites(t *testing.T)`
- `func TestTaskEditDoesNotSendLeftOrConsumed(t *testing.T)`
- `func (s *Service) prepareWindowProducts(ctx context.Context, actor *model.User, window *model.VersionWindow, products []WindowProductInput, account string) ([]*model.VersionWindowProduct, error)`
- `func (s *Service) SaveStoryScheduling(ctx context.Context, actor *model.User, storyID uint, req *SaveSchedulingReq) error`
- `func (s *Service) saveScheduling(ctx context.Context, actor *model.User, id uint, req *SaveSchedulingReq, independent bool) error`
- `func (s *Service) syncScheduling(ctx context.Context, req schedulingSaveReq) error`
- `func (s *Service) saveSchedulingMetadata(ctx context.Context, req schedulingSaveReq) error`
- `func (s *Service) saveSchedulingStories(ctx context.Context, account string, demandID, mainSystemID uint, launch string, req *SaveSchedulingReq) error`
- `func (s *Service) saveIndependentScheduling(ctx context.Context, account string, storyID, productID uint, launch string, req *SaveSchedulingReq) error`
- `func (s *Service) applyZenTaoDeletes(ctx context.Context, req *SaveSchedulingReq) error`
- `func (s *Service) applyZenTaoAssigns(ctx context.Context, req *SaveSchedulingReq) error`
- `func (s *Service) resolvePlanForProduct(ctx context.Context, repo *Repo, account string, windowID, productID uint) (uint, error)`
- `func (s *Service) validateSchedulingObjects(ctx context.Context, actor *model.User, demandID, storyID uint, req *SaveSchedulingReq) error`
- `func (s *Service) validateSchedulingStory(ctx context.Context, products map[uint]bool, owners map[uint]uint, story SaveSchedulingStory, originalProduct uint) error`
- `func (s *Service) validateSchedulingTask(ctx context.Context, story SaveSchedulingStory, owners map[uint]uint, task SaveSchedulingTask) error`
- `func (s *Service) schedulingProductAccess(ctx context.Context, actor *model.User) (map[uint]bool, error)`
- `func (s *Service) validateWindowProducts(ctx context.Context, actor *model.User, products []WindowProductInput) error`
- `func (s *Service) applyNewStoriesViaToStory(`
- `func (s *Service) saveEditedTask(ctx context.Context, taskReq SaveSchedulingTask) error`
- `func TestDeleteWindowRejectsLinkedWorkItemsBeforeDelete(t *testing.T)`
- `func (l *gormSQLLogger) ParamsFilter(_ context.Context, sql string, _ ...interface`
- `func Sensitive(key string) bool`
- `func JSON(raw []byte) any`
- `func filter(value any)`
- `func SQL(query string) string`
- `func encodeResponseBody(raw []byte) any`
- `func TestAPILogRedactsNestedCredentials(t *testing.T)`
- `func TestNativeWriteHTTP200FailureAndReadNotice(t *testing.T)`
- `func TestProductionSchedulingRouteRequiresCSRF(t *testing.T)`

未添加第三方依赖；没有移动生产目录、压行或生成代码来掩盖增长。已有超过 500 行文件在本批均不得净增长。CSS 选择器变化在最终总报告单列。
