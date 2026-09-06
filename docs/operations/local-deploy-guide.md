# 本地化部署手册 (workbench-claude-po @ :8093)

> 本文件不进入 git（`.gitignore` 已加 `/docs/operations/`），仅供本机开发者按手册
> 重建 8093 部署。业务变更请走 `docs/plan/` + PR；本文件不构成工程真源。

适用：Mac (darwin) + Homebrew + Go (`go.mod` 中指定版本) + 现有 SSH 别名 `vm-zentao`。

---

## 0. 一次性环境

| 项 | 值 |
|---|---|
| 仓库根 | `/Users/yuyan9923/GitHub/workbench-claude-po` |
| 工作分支 | `Claude-PO` |
| SSH 别名 | `vm-zentao` → `10.211.55.4`（用户 `parallels`，key `~/.ssh/id_ed25519`，见 `~/.ssh/config`） |
| DB 账号 | `zentao / zentao123`（来自 `configs/config.dev.yaml`，**该文件已 `.gitignore`**） |
| 监听端口 | `:8093`（dev 配置） |
| 启动日志 | `/tmp/workbench_run.log` |
| 二进制路径 | `./tmp/workbench_dev` |

---

## 1. 数据库准备（两条路径，任选）

### 1.1 推荐：SSH 隧道到 PD VM（vm-zentao）

VM 上的 MySQL **只监听 127.0.0.1**，从 Mac 无法直连。需要打通本地回环到 VM 的隧道。

```bash
# 后台建立隧道：Mac 13306 → VM 127.0.0.1:3306
ssh -fN -o StrictHostKeyChecking=accept-new -o ServerAliveInterval=30 \
    -L 13306:127.0.0.1:3306 vm-zentao

# 验证隧道
lsof -i :13306 -sTCP:LISTEN | head -3
mysql -h 127.0.0.1 -P 13306 -uzentao -pzentao123 \
    -e "USE zentaopms; SHOW TABLES" | head
```

隧道失效后用同样命令重建。停止：`pkill -f "ssh.*13306:127.0.0.1:3306"`。

### 1.2 本地空 schema（不能用于 UI 验收，仅用于冒烟启动）

```bash
brew services start mysql
mysql -uroot <<'SQL'
CREATE DATABASE IF NOT EXISTS zentaopms CHARACTER SET utf8mb4;
CREATE USER IF NOT EXISTS 'zentao'@'%' IDENTIFIED BY 'zentao123';
GRANT ALL ON zentaopms.* TO 'zentao'@'%';
FLUSH PRIVILEGES;
SQL
```

此种情况下服务能起，但 4 页所有数据接口 500，不可作为验收环境。

---

## 2. 同步本地代码（与 origin/Claude-PO 对齐）

```bash
cd /Users/yuyan9923/GitHub/workbench-claude-po
git fetch origin
git status   # 确认 ahead/behind 都是 0 才继续
```

如果出现 uncommitted 工作树改动，**先 stash**（不要丢弃），pull 成功后再 `git stash pop` 逐文件解冲突：

```bash
git stash push -m "WIP before deploy"
git pull origin Claude-PO
git stash pop
```

---

## 3. 编译

```bash
go build -o ./tmp/workbench_dev ./cmd/server
```

产物覆盖现有 `tmp/workbench_dev`。如要确保源干净可先 `make check`（耗时较久，含 `go test ./...`）。

---

## 4. 启动服务

### 4.1 SSH 隧道路径（与 1.1 配套）

```bash
WORKBENCH_MODE=dev \
WORKBENCH_DATABASE_PORT=13306 \
WORKBENCH_DATABASEREADONLY_PORT=13306 \
  ./tmp/workbench_dev > /tmp/workbench_run.log 2>&1 &
```

`WORKBENCH_MODE=dev` → 加载 `configs/config.dev.yaml`；env 变量覆盖 DB 端口。

### 4.2 本地直连路径（与 1.2 配套）

```bash
WORKBENCH_MODE=dev ./tmp/workbench_dev > /tmp/workbench_run.log 2>&1 &
```

### 4.3 启动成功标志

启动后 3–5 秒内：

