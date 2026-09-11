// =============================================================================
// 文件: internal/module/testtask/builds.go
// 模块: 提测办理
// 类型: action
// 职责: 将禅道产品版本列表装配为「已有版本」下拉项。
// 依赖: 无
// =============================================================================

package testtask

import "strconv"

// BuildBuildOptions 将禅道产品 builds 装配为检索下拉项；跳过 id=0，空名称回退为 id。
func BuildBuildOptions(items []productBuild) []BuildOption {
	out := make([]BuildOption, 0, len(items))
	for _, item := range items {
		if item.ID == 0 {
			continue
		}
		value := strconv.FormatUint(uint64(item.ID), 10)
		label := item.Name
		if label == "" {
			label = value
		}
		out = append(out, BuildOption{Value: value, Label: label})
	}
	return out
}
