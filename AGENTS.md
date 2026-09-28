# 仓库 AI 主规则

本文件是 Codex、Antigravity、Cursor 共用的唯一主规则。开始任务前须完整阅读。
工具入口和规则副本不得另立标准；前端规则在本文件、`.agents/rules/frontend.md`、`.cursor/rules/conventions.mdc` 三处同步。
保留的旧条目与本文件新增规则冲突时，以用户确认的新增规则为准；其他冲突须停下询问用户。
既有超长代码和规则文档的行数约束按下方 f 条执行；其余代码仍遵守原 500 行硬线。

## 原始主规则（原文保留）

来源：`/Users/yuyan9923/GitHub/workbench/AGENTS.md`，于 2026-09-28 只读复制。
以下来源正文原样保留；其中“本文件”指本 AGENTS.md。

## 项目类型
前后端不分离的 Go 单体 Web 项目（SSR）。URL 遵循 RESTful 风格，写操作（POST/PUT/DELETE）统一使用 ajax 请求，后端返回 JSON，前端根据 `redirectUrl` 跳转。ajax 请求直接使用真实 HTTP Method（`PUT` / `DELETE`）

## 技术栈
- Web: Gin
- ORM: GORM + MySQL
- 数据库: MySQL 8.0+
- 前端: HTML + CSS + JS
---

## 核心底线（违反即为 bug）

1. **所有路由（包括 List）必须绑定 `middleware.RequirePerm(perm.Xxx)`**
   - 权限粒度：`XxxList` / `XxxCreate` / `XxxUpdate` / `XxxDelete`
2. **所有 POST/PUT/DELETE 成功后返回 JSON**，格式为 `{ success: true, message: "xxx", redirectUrl: "/xxx" }`，前端根据 `redirectUrl` 跳转
3. **Service 写操作必须接收 `actor *model.User`**
4. **Service 方法用 Req/Resp 结构体，禁止多位置参数**（单个 id int64 除外，见 §6）
5. **SQL 必须参数化(复杂报表除外)，禁止 `db.Raw()` 和 `db.Exec()` 拼接字符串**
   - 复杂报表 JOIN / 子查询，必须使用 `?` 占位符，禁止字符串拼接，SQL 只能写在 Repo 层
6. **单文件 ≤500 行**（硬线），建议 300 行；超限按职责拆分（见 §17.1）
7. **禁止 `c.PostForm()` / `c.Query()` 散落，强制结构体绑定**
   - Path 参数（`:id`）通过 `c.Param("id")` 获取
   - Query 参数（GET）：`c.ShouldBindQuery(&req)` 或带 `form` tag 的 `c.ShouldBind(&req)`；Req 使用 `form` tag
   - JSON Body（POST/PUT，ajax）：`c.ShouldBindJSON(&req)`；Req 使用 `json` tag
   - 写入类 Req（`CreateReq`/`UpdateReq`）只用 `json` tag；读取类 Req（`ListReq`）只用 `form` tag
   - `DeleteReq.ID` 由 Handler 从 `c.Param("id")` 手动赋值
8. **文件头注释块必须存在**（格式见 `@.cursor/rules/conventions.mdc` §4）
9. **Repo `FindAll` 返回 `([]Xxx, int64, error)` 三元组**
10. **模板数据 key 固定：`Form` / `Resource`（编辑页）/ `BaseUrl`（列表页+编辑页）**
    - `Errors` 不经服务端注入模板；验证失败走 ajax 422 JSON，由前端 `renderErrors` 渲染
    - 列表页禁止 `Resource`，业务数据用具体名（如 `Users` / `Products`）
    - 禁止具体实体名（`.User`、`.Customer`）替代 `Resource`
