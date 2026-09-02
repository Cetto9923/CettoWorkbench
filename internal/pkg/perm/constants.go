// =============================================================================
// 文件: internal/pkg/perm/constants.go
// 模块: 基础设施
// 类型: infra
// 职责: 定义权限常量。
// 依赖: 无
// =============================================================================

package perm

// Permission 表示系统权限标识。
type Permission string

// PermInfo 表示权限的完整元信息。
type PermInfo struct {
	Code   Permission
	Name   string
	Module string
}

const (
	AuthLogout Permission = "auth:logout"

	UserList          Permission = "user:list"
	UserCreate        Permission = "user:create"
	UserUpdate        Permission = "user:update"
	UserDelete        Permission = "user:delete"
	UserAssignRole    Permission = "user:assignrole"
	UserResetPassword Permission = "user:resetpassword"

	OperationLogList Permission = "operationlog:list"
	LoginLogList     Permission = "loginlog:list"

	// 角色管理
	RoleList   Permission = "role:list"
	RoleCreate Permission = "role:create"
	RoleEdit   Permission = "role:edit"
	RoleDelete Permission = "role:delete"

	// 菜单管理
	MenuList   Permission = "menu:list"
	MenuCreate Permission = "menu:create"
	MenuEdit   Permission = "menu:edit"
	MenuDelete Permission = "menu:delete"

	// 部门管理
	DeptList   Permission = "dept:list"
	DeptCreate Permission = "dept:create"
	DeptEdit   Permission = "dept:edit"
	DeptDelete Permission = "dept:delete"

	// 排期工作台
	ScheduleList   Permission = "schedule:list"
	ScheduleCreate Permission = "schedule:create"
	ScheduleUpdate Permission = "schedule:update"
	ScheduleDelete Permission = "schedule:delete"

	// PO 工作台首页（与 zt_menus.perm='po:home' 对齐）
	PoHome Permission = "po:home"
)

// allPermInfos 必须与上方 const 块中的所有 Permission 常量保持一一对应。
// CI 通过 TestAllPermissionsComplete 自动检查数量一致性。
var allPermInfos = []PermInfo{
	{Code: AuthLogout, Name: "认证-登出", Module: "auth"},
	{Code: UserList, Name: "用户-列表", Module: "user"},
	{Code: UserCreate, Name: "用户-新增", Module: "user"},
	{Code: UserUpdate, Name: "用户-编辑", Module: "user"},
	{Code: UserDelete, Name: "用户-删除", Module: "user"},
	{Code: UserAssignRole, Name: "用户-分配角色", Module: "user"},
	{Code: UserResetPassword, Name: "用户-重置密码", Module: "user"},
	{Code: OperationLogList, Name: "操作日志-列表", Module: "operationlog"},
	{Code: LoginLogList, Name: "登录日志-列表", Module: "loginlog"},
	{Code: RoleList, Name: "角色-列表", Module: "role"},
	{Code: RoleCreate, Name: "角色-新增", Module: "role"},
	{Code: RoleEdit, Name: "角色-编辑", Module: "role"},
	{Code: RoleDelete, Name: "角色-删除", Module: "role"},
	{Code: MenuList, Name: "菜单-列表", Module: "menu"},
	{Code: MenuCreate, Name: "菜单-新增", Module: "menu"},
	{Code: MenuEdit, Name: "菜单-编辑", Module: "menu"},
	{Code: MenuDelete, Name: "菜单-删除", Module: "menu"},
	{Code: DeptList, Name: "部门-列表", Module: "dept"},
	{Code: DeptCreate, Name: "部门-新增", Module: "dept"},
	{Code: DeptEdit, Name: "部门-编辑", Module: "dept"},
	{Code: DeptDelete, Name: "部门-删除", Module: "dept"},
	{Code: ScheduleList, Name: "排期-列表", Module: "schedule"},
	{Code: ScheduleCreate, Name: "排期-新增", Module: "schedule"},
	{Code: ScheduleUpdate, Name: "排期-编辑", Module: "schedule"},
	{Code: ScheduleDelete, Name: "排期-删除", Module: "schedule"},
	{Code: PoHome, Name: "PO工作台-首页", Module: "po"},
}

// systemPerms 是系统内置放行权限，不对外暴露到权限配置 UI。
var systemPerms = map[Permission]bool{
	AuthLogout: true,
}

// Configurable 返回可分配给角色的权限列表（排除系统内置权限）。
func Configurable() []PermInfo {
	out := make([]PermInfo, 0, len(allPermInfos))
	for _, p := range allPermInfos {
		if !systemPerms[p.Code] {
			out = append(out, p)
		}
	}
	return out
}

// String 返回权限的字符串值。
func (p Permission) String() string {
	return string(p)
}

// All 返回所有已定义权限，供种子数据和初始化流程使用。
func All() []Permission {
	out := make([]Permission, len(allPermInfos))
	for i, p := range allPermInfos {
		out[i] = p.Code
	}
	return out
}

// AllWithInfo 返回所有权限及其中文名称、模块信息。
func AllWithInfo() []PermInfo {
	out := make([]PermInfo, len(allPermInfos))
	copy(out, allPermInfos)
	return out
}
