// Package workbenchroles 提供工作台角色 key、角色字典与默认允许角色的静态推导（不依赖 DB）。
package workbenchroles

// 角色 key 常量。key 在全系统内必须保持稳定，用于权限位、模板分支、
// 路由分组等场景。
const (
	RoleLead = "lead"
	RolePO   = "po"
	RoleSM   = "sm"
	RoleDev  = "dev"
	RoleQA   = "qa"
	RolePMO  = "pmo"
)

// RoleDef 描述一个角色的静态元信息。
//
// Label 是面向中文用户展示的名称；Description 是对角色职责的简要说明，
// 当前版本允许为空字符串，但调用方不应假定 Description 一定有值。
type RoleDef struct {
	Key         string
	Label       string
	Description string
}

// roleEntries 是 RoleMap 的内部真源，必须与 Role* 常量保持一一对应。
// 顺序固定：与 AllRoleKeys() 保持一致。
var roleEntries = []RoleDef{
	{Key: RolePO, Label: "产品负责人", Description: "负责需求梳理、价值定义与跨团队拉通"},
	{Key: RoleLead, Label: "团队长", Description: "负责团队交付、人员安排与质量把关"},
	{Key: RoleSM, Label: "Scrum Master", Description: "负责迭代节奏、阻塞清理与持续改进"},
	{Key: RoleDev, Label: "研发", Description: "负责代码实现、自测与日常开发任务"},
	{Key: RoleQA, Label: "测试", Description: "负责用例设计、测试执行与质量反馈"},
	{Key: RolePMO, Label: "PMO", Description: "负责项目组合管理、过程度量与跨项目协同"},
}

// RoleMap 返回固定的角色字典（key → RoleDef）。
//
// 返回值是新拷贝的 map，调用方可以安全地就地修改；底层 RoleDef 是值类型，
// 修改不会影响全局状态。
func RoleMap() map[string]RoleDef {
	out := make(map[string]RoleDef, len(roleEntries))
	for _, r := range roleEntries {
		out[r.Key] = r
	}
	return out
}

// AllRoleKeys 返回全部角色 key 的稳定顺序列表。
//
// 顺序保证：PO 在第一位（首页/默认身份），其余按 roleEntries 中声明顺序。
// 调用方若需要按字典序遍历，请自行排序，不要依赖本函数的顺序语义。
func AllRoleKeys() []string {
	out := make([]string, len(roleEntries))
	for i, r := range roleEntries {
		out[i] = r.Key
	}
	return out
}

// DefaultAllowedFor 根据账号、是否超管、部门 ID 返回默认允许的角色 key 列表。
//
// 当前最小规则：
//   - super admin → 返回 AllRoleKeys 全集；
//   - 其它账号 → 返回 []string{RolePO, RoleDev, RoleQA}；
//
// deptID 当前未参与判定；后续版本可在此处按部门 ID 进一步细分（例如对
// 研发部门默认额外追加 RoleLead，对 PMO 部门追加 RolePMO 等）。本版本不实现。
//
// 返回值是新分配的 slice，调用方可以安全地就地修改。
func DefaultAllowedFor(account string, isSuperAdmin bool, deptID uint64) []string {
	_ = account
	_ = deptID
	if isSuperAdmin {
		return AllRoleKeys()
	}
	return []string{RolePO, RoleDev, RoleQA}
}
