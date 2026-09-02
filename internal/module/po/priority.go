// =============================================================================
// 文件: internal/module/po/priority.go
// 模块: PO 工作台
// 类型: readonly
// 职责: 禅道需求的优先级字段在不同表里类型不一致—— zt_demand.pri 是 char(30)，
//       zt_story.pri 是 tinyint unsigned。这里把两类来源归一为可排序的整数权重
//       (PriRank) 与对外展示字符串 ("P1" ... "P4"/"")，避免上层做零散判断。
// 依赖: 无
// =============================================================================

package po

import (
	"strconv"
	"strings"
)

// PriRankUnknown 表示未识别优先级（未填、空、异常格式）；统一置于排序末位。
const PriRankUnknown = 99

// ParsePriDemand 解析 zt_demand.pri（字符串）到权重 (PriRank) 和展示文案。
// 数据库实际存值为 '1'/'2'/'3'/'4'（空串或未知值视为 PriRankUnknown）。
func ParsePriDemand(raw string) (rank int, label string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return PriRankUnknown, ""
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 4 {
		return PriRankUnknown, ""
	}
	return n, "P" + s
}

// ParsePriStory 解析 zt_story.pri（tinyint 数字）到权重和展示文案。
func ParsePriStory(raw int) (rank int, label string) {
	if raw < 1 || raw > 4 {
		return PriRankUnknown, ""
	}
	return raw, "P" + strconv.Itoa(raw)
}
