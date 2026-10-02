#!/usr/bin/env bash
# =============================================================================
# 文件: tools/check-zentao-write.sh
# 模块: 质量门
# 类型: check script
# 职责: 拦截非测试 Go 代码里对禅道原生表（zt_*）的直接 INSERT/UPDATE/DELETE。
#       禅道已有的业务动作一律走 internal/pkg/zentao（zentao.Client），不得在工作台
#       重复实现，也不得直接写禅道表。
# 白名单: tools/zentao-write-allowlist.txt，每行 "<文件> <函数> <表>  # 理由"，理由必填。
# 待迁移: 理由以「待迁移」起首的条目。命中时醒目提示但不失败；键名不得超过
#         下方 PENDING_MAX（只允许减少，不允许新增）。超出清单的新直写仍失败。
# 退出码: 0=通过（含仅剩待迁移）；1=存在未豁免的禅道直写、待迁移新增，或白名单失效/格式有误。
# =============================================================================
set -euo pipefail

cd "$(dirname "$0")/.."

ALLOWLIST=tools/zentao-write-allowlist.txt
if [[ ! -f "$ALLOWLIST" ]]; then
  echo "缺少白名单文件：$ALLOWLIST"
  exit 1
fi

python3 - "$ALLOWLIST" <<'PY'
import re
import sys
from pathlib import Path

ALLOWLIST = Path(sys.argv[1])

WRITE = re.compile(r"(?:\.\s*|^\s*)(Create|Save|Updates?|Update|Delete|Exec)\s*\(")
TABLE_VAR = re.compile(r"Table\(\s*(\w+)\s*\)")
TABLE_EXPR = re.compile(r"Table\(\s*\((\w+)\{\}\)\.TableName\(\)")
TABLE = re.compile(r'Table\(\s*"([A-Za-z_]+)"')
RAWTBL = re.compile(r"(?:INSERT\s+(?:IGNORE\s+)?INTO|UPDATE|DELETE\s+FROM)\s+`?([A-Za-z_]+)`?", re.I)
INLINE_MODEL = re.compile(r"(?:Model|Create|Save|Delete)\s*\(\s*&(?:[\w]+\.)?([A-Za-z_]\w*)\s*\{")
VAR_MODEL = re.compile(r"(?:Model|Create|Save|Delete)\s*\(\s*&(\w+)\s*[,)]")
DECL = re.compile(r"\b(\w+)\s*:?=\s*(?:[\w]+\.)?([A-Za-z_]\w*)\s*\{")
# x := make([]T, ...) 形式的切片变量
MAKE = re.compile(r"\b(\w+)\s*:?=\s*make\(\s*\[\](?:[\w]+\.)?([A-Za-z_]\w*)\s*,")
FUNC = re.compile(r"^func\s")
# 同时匹配 `func (T) TableName()` 与 `func (r *T) TableName()`
TABLENAME = re.compile(
    r"func\s+\(\s*(?:\*?\s*([A-Za-z_]\w*)|\w+\s+\*?\s*([A-Za-z_]\w*))\s*\)\s*TableName\(\)"
    r'\s*string\s*\{\s*return\s+"([A-Za-z_]+)"')
# ztaction 是禅道 zt_action / zt_history 的唯一写通道
ZTACTION_CALL = {
    "Create": ("zt_action",),
    "LogEdited": ("zt_action", "zt_history"),
    "LogHistory": ("zt_history",),
}
ZTACTION = re.compile(r"ztaction\.(" + "|".join(ZTACTION_CALL) + r")\s*\(")
# 有禅道接口但仍直写的存量上限。允许从白名单删掉已改掉的键，不允许新增键。
PENDING_MAX = {
    ("internal/module/po/repo_deliver.go", "UpdateDemandDeliverFull", "zt_demand"),
    ("internal/module/po/repo_home_actions.go", "updateHomeDemandStatus", "zt_demand"),
    ("internal/module/schedule/repo_scheduling_write.go", "UpdateTask", "zt_task"),
}

go_files = [p for p in sorted(Path("internal").rglob("*.go")) if not p.name.endswith("_test.go")]

# --- 类型 -> 表名映射（含禅道模型与模块内局部模型）--------------------------
TYPE_TABLE = {}
for gomod in go_files:
    for m in TABLENAME.finditer(gomod.read_text("utf-8")):
        TYPE_TABLE[m.group(1) or m.group(2)] = m.group(3).lower()

# --- 工作台自有表：不算禅道直写 ---------------------------------------------
# 表所有权采用明确清单；SQL 建表和模型目录均不能自动把原生表变成自有表。
WORKBENCH = {
    "zt_login_failures",
    "zt_login_logs",
    "zt_roles",
    "zt_role_permissions",
    "zt_gf_user_roles",
    "zt_menus",
    "zt_versionwindow",
    "zt_versionwindowproduct",
    "zt_wb_versionwindow_milestone",
    "zt_wb_agileteam_orgmap",
    "zt_demandwindow",
    "zt_operation_logs",
    "zt_workbench_notify_reads",
    "zt_depts",
    "zt_wb_profile_pref_kv",
    "zt_wb_dept_manager_override",
    "zt_wb_agileteam_adjustment",
    "zt_wb_agileteam_adjustment_item",
    "zt_wb_agileteam_history",
}
# 只有上述明确登记的工作台表可免检查。
def is_zentao(table):
    return table.startswith("unresolved:") or (table.startswith("zt_") and table not in WORKBENCH)


def enclosing_func(lines, idx):
    for k in range(idx, -1, -1):
        m = FUNC.match(lines[k])
        if m:
            name = re.match(r"^func\s*(?:\([^)]*\)\s*)?([A-Za-z_]\w*)", lines[k])
            return name.group(1) if name else "?"
    return "?"


