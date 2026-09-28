以 AGENTS.md 为准，两处须同步修改。

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
