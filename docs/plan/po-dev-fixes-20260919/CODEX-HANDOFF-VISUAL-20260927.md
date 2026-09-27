# Codex 交接：workbench integrate 分支视觉验收续作（2026-09-27）

你将在用户 Mac 上继续 workbench PO 工作台的**视觉**优化。请完整阅读本文件后开工，不要向用户提问。

## 1. 环境
- 仓库 worktree：`~/GitHub/workbench-claude-po`，分支 `integrate/main-plus-explore-20260922`，当前 HEAD `71cddc5c`（开工前 `git log --oneline -1` 核对）。
- 另一个 worktree `~/GitHub/workbench` 是生产 main：**禁止修改/提交/切分支**。
- 视觉基线：冻结探索版 **8096**（容器 workbench-dev-ct，镜像 po-night-a0，只读查看，禁止重建/重启）。探索代码参考 `baseline/po-explore-a0cdcfe9`（`git show baseline/po-explore-a0cdcfe9:<path>` 读取，不要 checkout）。`docs/Demo` 原型可能过时，仅宽松参考。
- 部署目标：**8098**（容器 workbench-main-plus-explore-8098）。登录 003030 / 123456。
- 规范：`.cursorrules`、`.cursor/rules/conventions.mdc`（文件头注释块、单文件 ≤500 行、gofmt、RequirePerm 等）。

## 2. 硬性范围
- 只做视觉：CSS / 模板 / 静态资源 / 前端展示 JS。不改业务逻辑、计数口径、权限、状态流、数据语义；发现业务差异写入问题清单。
- 不 push；不碰 main、8090、8096、8097；不改禅道源码；不对共享库做破坏性操作。
- **永远不要执行 `git checkout <sha> --`（不带路径）或 reset --hard / force**；始终停留在分支上，提交前确认 `git branch --show-current`。
- 截图放仓库外（`/tmp/wb-visual/<批次>/`），不要提交截图。

## 3. 构建部署
Docker Desktop 需在运行（`open -a Docker`）。若 OceanBase `obstandalone` 退出：`docker stop obstandalone; docker restart ob-ip-holder; docker start obstandalone`，再按 main 仓库 `docs/plan/rebuild-8090-20260922/RESPONSE.md` 执行 obd cluster start。
脚本 `/tmp/wb_build_deploy.sh`（`TAG=xxx` 可覆盖镜像标签用于未提交预览）；环境变量文件 `/tmp/wb8098.env`（600，从 8096 导出：`docker inspect workbench-dev-ct --format '{{range .Config.Env}}{{println .}}{{end}}' > /tmp/wb8098.env; chmod 600 /tmp/wb8098.env`，勿打印）。若 /tmp 被清空，脚本内容如下：
```bash
#!/bin/bash
# 构建 integrate 当前 HEAD 为 linux/arm64 镜像并在 8098 重建容器（配置同 8096 dev 口径）
set -e
cd ~/GitHub/workbench-claude-po
[ "$(git branch --show-current)" = "integrate/main-plus-explore-20260922" ] || { echo "NOT ON BRANCH"; exit 1; }
SHA=${TAG:-$(git rev-parse --short=8 HEAD)}; ST=/tmp/wb-integrate-build
rm -rf $ST && mkdir -p $ST/dist/workbench/configs
cp -a configs/config.yaml $ST/dist/workbench/configs/ && cp -p configs/config.dev.yaml $ST/dist/workbench/configs/
cp -a web db $ST/dist/workbench/
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $ST/dist/workbench/workbench ./cmd/server
printf 'FROM alpine:3.20\nRUN apk add --no-cache ca-certificates tzdata\nWORKDIR /app\nCOPY dist/workbench/ /app/\nRUN chmod +x /app/workbench && mkdir -p /app/logs\nEXPOSE 8080\nCMD ["./workbench"]\n' > $ST/Dockerfile
(cd $ST && docker build -q -t workbench:main-plus-explore-$SHA . >/dev/null)
docker rm -f workbench-main-plus-explore-8098 >/dev/null 2>&1 || true
docker run -d --name workbench-main-plus-explore-8098 --restart unless-stopped -p 8098:8080 --env-file /tmp/wb8098.env workbench:main-plus-explore-$SHA >/dev/null
for i in $(seq 1 30); do sleep 2; c=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8098/login || true); [ "$c" = "200" ] && break; done
echo "deployed workbench:main-plus-explore-$SHA login=$c"
```
截图脚本 `/tmp/wb_visual.py <port> "<逗号分隔路径或空=全部>" <输出目录>`（Playwright Chromium，1440x900，登录后逐页新标签，输出 JSON：status/consoleErrors/4xx5xx/失败文案）。若丢失，内容如下：
```python
# 浏览器冒烟：登录后逐页打开，记录 console error / 4xx-5xx 请求 / 失败文案，并截图
import sys, json, re
from playwright.sync_api import sync_playwright
PORT = sys.argv[1] if len(sys.argv) > 1 else "8098"
B = f"http://127.0.0.1:{PORT}"
PAGES = ["/home","/todos","/done","/notice","/follow","/schedule","/query","/board/demand","/board/task",
         "/agileteam","/issues/risk","/metrics/radar","/metrics/manage","/profile","/admin/roles","/admin/users",
         "/kanban/story","/kanban/task","/demands/63425"]
if len(sys.argv) > 2 and sys.argv[2]: PAGES = sys.argv[2].split(",")
OUT = sys.argv[3] if len(sys.argv) > 3 else "/tmp/wb-visual/x"
import os; os.makedirs(OUT, exist_ok=True)
FAIL_TEXT = re.compile(r"加载失败|请求失败|服务异常|出错了|template:|<no value>|undefined|NaN")
res = []
with sync_playwright() as p:
    br = p.chromium.launch()
    ctx = br.new_context(viewport={"width": 1440, "height": 900}, locale="zh-CN")
    pg = ctx.new_page()
    pg.goto(B + "/login")
    pg.fill("input[name=account]", "003030")
    pg.fill("input[name=password]", "123456")
    pg.click("button[type=submit]")
    pg.wait_for_load_state("networkidle")
    for path in PAGES:
        cons, bad = [], []
        pg = ctx.new_page()
        pg.on("console", lambda m, c=cons: c.append(m.text[:200]) if m.type == "error" else None)
        pg.on("response", lambda r, b=bad: b.append(f"{r.status} {r.url.replace(B,'')[:120]}") if r.status >= 400 else None)
        pg.on("pageerror", lambda e, c=cons: c.append("PAGEERROR " + str(e)[:200]))
        try:
            resp = pg.goto(B + path, wait_until="networkidle", timeout=45000)
            status = resp.status if resp else 0
        except Exception as e:
            status = "timeout"
        pg.wait_for_timeout(1500)
        body = pg.inner_text("main") if pg.query_selector("main") else pg.inner_text("body")
        hits = sorted(set(FAIL_TEXT.findall(body)))
        name = path.strip("/").replace("/", "_") or "root"
        pg.screenshot(path=f"{OUT}/{name}.png", full_page=False)
        res.append({"path": path, "status": status, "consoleErrors": cons[:5], "bad": bad[:8], "failText": hits, "textLen": len(body)})
        pg.close()
    br.close()
for r in res:
    print(json.dumps(r, ensure_ascii=False))
```