- `/tmp/workbench_run.log` 末尾出现 `Listening and serving HTTP on :8093`（Gin 模式）
- `lsof -i :8093 -sTCP:LISTEN` 有 `workbench_dev` 进程

---

## 5. 健康探测

```bash
for p in /login /home /todos /done /notice; do
    printf "%s %s\n" "$p" "$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:8093${p}")"
done
```

预期：

| 路径 | 期望码 |
|---|---|
| `/login` | 200 |
| `/home` `/todos` `/done` `/notice` | 303（未登录重定向到 login） |

如果 `/login` 不是 200，看 `/tmp/workbench_run.log` 末尾。

---

## 6. 浏览器验收

打开 `http://127.0.0.1:8093/login`，用真实禅道账号登录（截图原账号 `003030`）。

四页验收路径：

- `/home` —— 首页价值流、行动列表、版本窗口、列表获取时间
- `/todos` —— 5 概览、未接入分类（带「未接入」角标并 disabled）、关系分段
- `/done` —— 累计处理 + 时间范围、全部已办/审批决策/...
- `/notice` —— 关联信息缺失、邮件通知、全部标为已读（带「非仅当前筛选」提示）

---

## 7. 停止 / 重启

```bash
# 停止服务
pkill -f workbench_dev

# 重启（不重建）
pkill -f workbench_dev
WORKBENCH_MODE=dev WORKBENCH_DATABASE_PORT=13306 WORKBENCH_DATABASEREADONLY_PORT=13306 \
  ./tmp/workbench_dev > /tmp/workbench_run.log 2>&1 &

# 重启（重建）
pkill -f workbench_dev
go build -o ./tmp/workbench_dev ./cmd/server
# 然后用上面的启动命令
```

---

## 8. 故障排查

| 症状 | 排查路径 |
|---|---|
| `Connection refused` 127.0.0.1:3306 | MySQL 没启；`brew services start mysql` 或走 SSH 隧道 |
| 隧道不通 | `lsof -i :13306`；重建 `ssh -fN -L 13306:127.0.0.1:3306 vm-zentao` |
| 启动后立刻退出 | `tail -20 /tmp/workbench_run.log`，看 init database / config 错误 |
| `/login` 500 | DB 可达但 schema 缺表；走 1.2 路径不行，必须 1.1 |
| 已登录但页面 500 | `/tmp/workbench_run.log` 末尾看具体栈；多为 SQL 权限或 `zt_notify` 缺列 |
| CSRF 403 | 别手动改 cookie；用浏览器原生表单提交 |
| `make check` 新增 advisory 失败 | 看是否改了 `internal/module/po` 之外的符号；advisory 不阻塞，但 `git diff --check` 与 hard pattern 必须清 |

---

## 9. 不变量（部署必须遵守）

- **不修改** `configs/config.dev.yaml`（即便密码变更也只改本机文件，不入库）
- **不 git commit / push** 本部署相关的运行产物（`tmp/workbench_dev` 在 `.gitignore`）
- **不写入真实业务数据到本地**（通知已读、标跟进等操作只在验收时点真实 VM 上的数据）
- **保留** 其他任务的 WIP（`web/static/js/po/workboard.js`、`web/templates/po/workboard.html`、`web/static/css/po/board.css`），**不纳入本次部署改动**
- 不修改 `AGENTS.md` / `.cursorrules` / 全局 Rules
- 不修改 CI、`Makefile`、baseline 文件

---

## 10. 相关路径速查

| 用途 | 路径 |
|---|---|
| 仓库根 | `/Users/yuyan9923/GitHub/workbench-claude-po` |
| 入口 main | `cmd/server/main.go` |
| dev 配置（git 忽略） | `configs/config.dev.yaml` |
| prod 配置 | `configs/config.yaml` |
| 前端共享样式 | `web/static/css/po/personal-workspace.css` |
| 前端共享 JS | `web/static/js/po/personal-list.js` |
| 四页模板 | `web/templates/po/{home,todos,done,notice}.html` |
| 后端 PO 模块 | `internal/module/po/{handler,service}.go` |
| 启动日志 | `/tmp/workbench_run.log` |
| 服务日志目录 | `logs/`（git 忽略） |
| 工程真源 | `AGENTS.md` → `docs/engineering/*.md` → `.cursor/rules/*.mdc` |