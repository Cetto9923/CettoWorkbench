# 审查证据说明

对应 [审查与修复计划](/Users/yuyan9923/GitHub/workbench-claude-po/docs/plan/architecture-review-20260906/AUDIT-AND-REPAIR-PLAN.md)。2026-09-06 开始，2026-09-07 形成报告。

## 当前工作树与范围

- 仓库：`/Users/yuyan9923/GitHub/workbench-claude-po`
- 分支：`Claude-PO`
- HEAD：`a430311a6d46d99a12d821180efbb3b1854d9d39`
- 审查前：38 项已跟踪修改，24 项未跟踪状态项。
- 报告新增目录：`/Users/yuyan9923/GitHub/workbench-claude-po/docs/plan/architecture-review-20260906/`
- 当前工作树含大量 WIP；上述 HEAD 不能单独重建本次全部审查输入。

## 自动检查

运行目录为仓库根目录。命令：

```sh
git diff --check
make check
GOCACHE="$PWD/tmp/gocache" go test -count=1 ./...
node tests/unit/frontend/home-focus.test.js
node tests/unit/frontend/notice-filters.test.js
node tests/e2e/theme-tokens.spec.js
```

另外对 `web/static/js/po` 和 `web/static/js/schedule` 共 24 个 JS 执行 `node --check`，全部通过。

`make check` 结果摘要：

```text
go vet regression gate passed (existing debt: 0 diagnostic(s))
whitespace gate passed
file-length regression gate passed (existing debt: 18 file(s) over 500 lines)
advisory pattern scan completed (64 finding(s), 34 new)
hard-pattern non-growth gate passed (0 existing finding(s))
secret regression gate passed (existing debt: 0 fingerprint(s); values suppressed)
architecture boundary regression gate passed (existing debt: 1 file(s))
required regression gates passed; inspect the printed existing-debt counts
```

本机完整临时日志（不属于仓库版本管理，可能被系统清理）：

- [统一门禁日志](/tmp/workbench-architecture-audit-20260906-check.log)
- [收尾重跑门禁日志](/tmp/workbench-architecture-audit-20260906-final-check.log)
- [无缓存 Go 测试日志](/tmp/workbench-architecture-audit-20260906-go-test.log)
- [隔离探针日志](/tmp/workbench-architecture-audit-20260906/probe-results.log)
- [审查源码 SHA-256 清单](/tmp/workbench-architecture-audit-20260906/source-hashes.json)

收尾核对发现看板 CSS 和 workboard.js 在审查过程中发生了本审查未执行的外部修改。未覆盖这些修改；检查其差异后重新运行 `make check`、workboard.js 语法检查和 `git diff --check`，仍全部通过，门禁统计不变。其它 345 个清单内源文件的哈希与捕获时一致。源码哈希清单记录的是捕获时的审查输入，不是收尾后的全量快照。

## 隔离 Go 探针

通过 `go test -overlay` 临时增加 package-local 测试文件；只使用 sqlmock、合成账号和合成记录。没有写真实数据库，没有在源码目录增加测试文件。源码仍调用本次工作树中的真实 Service、Repo 和组装函数。

本机复现：

```sh
GOCACHE="$PWD/tmp/gocache" go test \
  -overlay=/tmp/workbench-architecture-audit-20260906/overlay.json \
  -run TestAuditProbe -count=1 -v ./internal/module/po
```

探针源文件：[audit_probe_test.go](/tmp/workbench-architecture-audit-20260906/audit_probe_test.go)。它们断言当前缺陷行为，不是将来应保留的产品行为，不应原样加入正式回归套件。

| 探针 | 输入/操作 | 实际结果 |
|---|---|---|
| DetailIgnoresActorAndChildFailure | 主记录成功，子需求查询返回合成错误；nil actor 调用详情。后续未设期望的 SQL 也返回 mock 错误。 | 仍返回 `Success=true`，没有身份拒绝；后续错误同样被忽略。 |
| DoneReadsOtherActor | `audit_requester` 调用动作详情，数据库动作 actor 为 `audit_other`。 | 返回另一账号动作；全部 mock 期望满足，没有额外授权查询。 |
| QualityAndStage | 仅一个合成故事构建质量树；分别输入 waitdeliver/released/closed 构建阶段条。 | 无扫描来源仍通过且 92.5 分；三个状态当前阶段都为 clarify。 |

三个探针全部 PASS 的含义是“成功复现上述不正确行为”，不代表系统通过安全验收。nil actor 是 Service 边界验证；HTTP 路由仍有登录和能力中间件，不能将它描述成匿名远程读取已被证实。

## 富文本输出探针

运行真实 `DemandDetailRender.renderTabRequirement`，使用合成事件属性，无真实用户数据：

```javascript
const assert = require('node:assert/strict');
global.window = global;
const R = require('./web/static/js/po/demand-detail-render.js');
const probe = '<img src="/audit-missing" onerror="window.__auditMarker=1">';
const html = R.renderTabRequirement({specHtml: probe, verifyHtml: probe});
assert.ok(html.includes(probe));
```

结果：可执行事件属性原样保留。结合控制器 `innerHTML` 与当前 CSP，确认了未净化输出路径。没有在浏览器中执行此载荷，也没有声称现存业务数据包含攻击内容。

## 未执行边界

- 没有配置 `WB_TEST_MYSQL_DSN`；未运行带 integration tag 的真实 MySQL 用例，也未访问业务库替代隔离库。
- 没有执行真实账号跨用户访问、浏览器旅程、主题视觉验收或运行时 SQL 计数。
- 没有触发、查询或宣称当前 GitHub Actions 已通过。仅阅读 workflow 定义；本地绿灯和历史截图不是当前远端/浏览器证据。
- 未验证实际 DBA GRANT、唯一索引及部署账户权限。
