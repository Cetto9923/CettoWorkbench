// =============================================================================
// 文件: internal/module/po/repo.go
// 模块: PO 工作台
// 类型: action
// 职责: Repo 结构体与 NewRepo 构造器；其它数据访问职责拆分为：
//       repo_valuestream.go（价值流阶段聚合查询）
//       repo_kpi.go（首页 5 个焦点摘要 KPI 计数）
//       repo_todo.go（我的待办聚合查询）
//       repo_done.go（我的已办 zt_action 查询）
// 依赖: 无
// =============================================================================

package po

import (
	"gorm.io/gorm"
)

// Repo PO 工作台数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

