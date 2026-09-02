# AGENTS.md · Cross-tool entry (Codex CLI / Trae IDE)

> 本文件**仅作为 Codex CLI 与 Trae IDE 的入口**,把工具引导到禅道官方真源。
> **本文件不复述任何规则** —— 所有规则都在禅道官方主源里。
> 本文件**不包含任何项目级业务规范**(PO 视图 / 9 阶段 / 部署真源等),那些不在禅道主源上。

---

## 1. 禅道官方真源(必须读完)

按优先级加载:

1. **`.cursorrules`** ← 禅道官方核心底线(13 条)
2. **`.cursor/rules/conventions.mdc`** ← 禅道官方 Go 开发完整规范(20 章)
3. **`README.md`** ← 禅道项目说明
4. **`docs/demo-analysis.md`** ← 禅道示例分析
5. **`docs/design/schedule-biz-demands.md`** ← 禅道业务设计文档

## 2. 工具加载指引

| 工具 | 加载入口 |
|---|---|
| **Cursor** | 直接读 `.cursorrules` + `.cursor/rules/*.mdc`(原生支持) |
| **Codex CLI** | 读本文件 `AGENTS.md`(自动扫描 git root) |
| **Claude Code** | 无原生入口;如果使用,请参考本文件 §1 自行加载禅道主源 |
| **Trae IDE** | 自动复用根目录 `AGENTS.md`(本文件) |

## 3. 工作流

1. 起手前读完 §1 的禅道官方真源。
2. 写代码中严格按 `.cursor/rules/conventions.mdc` §18 新模块 Checklist 走。
3. 交付前自检是否违反 `.cursorrules` 13 条底线。

---

**项目原则**:本仓库严格遵守禅道 goframework 官方规范,不引入项目级业务规则文件以避免多源冲突。