## 4. 剩余任务（按优先级）
1. 排期页「独立研发需求」表：同业务需求表处理（取消 min-width:1180 与标题列内联宽度，写在 `web/static/css/po/schedule-shell.css`，仅 `body.po-workbench-shell` 下生效）。
2. /profile、/admin/*、/kanban/*：仍是 Main 后台外壳。先**不要**改路由归属（需用户决定），只修明显错位（品牌区/侧栏收起按钮与顶栏重叠等）。
3. 人员选择器 user 模式（头像/部门/拼音）：参考探索 `web/static/js/ui.js` 的 mode:"user" 与 `autocomplete.css` 的 `--user` 样式，在不破坏 Main autocomplete 行为的前提下增量扩展 `window.initUserPicker`。
4. 逐页像素级比对 8096（首页、待办、已办、通知、关注、查询、看板、敏捷小组、问题风险、指标雷达/管理、需求详情）：间距、字号、空态、弹窗/抽屉打开态（需点击打开后截图）。
5. 弹窗与抽屉打开态巡检：澄清、提交评审、评审、提测、交付、验收、催办、看板任务抽屉——检查按钮契约（button-system.css）是否把小按钮撑大（修法：给按钮加 `action-btn--sm` 或提高局部选择器优先级，不要改 button-system 全局）。
6. `/kanban/story` 标题出现 `&quot;`：确认是模板二次转义还是数据本身，若为展示层问题再修。

## 5. 验收标准
- 每批：`go build ./... && go vet ./... && go test -count=1 ./...` 全绿；`for f in tests/unit/frontend/*.test.js; do node $f; done` 全 PASS。
- 每批单独提交（中文 message，说明原因与影响范围），提交后 `bash /tmp/wb_build_deploy.sh` 部署 8098。
- 每页截图与 8096 同页对比（8096 用同一脚本 `python3 /tmp/wb_visual.py 8096 "" /tmp/wb-visual/ref-8096`）；19 页 HTTP 200、console 错误 0、静态资源 404 为 0。
- 在 `docs/plan/po-dev-fixes-20260919/INTEGRATE-ROUND-20260927.md` 第 8 节视觉问题表追加/更新状态（页面/问题/严重度/原因/状态/依据）。
- 最终汇报（中文）：HEAD、提交列表、前后截图路径、已修/未修表、需用户决定的问题。

## 6. 背景（已完成，勿重复）
见 INTEGRATE-ROUND-20260927.md：main 合入（fb049683）、功能补齐（至 f5017b70）、视觉四批（37f9761a、13ab5be7、07dfbee0、71cddc5c）、业务问题清单 1–13。
