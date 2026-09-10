// =============================================================================
// 文件: internal/module/testtask/execution.go
// 模块: 提测办理
// 类型: action
// 职责: 对齐禅道版本创建执行下拉：stagefilter / leaf / order_asc / noclosed。
// 依赖: 无
// =============================================================================

package testtask

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var waterfallModels = map[string]struct{}{
	"waterfall":     {},
	"waterfallplus": {},
}

var stagefilterAttrs = map[string]struct{}{
	"request": {},
	"design":  {},
	"review":  {},
}

// BuildExecutionOptions 按禅道 create build 的 mode 过滤并装配下拉项。
// mode 语义：stagefilter | leaf | order_asc | 可选 noclosed（CRExecution==0 时）。
func BuildExecutionOptions(rows []executionRow, noClosed bool) []ExecutionOption {
	if len(rows) == 0 {
		return []ExecutionOption{}
	}

	parentSet := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		if row.Parent > 0 {
			parentSet[row.Parent] = struct{}{}
		}
	}

	filtered := make([]executionRow, 0, len(rows))
	for _, row := range rows {
		if _, isParent := parentSet[row.ID]; isParent {
			continue // leaf
		}
		if noClosed && (row.Status == "done" || row.Status == "closed") {
			continue // noclosed
		}
		model := strings.ToLower(row.ProjectModel)
		if _, isWF := waterfallModels[model]; isWF {
			if _, ban := stagefilterAttrs[strings.ToLower(row.Attribute)]; ban {
				continue // stagefilter
			}
		}
		filtered = append(filtered, row)
	}

	sorted := resetExecutionSortsByProject(filtered)
	out := make([]ExecutionOption, 0, len(sorted))
	for _, row := range sorted {
		projName := row.ProjectName
		if projName == "" {
			projName = strconv.FormatUint(uint64(row.ProjectID), 10)
		}
		execName := row.Name
		if execName == "" {
			execName = strconv.FormatUint(uint64(row.ID), 10)
		}
		out = append(out, ExecutionOption{
			Value:       fmt.Sprintf("%d-%d", row.ProjectID, row.ID),
			Label:       projName + "/" + execName,
			ProjectID:   row.ProjectID,
			ExecutionID: row.ID,
		})
	}
	return out
}

// resetExecutionSortsByProject 按项目分组后做阶段树序（对齐 execution::resetExecutionSorts）。
func resetExecutionSortsByProject(rows []executionRow) []executionRow {
	if len(rows) == 0 {
		return nil
	}
	byProject := make(map[uint][]executionRow, len(rows))
	projectOrder := make([]uint, 0)
	seenProj := make(map[uint]struct{})
	for _, row := range rows {
		if _, ok := seenProj[row.ProjectID]; !ok {
			seenProj[row.ProjectID] = struct{}{}
			projectOrder = append(projectOrder, row.ProjectID)
		}
		byProject[row.ProjectID] = append(byProject[row.ProjectID], row)
	}
	sort.SliceStable(projectOrder, func(i, j int) bool {
		return projectOrder[i] < projectOrder[j]
	})

	out := make([]executionRow, 0, len(rows))
	for _, pid := range projectOrder {
		out = append(out, resetExecutionSorts(byProject[pid])...)
	}
	return out
}

func resetExecutionSorts(executions []executionRow) []executionRow {
	if len(executions) == 0 {
		return nil
	}
	inSet := make(map[uint]executionRow, len(executions))
	var grade1 []executionRow
	children := make(map[uint][]executionRow)
	for _, e := range executions {
		inSet[e.ID] = e
		if e.Grade == 1 {
			grade1 = append(grade1, e)
		}
		if e.Grade > 1 && e.Parent > 0 {
			children[e.Parent] = append(children[e.Parent], e)
		}
	}

	sorted := make([]executionRow, 0, len(executions))
	added := make(map[uint]struct{}, len(executions))
	var walk func(parents []executionRow)
	walk = func(parents []executionRow) {
		for _, p := range parents {
			if _, ok := added[p.ID]; !ok {
				if orig, exists := inSet[p.ID]; exists {
					sorted = append(sorted, orig)
					added[p.ID] = struct{}{}
				}
			}
			if kids := children[p.ID]; len(kids) > 0 {
				walk(kids)
			}
		}
	}
	walk(grade1)

	// 未落入树序的（无 grade/parent 信息）按原序追加
	for _, e := range executions {
		if _, ok := added[e.ID]; ok {
			continue
		}
		sorted = append(sorted, e)
		added[e.ID] = struct{}{}
	}
	return sorted
}
