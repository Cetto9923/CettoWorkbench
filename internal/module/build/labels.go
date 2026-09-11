// =============================================================================
// 文件: internal/module/build/labels.go
// 模块: 版本管理
// 类型: action
// 职责: 关联需求列表展示文案与默认勾选规则（对齐禅道 story 语言包）。
// 依赖: 无
// =============================================================================

package build

import (
	"fmt"
	"strconv"
	"strings"
)

var storyStatusLabels = map[string]string{
	"draft":      "草稿",
	"reviewing":  "评审中",
	"active":     "激活",
	"changing":   "变更中",
	"closed":     "已关闭",
	"launched":   "已投产",
	"developing": "研发中",
}

var storyStageLabels = map[string]string{
	"wait":       "未开始",
	"planned":    "已计划",
	"projected":  "研发立项",
	"designing":  "设计中",
	"designed":   "设计完毕",
	"developing": "研发中",
	"developed":  "研发完毕",
	"testing":    "测试中",
	"tested":     "测试完毕",
	"verified":   "已验收",
	"rejected":   "验收失败",
	"delivering": "交付中",
	"delivered":  "已交付",
	"released":   "已发布",
	"closed":     "已关闭",
}

// ResolveExecutionID 对齐禅道：execution 优先，否则用 project。
func ResolveExecutionID(executionID, projectID uint) uint {
	if executionID != 0 {
		return executionID
	}
	return projectID
}

// ParseCSVUintIDs 解析逗号分隔 ID，去重且跳过 0。
func ParseCSVUintIDs(csv string) []uint {
	parts := strings.Split(csv, ",")
	seen := make(map[uint]struct{}, len(parts))
	out := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil || n == 0 {
			continue
		}
		id := uint(n)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// MergeAllStoriesCSV 对齐禅道 joinChildBuilds：合并父版本与子版本 stories。
func MergeAllStoriesCSV(stories string, childStories []string) []uint {
	var b strings.Builder
	b.WriteString(stories)
	for _, s := range childStories {
		if strings.TrimSpace(s) == "" {
			continue
		}
		b.WriteByte(',')
		b.WriteString(s)
	}
	return ParseCSVUintIDs(b.String())
}

// ShouldDefaultCheckStage 对齐禅道：developed / tested / closed 默认勾选。
func ShouldDefaultCheckStage(stage string) bool {
	switch stage {
	case "developed", "tested", "closed":
		return true
	default:
		return false
	}
}

// StoryStatusLabel 状态中文；未知则回退原值。
func StoryStatusLabel(status string) string {
	if label, ok := storyStatusLabels[status]; ok {
		return label
	}
	return status
}

// StoryStageLabel 阶段中文；未知则回退原值。
func StoryStageLabel(stage string) string {
	if label, ok := storyStageLabels[stage]; ok {
		return label
	}
	return stage
}

// FormatEstimate 预计工时展示（对齐禅道 hourUnit=h）。
func FormatEstimate(estimate float32) string {
	if estimate == float32(int(estimate)) {
		return fmt.Sprintf("%dh", int(estimate))
	}
	return fmt.Sprintf("%gh", estimate)
}

// PriorityClass 优先级样式 class（0 视为 P4）。
func PriorityClass(pri uint8) string {
	if pri == 0 {
		return "P4"
	}
	if pri > 4 {
		return "P4"
	}
	return fmt.Sprintf("P%d", pri)
}
