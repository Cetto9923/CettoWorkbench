# 规范与开发代码快照提交

日期：2026-09-07。用户在规范体系核对后明确要求：把规范及当前代码推送到 GitHub。
本文件记录本次提交的内容和验收边界；远端提交 SHA 与 CI 结果以提交后的核验为准。

## 提交范围

- 目标：`github` → `Cetto9923/workbench-claude-po`，分支 `Claude-PO`。
- 原本地 HEAD：`a430311a6d46d99a12d821180efbb3b1854d9d39`。
- 已安全快进到远端基线 `6c6341234a22dc157c73304d35f4c9ba3b9df833`，保留两项主题提交。
  远端涉及的 14 个路径与本地 WIP 不重叠；没有 rebase、force push 或覆盖用户文件。
- 包含本地工程规范、各工具入口、审查与任务文档、Go 依赖、个人页面相关代码与测试、
  侧栏角标和 render 拆分文件。它们是当前开发快照，不代表业务能力全部接通。
- `.workbuddy/memory/` 属于本机工具会话上下文，保留在本地；忽略的运行配置、凭据、
  上传附件、二进制和日志不加入提交。
- 发布过程不修业务缺陷，不删除失败测试，不扩大任何 baseline，不操作业务数据库或部署服务。

## 检查结果

应用验收：**BLOCKED BY EXISTING BASELINE**。本快照不作为可发布到运行环境的版本。

- `make check` 失败：`form_done.go` 与 `form.go` 重复声明 DoneTab、TimeRange 等类型。
- 全部 10 个前端 unit/behavior 文件：5 通过、5 失败。
  失败项：home-focus、html-sanitize、navigation-primary-action、notice-filters、priority-helpers。
- 架构门禁失败：新增 Service 文件内的 Repo SQL 等问题仍需处理。
- 文件长度门禁通过；凭据扫描通过，零指纹；补充的 URI/命令行凭据扫描未发现候选。
- 门禁自测 23 项通过。没有把门禁自测当成业务验收。
- 真实登录浏览器、线上性能、真实数据库并发和部署验收未执行。
- GitHub CI 会针对上传提交运行；源码上传成功不等于 CI 通过。

前序 [体系核对](SYSTEM-AND-PUBLICATION.md) 中“未提交”的统计是当时快照，
不能在这次上传后继续作为当前发布状态。规范目录仍以
[spec-index.md](../../engineering/spec-index.md) 为统一导航，默认 main 的旧快照未修改。

下一项开发工作应先恢复编译及失败回归测试，核对权限判断与各页面实际接入，
再完成对应业务、浏览器和共享库验收。不要因本次保存代码就把相关任务标为 COMPLETE。
