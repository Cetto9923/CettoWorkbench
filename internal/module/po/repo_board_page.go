package po

import (
	"context"

	"gorm.io/gorm"
)

// boardRootQuery 的候选集同时用于总数与分页，避免分别截断两类根节点。
func (r *Repo) boardRootQuery(ctx context.Context, req BoardDemandReq) *gorm.DB {
	account := req.POAccount
	demands := r.db.WithContext(ctx).Table("zt_demand").
		Where(`id IN (SELECT demand FROM zt_demandclarify WHERE PM = ?)
			OR QD = ? OR RD = ? OR BRA = ?`, account, account, account, account).
		Where("parent IN ? AND deleted = ? AND status NOT IN ?", []int{0, -1}, "0", []string{"closed", "cancel"})
	if req.TeamgroupID > 0 {
		demands = demands.Where("teamGroup = ?", req.TeamgroupID)
	}
	stories := r.db.WithContext(ctx).Table("zt_story").
		Where("(fromDemand IS NULL OR fromDemand = 0) AND deleted = ? AND status <> ?", "0", "closed").
		Where("sourceType NOT IN ?", []string{"", "demandpool", "demandlib", "feedback"}).
		Where("assignedTo = ? OR openedBy = ?", account, account)
	if req.Stage != "" && req.Stage != "all" {
		demands = demands.Where("status = ?", req.Stage)
		stories = stories.Where("status = ?", req.Stage)
	}
	if req.Keyword != "" {
		demands = demands.Where("name LIKE ?", "%"+req.Keyword+"%")
		stories = stories.Where("title LIKE ?", "%"+req.Keyword+"%")
	}
	return r.db.WithContext(ctx).Table("(? UNION ALL ?) AS roots",
		demands.Select("id, 'demand' AS kind"), stories.Select("id, 'story' AS kind"))
}

func (r *Repo) findBoardRoots(ctx context.Context, req BoardDemandReq) ([]boardDemandRow, []boardStoryRow, int64, error) {
	q := r.boardRootQuery(ctx, req)
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, nil, 0, err
	}
	var refs []struct {
		ID   int64
		Kind string
	}
	if err := q.Select("id, kind").Order("kind ASC, id DESC").
		Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Scan(&refs).Error; err != nil {
		return nil, nil, 0, err
	}
	var demandIDs, storyIDs []int64
	for _, ref := range refs {
		if ref.Kind == "demand" {
			demandIDs = append(demandIDs, ref.ID)
		} else {
			storyIDs = append(storyIDs, ref.ID)
		}
	}
	var demands []boardDemandRow
	if len(demandIDs) > 0 {
		if err := r.db.WithContext(ctx).Table("zt_demand").Where("id IN ? AND deleted = ?", demandIDs, "0").
			Select("id, parent, name, status, pri, assignedTo, deadline").Order("id DESC").Find(&demands).Error; err != nil {
			return nil, nil, 0, err
		}
	}
	var stories []boardStoryRow
	if len(storyIDs) > 0 {
		if err := r.db.WithContext(ctx).Table("zt_story").Where("id IN ? AND deleted = ?", storyIDs, "0").
			Select("id, fromDemand, title, status, stage, pri, assignedTo, product, sourceType").Order("id DESC").Find(&stories).Error; err != nil {
			return nil, nil, 0, err
		}
	}
	return demands, stories, total, nil
}
