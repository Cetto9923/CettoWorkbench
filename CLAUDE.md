# CLAUDE.md · Claude Code entry (pure pointer, no business rules)

> 本文件**仅作为 Claude Code 的入口**,把工具引导到禅道官方真源。
> **本文件不复述任何规则** —— 所有规则都在禅道官方主源里。
> 本文件**不包含任何项目级业务规范**,只列指针。

---

## 0. 强制导入(Claude Code 不原生读 AGENTS.md)

Claude Code 截至 2026-07 仍**不自动读取 `AGENTS.md`**。下面的 `@` 用 Claude 的 @-import 语法把禅道主源强制拉进上下文 —— 不写这一行,Claude 就读不到禅道规则。

```markdown
@.cursorrules
@.cursor/rules/conventions.mdc
@README.md
@AGENTS.md
```

## 1. 禅道官方真源(必须读完)

按优先级加载:

1. **`.cursorrules`** ← 禅道官方核心底线(13 条)
2. **`.cursor/rules/conventions.mdc`** ← 禅道官方 Go 开发完整规范(20 章)
3. **`README.md`** ← 禅道项目说明
4. **`AGENTS.md`** ← Codex / Trae 入口(与本文件对称)

## 2. 工具加载指引

| 工具 | 加载入口 |
|---|---|
| **Cursor** | 直接读 `.cursorrules` + `.cursor/rules/*.mdc`(原生支持) |
| **Codex CLI** | 读 `AGENTS.md`(自动扫描 git root) |
| **Claude Code** | 读本文件 `CLAUDE.md`(本仓库的 Claude 入口) |
| **Trae IDE** | 自动复用根目录 `AGENTS.md`(同 Cursor / Codex) |

## 3. 工作流

1. 起手前读完 §1 的禅道官方真源(`@`-import 已自动加载)。
2. 写代码中严格按 `.cursor/rules/conventions.mdc` §18 新模块 Checklist 走。
3. 交付前自检是否违反 `.cursorrules` 13 条底线。

---

**项目原则**:本仓库严格遵守禅道 goframework 官方规范,不引入项目级业务规则文件以避免多源冲突。