11. **表单验证统一手写 `Validate() []FieldError` 方法，禁止引入 `go-playground/validator`**
13. **除弹窗外的所有页面均在当前页打开**，禁止打开新窗口`
14. 在 `db/install.sql` 新增建表 SQL，字段名小驼峰，含 `deletedAt` 、`createdBy` 、`createdDate` 、`updatedBy` 、`updatedDate` 字段，表名前缀为zt_
---

## 命名规则
- **文件名/目录名**：全小写拼接
  - 例外：静态资源文件（字体、第三方库、图片）保留原始命名
- **Go 标识符**：未导出小驼峰，导出大驼峰

---

## 模块类型
- **crud**: List → CreatePage → Create → EditPage → Update → Delete（参照 `@internal/module/user/`）
- **readonly**: 只有 List（参照 `@internal/module/operationlog/`）
- **action**: 自定义方法（参照 `@internal/module/login/`）

Handler / Service / Repo 方法顺序固定，结构对称是硬约束。

---

## 新模块开发顺序
1. **form.go**：Req/Resp 结构体 + `Validate()` 方法
2. **model.go + install.sql**
3. **repo.go / service.go / 真实 handler.go**
4. **routes.go + bootstrap.go 注册**

---

## 规范优先级（冲突时以高优先级为准）
1. **核心底线**（本文件，必须）
2. **模块 golden reference**（`@internal/module/user/`）

## 规范索引
- **Go 开发完整规范** → `@.cursor/rules/conventions.mdc`

## 工作树既有补充（原文保留）

来源：本工作树 `.cursorrules`，以下条目继续有效。

15.所有表需要在 model 中实体化
16.所有下拉都为检索下拉
17.所有日期的格式为 y-m-d

### 其他注意事项
1.所有 sql 写法需要兼容oceanbase
2.用户的显示，可以使用 module/user 的AccountDisplayMap方法

## 前端与 UI 规则（硬性）

a. 颜色只能引用已定义的语义变量：`web/static/css/layout/variables.css` 的 `--color-*`；`web/static/css/tokens.css` 的 `--wb-*` 仅限已定义的。禁止新建第三套变量体系。禁止在页面 CSS 写十六进制、rgb、white 等字面颜色；仅上述两个文件的变量定义处例外。
b. 新增任何颜色变量，必须同时在浅色块和 `html[data-theme="dark"]` 块给值。提交前运行下方扫描，“被引用但未定义”的变量必须为 0；扫描不替代双主题覆盖检查。
c. 主题逻辑只有 `web/static/js/layout/theme.js` 一份，存储键为 `workbench.theme.preference`。禁止在其他 JS 重复实现主题切换或重复绑定 `#themeQuickToggle`；使用已有 `WorkbenchTheme` API。
d. 禁止新增 `!important`。禁止新增内联 `style=`，包括 JS 拼接的 style 字符串。隐藏元素使用 `hidden` 属性或已有 class。
e. 新写的公共组件样式必须当轮被页面使用，禁止预先堆放“备用”样式类。
f. 已经超过 500 行的代码文件，每一轮的净增行数不能超过 0，新功能一律放到新文件里。如果某个修复确实必须让这类文件变长，就先停下来问用户，不要自己决定。规则文档不受这一条限制，但 AGENTS.md 本身不超过 400 行。
g. 不准新增 `window.*` 全局函数；公共工具函数放已有模块。
h. 只改任务点名的页面或文件。修改 `web/templates/layout/base.html`、`web/templates/layout/header.html`、`web/static/js/app.js`、`web/static/js/layout/theme.js` 等全局文件前，必须在报告里写明理由；未获授权的范围不得扩展。禁止“顺手优化”、擅自改配色和布局。
i. 本期只交付“需求管理”视角，角色称为“产品经理”。“团队管理”视角及入口顺延二期，代码保留、默认关闭，任何页面不得出现其入口。不使用“PO/产品负责人”称谓。
j. 多页面改造必须每改完 1 页停下交截图，等用户确认再继续。任务写了“停下”就必须停。
k. 每个改动页面在浅色、深色各截一张全页图；页面内部滚动区域须截到底，等数据加载完再截。自己逐张检查白块、看不清的字、截断溢出、错位。截图必须有真实数据；数据库本来为空时注明。只报“接口通过/测试通过”不算完成。
l. 截图、走查脚本、临时文件不得提交进 git。

### 未定义颜色变量扫描

在仓库根目录执行；统计 `web/static`、`web/templates` 中 CSS/JS/HTML 的 `--wb-*`、`--color-*` 引用，与源 CSS 定义比较。忽略块注释；排除构建产物、依赖和第三方目录的定义，避免旧产物掩盖漏定义。输出变量名和引用次数；有缺失时退出码为 1。

```sh
python3 - <<'PY'
from pathlib import Path
from collections import Counter
import re
import sys

used = Counter()
defined = set()
for root in (Path("web/static"), Path("web/templates")):
    for path in sorted(root.rglob("*")):
        if not path.is_file() or path.suffix not in {".css", ".js", ".html"}:
            continue
        text = path.read_text(encoding="utf-8")
        text = re.sub(r"/\*.*?\*/|<!--.*?-->|\{\{/\*.*?\*/\}\}", "", text, flags=re.S)
        used.update(re.findall(r"var\(\s*(--(?:wb|color)-[\w-]+)", text))
for path in sorted(Path(".").rglob("*.css")):
    if any(part in {".git", "node_modules", "dist", "vendor"} for part in path.parts):
        continue
    text = re.sub(r"/\*.*?\*/", "", path.read_text(encoding="utf-8"), flags=re.S)
    defined.update(re.findall(r"(--(?:wb|color)-[\w-]+)\s*:", text))
missing = {name: count for name, count in sorted(used.items()) if name not in defined}
for name, count in missing.items():
    print(f"{name}: {count}")
print(f"未定义变量：{len(missing)}；引用次数：{sum(missing.values())}")
sys.exit(bool(missing))
PY
```

## 协作纪律

- git push 和合并 main 必须先获得用户明确同意。
- 不碰 `~/GitHub/workbench`；仅用户明确指定的只读参考允许读取，禁止修改。
- 不改禅道源码。
- 数据库只允许 SELECT；写库须先获得用户同意。
- 不碰 8090、8095、8080、9000 端口的服务和数据库容器。
- 重建 8098 前先保留旧容器，确保可以回滚；此条不构成重建授权。
