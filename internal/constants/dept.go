// =============================================================================
// 文件: internal/constants/dept.go
// 模块: 常量
// 职责: 集中登记跨模块共用的部门口径常量，避免部门 ID 字面量散落在各处 SQL 中。
// =============================================================================

package constants

import "fmt"

// DeptTechHQID 科技部本部在 zt_dept 中的部门 ID。
// 部门负责人补缺表（zt_wb_dept_manager_override）与团队长管辖范围查询共用该口径。
const DeptTechHQID = 52

// DeptTechHQPathPattern 返回 zt_dept.path 中命中科技本部及其子孙部门的 LIKE 模式。
// 供 SQL 以 ? 占位符参数化传入，禁止把该 ID 直接拼进 SQL 字符串。
func DeptTechHQPathPattern() string {
	return fmt.Sprintf("%%,%d,%%", DeptTechHQID)
}
