# V2 第二批次实施记录

## 分支与边界

- 起点：公司 Main，commit `a4362f2314362244164e471471980368b98740bd`。
- 分支：`codex/v2-wave2-20260928`；不合并或推送 Main。
- 部署范围：只允许更新 8098；不操作 8090、8095、OB 容器或 ZenTao 源码/自定义目录。
- 用户已确认：从工作分支按需移植前置模块；审批阻塞采用 A4/正式版口径，计数与 `focus=blocked` 列表一致。

## 已完成

- 从 Main 对应基线按需移植工作看板、问题风险、敏捷团队、团队首页及所需基础模块。
- 需求首页阻塞卡使用审批阻塞 KPI，深链进入 `focus=blocked`；KPI 和列表复用同一开发完成日边界。
- 版本窗口支持包含当前用户关联需求的窗口；跳排期时保留窗口、小组及当前阶段参数。
- 工作看板、问题风险和敏捷团队页面读取 URL 筛选状态。
- 团队首页收敛为四张卡，并按部门、组织团队、敏捷小组逐层筛选；子范围由后端重新验证。
- 敏捷团队管理页已有 PMO 组织挂靠选择入口及历史快照存储。
- PRD V0.9 已回写阶段名、评价反馈、默认焦点和阻塞口径。
- 健康阈值、健康后端和健康卡按最新裁剪版移除。

## 仍需处理

- D1 偏好存储：KV 使用独立 `zt_wb_profile_pref_kv` 表，保留旧版 `zt_wb_profile_prefs(account, preferredRoles, createdDate, updatedDate)`；旧自选角色在 KV 尚无记录时回退读取，后续保存写入 KV 表。
- 部署到 8098 并完成登录态验收，留存七张真实截图。

## 合并与冲突记录

- 交接指定分支未能从远端获取；同名本地分支存在于 `/Users/yuyan9923/GitHub/workbench-claude-po`，HEAD `642f87e6d54dbee76e608ae2584a053a2da21ddc`。已将其合并到本分支，未向远端推送。
- 共 25 个冲突：保留本分支的模块注册/团队只读授权、子范围校验、A3 需求关联窗口和四卡三级筛选；敏捷团队组织部门查询按运行库真实 schema 移除不存在的 `zt_dept.deleted`；工作台 UI 保留四卡版本。
- `db/install.sql` 采用工作分支已有安装 SQL（含组织挂靠表），再保留本分支偏好表定义；偏好表兼容仍待确认。
- Go 冲突文件：`internal/bootstrap/bootstrap.go`；`internal/module/agileteam/{handler.go,repo_orgmap.go,repo_scope.go}`；`internal/module/po/{form_issue_risk.go,handler.go,handler_home_team.go,handler_issue_risk.go,home_focus_test.go,home_stage_repo.go,repokpi.go,repovaluestream.go,service.go}`；`internal/module/profile/{handler.go,repo.go}`。
- Web 冲突文件：`web/static/css/agileteam/agileteam.css`；`web/static/js/agileteam/agileteam.js`；`web/static/js/po/{home-team.js,home.js,issue-risk.js,workboard-core.js,workboard.js}`；`web/templates/po/{home.html,home_team.html}`。

## 当前验证

- 合并后 `go build ./...`、`make check`（含全量 `go test ./...` 与 `go vet ./...`）：PASS。
- 前端 15 项 Node 回归、`git diff --check`、团队首页和敏捷团队脚本语法检查：PASS。
- 部署和真实登录态浏览器验收尚未执行。
