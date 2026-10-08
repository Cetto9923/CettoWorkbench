// =============================================================================
// 文件: internal/module/po/repo_deliver_scan.go
// 模块: PO 工作台
// 类型: repository
// 职责: 业务需求和独立研需共用交付缺陷计数及窗口查询结果读取。
// =============================================================================
package po

import "context"

func (r *Repo) readDeliverBugCounts(ctx context.Context, query string, id uint) (int, int, error) {
	db, err := r.homeActionWriter()
	if err != nil {
		return 0, 0, err
	}
	var row struct {
		SevereCount int `gorm:"column:severe_count"`
		OpenCount   int `gorm:"column:open_count"`
	}
	if err := db.WithContext(ctx).Raw(query, id).Scan(&row).Error; err != nil {
		return 0, 0, err
	}
	return row.SevereCount, row.OpenCount, nil
}

func (r *Repo) readDeliverWindow(ctx context.Context, query string, id uint) (uint, string, string, error) {
	db, err := r.homeActionWriter()
	if err != nil {
		return 0, "", "", err
	}
	var row struct {
		WindowID    uint   `gorm:"column:window_id"`
		WindowName  string `gorm:"column:window_name"`
		ReleaseDate string `gorm:"column:release_date"`
	}
	if err := db.WithContext(ctx).Raw(query, id).Scan(&row).Error; err != nil {
		return 0, "", "", err
	}
	return row.WindowID, row.WindowName, row.ReleaseDate, nil
}
