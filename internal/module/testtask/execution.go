// =============================================================================
// 文件: internal/module/testtask/execution.go
// 模块: 提测办理
// 类型: action
// 职责: 对齐禅道 productTao::buildExecutionPairs（mode=stagefilter, withProjectName）。
// 依赖: 无
// =============================================================================

package testtask

import (
	"fmt"
	"strconv"
	"strings"
)

// 对齐禅道 buildExecutionPairs stagefilter 过滤的阶段属性。
var stagefilterAttrs = map[string]struct{}{
	"request": {},
	"design":  {},
	"review":  {},
}

// 对齐 $lang->project->disableExecution（zh-cn：不启用执行的项目）。
const disableExecutionLabel = "不启用执行的项目"

// BuildExecutionOptions 对齐禅道 productTao::buildExecutionPairs($executions, 'stagefilter', true)。
func BuildExecutionOptions(rows []executionRow, stages []stageRow) []ExecutionOption {
	if len(rows) == 0 {
		return []ExecutionOption{}
	}

	stageByID := make(map[uint]stageRow, len(stages))
	parentSet := make(map[uint]struct{}, len(stages))
	for _, st := range stages {
		stageByID[st.ID] = st
		if st.Parent > 0 {
			parentSet[st.Parent] = struct{}{}
		}
	}

	out := make([]ExecutionOption, 0, len(rows))
	for _, row := range rows {
		if _, isParent := parentSet[row.ID]; isParent {
			continue
		}
		if _, ban := stagefilterAttrs[strings.ToLower(row.Attribute)]; ban {
			continue
		}

		label := formatExecutionLabel(row, stageByID)
		projName := row.ProjectName
		if projName == "" {
			projName = strconv.FormatUint(uint64(row.ProjectID), 10)
		}
		if isEmptyMultiple(row.Multiple) {
			label = projName + "(" + disableExecutionLabel + ")"
		}

		out = append(out, ExecutionOption{
			Value:       fmt.Sprintf("%d-%d", row.ProjectID, row.ID),
			Label:       label,
			ProjectID:   row.ProjectID,
			ExecutionID: row.ID,
		})
	}
	return out
}

// formatExecutionLabel 对齐 buildExecutionPairs 中 withProjectName=true 的名称拼装。
func formatExecutionLabel(row executionRow, stageByID map[uint]stageRow) string {
	projName := row.ProjectName
	if projName == "" {
		projName = strconv.FormatUint(uint64(row.ProjectID), 10)
	}

	if _, isStage := stageByID[row.ID]; isStage {
		name := ""
		for _, seg := range pathSegments(row.Path) {
			if st, ok := stageByID[seg]; ok {
				name += "/" + st.Name
			}
		}
		return projName + name
	}

	execName := row.Name
	if execName == "" {
		execName = strconv.FormatUint(uint64(row.ID), 10)
	}
	return projName + "/" + execName
}

func pathSegments(path string) []uint {
	path = strings.Trim(path, ",")
	if path == "" {
		return nil
	}
	parts := strings.Split(path, ",")
	// 对齐 PHP array_slice(explode(...), 1)：跳过首段（通常为项目/根）。
	if len(parts) <= 1 {
		return nil
	}
	out := make([]uint, 0, len(parts)-1)
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseUint(p, 10, 64)
		if err != nil || id == 0 {
			continue
		}
		out = append(out, uint(id))
	}
	return out
}

// isEmptyMultiple 对齐 PHP empty($execution->multiple)（'0'/'' 均视为空）。
func isEmptyMultiple(v string) bool {
	return v == "" || v == "0"
}
