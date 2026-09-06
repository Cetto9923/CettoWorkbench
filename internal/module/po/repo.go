// =============================================================================
// 文件: internal/module/po/repo.go
// 模块: PO 工作台
// 类型: action
// 职责: Repo 结构体与 NewRepo 构造器；其它数据访问职责拆分为：
//       repovaluestream.go（价值流阶段聚合查询）
//       repokpi.go（首页 5 个焦点摘要 KPI 计数）
//       repotodo.go（我的待办聚合查询）
//       repodone.go（我的已办 zt_action 查询）
// 依赖: 无
// =============================================================================

package po

import (
	"gorm.io/gorm"
)

// Repo PO 工作台数据访问。
// 读路径使用 db（只读池，可 nil 降级）；写路径使用 writeDB（主库）。
// 禁止把关注/已读等写操作发到只读连接。
type Repo struct {
	db      *gorm.DB
	writeDB *gorm.DB
}

// NewRepo 创建 Repo。
// readDB 供查询；writeDB 供 SaveDemandFollow / SaveNoticeRead 等写入。
// 单测可用同一句柄：NewRepo(db, db)。
func NewRepo(readDB, writeDB *gorm.DB) *Repo {
	return &Repo{db: readDB, writeDB: writeDB}
}