hits = []
for gomod in go_files:
    src = gomod.read_text("utf-8")
    lines = src.splitlines()
    for j, line in enumerate(lines):
        wm = WRITE.search(line)
        zm = ZTACTION.search(line)
        if not wm and not zm:
            continue
        # 语句起点：向上找上一条以 } ; { 收尾的行，避免跨语句误判
        begin = 0
        for k in range(j - 1, -1, -1):
            if lines[k].rstrip().endswith(("}", ";", "{")):
                begin = k + 1
                break
        # 语句终点：Exec( 等调用的 SQL 参数可能跨行，按括号配平向后延伸
        end, depth = j, 0
        for k in range(j, min(j + 12, len(lines))):
            depth += lines[k].count("(") - lines[k].count(")")
            end = k
            if k > j and depth <= 0:
                break
        func_start = 0
        for k in range(j, -1, -1):
            if FUNC.match(lines[k]):
                func_start = k
                break
        scope = {}
        for k in range(func_start, j + 1):
            m = DECL.search(lines[k])
            if m:
                scope[m.group(1)] = m.group(2)
            m = MAKE.search(lines[k])
            if m:
                scope[m.group(1)] = m.group(2)

        table_vars = {m.group(1): m.group(2).lower() for m in re.finditer(r'(\w+)\s*(?::?=)\s*"(zt_\w+)"', src)}
        if str(gomod) == "internal/module/agileteam/repo_basic_atomic.go":
            table_vars["table"] = "zt_teamgroup"
        tables = set()
        for k in range(begin, end + 1):
            seg = lines[k]
            for m in TABLE_VAR.finditer(seg):
                tables.add(table_vars.get(m.group(1), "unresolved:" + m.group(1)))
            for m in TABLE_EXPR.finditer(seg):
                if m.group(1) in TYPE_TABLE:
                    tables.add(TYPE_TABLE[m.group(1)])
            m = TABLE.search(seg)
            if m:
                tables.add(m.group(1).lower())
            for m in RAWTBL.finditer(seg):
                tables.add(m.group(1).lower())
            for m in INLINE_MODEL.finditer(seg):
                if m.group(1) in TYPE_TABLE:
                    tables.add(TYPE_TABLE[m.group(1)])
            for m in VAR_MODEL.finditer(seg):
                if scope.get(m.group(1), "") in TYPE_TABLE:
                    tables.add(TYPE_TABLE[scope[m.group(1)]])
            for m in ZTACTION.finditer(seg):
                tables.update(ZTACTION_CALL[m.group(1)])
        action = wm.group(1) if wm else ZTACTION.search(line).group(1)
        for table in sorted(t for t in tables if is_zentao(t)):
            key = (str(gomod), enclosing_func(lines, j), table)
            if key not in {(h[0], h[1], h[2]) for h in hits}:
                hits.append(key + (action,))

# --- 白名单 ----------------------------------------------------------------
exempt, pending = set(), set()
exempt_used, pending_used = set(), set()
bad_cfg = []
for lineno, raw in enumerate(ALLOWLIST.read_text("utf-8").splitlines(), 1):
    line = raw.strip()
    if not line or line.startswith("#"):
        continue
    body, _, reason = line.partition("#")
    parts = body.split()
    reason = reason.strip()
    if len(parts) != 3 or not reason:
        bad_cfg.append(f"{ALLOWLIST}:{lineno} 格式或豁免理由缺失：{raw}")
        continue
    key = (parts[0], parts[1], parts[2].lower())
    if reason.startswith("待迁移"):
        if key not in PENDING_MAX:
            bad_cfg.append(f"{ALLOWLIST}:{lineno} 待迁移清单不允许新增：{line}")
            continue
        if key in pending:
            bad_cfg.append(f"{ALLOWLIST}:{lineno} 重复条目：{line}")
        pending.add(key)
        pending_used.add(key)
        continue
    if key in PENDING_MAX:
        bad_cfg.append(f"{ALLOWLIST}:{lineno} 待迁移项不得改为静默豁免：{line}")
        continue
    if key in exempt or key in pending:
        bad_cfg.append(f"{ALLOWLIST}:{lineno} 重复条目：{line}")
    exempt.add(key)
    exempt_used.add(key)

violations, pending_hits = [], []
for path, func, table, verb in hits:
    key = (path, func, table)
    if key in pending:
        pending_used.discard(key)
        pending_hits.append((path, func, table, verb))
    elif key in exempt:
        exempt_used.discard(key)
    else:
        violations.append((path, func, table, verb))

for path, func, table, verb in sorted(violations):
    print(f"FAIL  {path}  函数 {func}  表 {table}  动作 {verb}")
for path, func, table in sorted(exempt_used):
    print(f"FAIL  白名单条目已无对应违规，请删除：{path}  函数 {func}  表 {table}")
for msg in bad_cfg:
    print(f"FAIL  {msg}")
if pending_hits or pending_used:
    print("======== 待迁移（不失败；只允许减少，不允许新增）========")
    for path, func, table, verb in sorted(pending_hits):
        print(f"待迁移  {path}  函数 {func}  表 {table}  动作 {verb}")
    for path, func, table in sorted(pending_used):
        print(f"待迁移已消除（允许减少，请从清单删除）  {path}  函数 {func}  表 {table}")
    print(f"======== 待迁移仍剩 {len(pending_hits)} 处直写。超出本清单的新直写仍失败。========")
exempted = len(hits) - len(violations) - len(pending_hits)
print(
    f"禅道直写检查：命中 {len(hits)} 处，其中豁免 {exempted} 处，"
    f"待迁移 {len(pending_hits)} 处，未豁免 {len(violations)} 处"
)
sys.exit(1 if (violations or bad_cfg or exempt_used) else 0)
PY
