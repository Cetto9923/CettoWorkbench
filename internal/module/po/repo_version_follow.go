// =============================================================================
// 文件: internal/module/po/repo_version_follow.go
// 模块: PO 工作台
// 类型: repo
// 职责: 版本跟进页只读数据：版本窗口、窗口关联需求、阶段分布、发布判断。
//       严格只读真实数据，不做任何兜底或回落。
// =============================================================================

package po

import (
	"context"
	"time"
)

// vfWindowRow 版本窗口原始行。
type vfWindowRow struct {
	ID          uint64
	Name        string
	ReleaseDate time.Time
	StartDate   *time.Time
	Status      string
	TeamgroupID uint
	TeamName    string
}

// vfDemandRow 窗口内需求的原始行，字段缺失一律留空由上层显示「暂无」。
// 注：zt_demand 表无 owner/updatedDate 列，负责人取 assignedTo，排序取 editedDate。
type vfDemandRow struct {
	ID               uint
	No               string
	Title            string
	Stage            string
	Status           string
	MainSystem       string
	Product          string
	Priority         string
	Deadline         *time.Time
	SchedulePlanDate *time.Time
	Acceptance       string
	VeriFier         string
	Owner            string
	FinalStatus      string
	EditedDate       *time.Time
}

// ListVFWindows 列出未删除的版本窗口；已按上线日排序。
func (r *Repo) ListVFWindows(ctx context.Context, teamgroupID uint, limit int) ([]vfWindowRow, error) {
	rows := []vfWindowRow{}
	q := r.db.WithContext(ctx).Table("zt_versionwindow vw").
		Select("vw.id, vw.name, vw.releaseDate, vw.startDate, vw.status, vw.teamgroup, tg.name AS teamName").
		Joins("LEFT JOIN zt_teamgroup tg ON tg.id = vw.teamgroup AND tg.deleted = '0'").
		Where("vw.deletedAt IS NULL")
	if teamgroupID > 0 {
		q = q.Where("vw.teamgroup = ?", teamgroupID)
	}
	if err := q.Order("vw.releaseDate ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountVFDemandsInWindow 统计窗口内已关联的业务需求数（story=0）。
func (r *Repo) CountVFDemandsInWindow(ctx context.Context, windowIDs []uint64) (map[uint64]int, error) {
	out := map[uint64]int{}
	if len(windowIDs) == 0 {
		return out, nil
	}
	type row struct {
		WindowID uint64
		N        int
	}
	rows := []row{}
	err := r.db.WithContext(ctx).Table("zt_demandwindow dw").
		Select("dw.versionWindow AS windowID, COUNT(*) AS n").
		Joins("JOIN zt_demand d ON d.id = dw.demand AND d.deleted = '0'").
		Where("dw.deletedAt IS NULL AND dw.story = 0 AND dw.versionWindow IN ?", windowIDs).
		Group("dw.versionWindow").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.WindowID] = x.N
	}
	return out, nil
}

// ListVFDemandsInWindow 列出窗口内关联的业务需求，按最近编辑时间倒序。
func (r *Repo) ListVFDemandsInWindow(ctx context.Context, windowID uint64) ([]vfDemandRow, error) {
	rows := []vfDemandRow{}
	err := r.db.WithContext(ctx).Table("zt_demandwindow dw").
		Select("d.id, d.id AS no, d.title, d.stage, d.status, d.mainSystem, d.product, "+
			"d.pri AS priority, d.deadline, d.schedulePlanDate, d.acceptance, d.veriFier, "+
			"d.assignedTo AS owner, d.finalStatus, d.editedDate").
		Joins("JOIN zt_demand d ON d.id = dw.demand AND d.deleted = '0'").
		Where("dw.deletedAt IS NULL AND dw.story = 0 AND dw.versionWindow = ?", windowID).
		Order("d.editedDate DESC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// CountVFOrphanTestedDemands 统计已到测试后阶段、但未挂任何版本窗口的需求数。
// 这是「AI 发现」的一条确定性规则，来源是真实数据而非本页列表。
func (r *Repo) CountVFOrphanTestedDemands(ctx context.Context) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("zt_demand d").
		Where("d.deleted = '0' AND d.status IN ?", vfTestedStatuses).
		Where("NOT EXISTS (SELECT 1 FROM zt_demandwindow dw WHERE dw.demand = d.id AND dw.deletedAt IS NULL AND dw.story = 0)").
		Count(&n).Error
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// vfTestedStatuses 是「已到测试后阶段」在 zt_demand.status 里的真实取值。
// 注：zt_demand.deleted 是 enum('0','1')，比较必须用字符串 '0'；用数字 0 在 OceanBase 上不匹配，会恒为空。
var vfTestedStatuses = []string{"testing", "delivered", "acceptanced", "waitacceptance", "waitdeliver", "released"}

// ListVFOrphanTestedDemands 列出上述未挂窗口的需求，最多 limit 条，供依据弹窗展示。
func (r *Repo) ListVFOrphanTestedDemands(ctx context.Context, limit int) ([]vfDemandRow, error) {
	rows := []vfDemandRow{}
	err := r.db.WithContext(ctx).Table("zt_demand d").
		Select("d.id, d.id AS no, d.title, d.stage, d.status, d.mainSystem, d.product, "+
			"d.pri AS priority, d.deadline, d.schedulePlanDate, d.acceptance, d.veriFier, "+
			"d.assignedTo AS owner, d.finalStatus, d.editedDate").
		Where("d.deleted = '0' AND d.status IN ?", vfTestedStatuses).
		Where("NOT EXISTS (SELECT 1 FROM zt_demandwindow dw WHERE dw.demand = d.id AND dw.deletedAt IS NULL AND dw.story = 0)").
		Order("d.editedDate DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
