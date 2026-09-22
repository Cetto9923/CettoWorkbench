// =============================================================================
// 文件: internal/module/build/repo.go
// 模块: 版本管理
// 类型: action
// 职责: 版本与可关联研发需求查询（对齐禅道 build::linkStory 默认列表）。
// 依赖: internal/model/zentao
// =============================================================================

package build

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"

	ztmodel "workbench/internal/model/zentao"
)

var errBuildNotFound = errors.New("build not found")

// Repo 版本数据访问。db 为主库：版本存在性判断不能容忍备库弱读延迟。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// storyTitleRow 已关联需求的 id/title（Repo 查询行）。
type storyTitleRow struct {
	ID    uint   `gorm:"column:id"`
	Title string `gorm:"column:title"`
}

// FindBuildByID 按 ID 读取未删除版本。
func (r *Repo) FindBuildByID(ctx context.Context, id uint) (*ztmodel.ZtBuild, error) {
	if r == nil || r.db == nil || id == 0 {
		return nil, errBuildNotFound
	}
	var m ztmodel.ZtBuild
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted = ?", id, "0").
		Limit(1).
		Take(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errBuildNotFound
		}
		return nil, err
	}
	return &m, nil
}

// FindChildBuildStories 读取子版本的 stories 字段（对齐 joinChildBuilds）。
func (r *Repo) FindChildBuildStories(ctx context.Context, childIDs []uint) ([]string, error) {
	if r == nil || r.db == nil || len(childIDs) == 0 {
		return nil, nil
	}
	var rows []ztmodel.ZtBuild
	err := r.db.WithContext(ctx).
		Select("id", "stories").
		Where("id IN ? AND deleted = ?", childIDs, "0").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Stories)
	}
	return out, nil
}

// FindStoryTitlesByIDs 按 ID 批量读取未删除研发需求标题。
func (r *Repo) FindStoryTitlesByIDs(ctx context.Context, ids []uint) ([]storyTitleRow, error) {
	if r == nil || r.db == nil || len(ids) == 0 {
		return nil, nil
	}
	var rows []storyTitleRow
	err := r.db.WithContext(ctx).
		Model(&ztmodel.ZtStory{}).
		Select("id", "title").
		Where("id IN ? AND deleted = ? AND type = ?", ids, "0", "story").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// FindParentStoryIDs 产品下父需求 ID（isParent=1），对齐 getExcludeStoryIdList。
func (r *Repo) FindParentStoryIDs(ctx context.Context, productID uint) ([]uint, error) {
	if r == nil || r.db == nil || productID == 0 {
		return nil, nil
	}
	var ids []uint
	err := r.db.WithContext(ctx).
		Model(&ztmodel.ZtStory{}).
		Where("product = ? AND type = ? AND isParent = ? AND deleted = ?", productID, "story", "1", "0").
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// linkableStoryRow 可关联需求查询行。
type linkableStoryRow struct {
	ID         uint    `gorm:"column:id"`
	Pri        uint8   `gorm:"column:pri"`
	Title      string  `gorm:"column:title"`
	OpenedBy   string  `gorm:"column:openedBy"`
	AssignedTo string  `gorm:"column:assignedTo"`
	Estimate   float32 `gorm:"column:estimate"`
	Status     string  `gorm:"column:status"`
	Stage      string  `gorm:"column:stage"`
	FromDemand uint    `gorm:"column:fromDemand"`
}

// RepoFindLinkableStoriesReq 可关联需求分页查询入参。
type RepoFindLinkableStoriesReq struct {
	ExecutionID uint
	ProductID   uint
	BranchCSV   string
	ExcludeIDs  []uint
	DemandID    uint
	Limit       int
	Offset      int
}

const linkableStorySelect = "DISTINCT s.id, s.pri, s.title, s.openedBy, s.assignedTo, s.estimate, s.status, s.stage, s.fromDemand"

// FindLinkableStories 对齐禅道 getExecutionStories 默认列表。
// DemandID>0 时并入当前业务需求拆分出的研发需求（fromDemand），并套用产品、分支与排除条件。
//
// 执行+业需并集不用 OR+EXISTS：优化器常退化成扫 zt_story；改为 id IN (projectstory ∪ fromDemand)。
func (r *Repo) FindLinkableStories(ctx context.Context, req RepoFindLinkableStoriesReq) ([]linkableStoryRow, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, nil
	}
	includeDemand := req.DemandID > 0 && req.ProductID > 0
	if req.ExecutionID == 0 && !includeDemand {
		return nil, 0, nil
	}

	base := func() *gorm.DB {
		var q *gorm.DB
		switch {
		case !includeDemand:
			q = r.db.WithContext(ctx).Table("zt_projectstory AS ps").
				Joins("INNER JOIN zt_story AS s ON ps.story = s.id").
				Joins("INNER JOIN zt_product AS p ON s.product = p.id").
				Where("ps.project = ?", req.ExecutionID).
				Where("s.deleted = ? AND p.deleted = ?", "0", "0").
				Where("s.type = ?", "story")
			if req.ProductID > 0 {
				q = q.Where("ps.product = ?", req.ProductID)
			}
		case req.ExecutionID > 0:
			idSub := r.db.Raw(`
SELECT ps.story AS id FROM zt_projectstory AS ps
WHERE ps.project = ? AND ps.product = ?
UNION
SELECT d.id FROM zt_story AS d
WHERE d.fromDemand = ? AND d.product = ? AND d.isParent = ? AND d.deleted = ? AND d.type = ?`,
				req.ExecutionID, req.ProductID,
				req.DemandID, req.ProductID, "0", "0", "story")
			q = r.db.WithContext(ctx).Table("zt_story AS s").
				Joins("INNER JOIN zt_product AS p ON s.product = p.id").
				Where("s.deleted = ? AND p.deleted = ?", "0", "0").
				Where("s.type = ?", "story").
				Where("s.id IN (?)", idSub)
		default:
			q = r.db.WithContext(ctx).Table("zt_story AS s").
				Joins("INNER JOIN zt_product AS p ON s.product = p.id").
				Where("s.deleted = ? AND p.deleted = ?", "0", "0").
				Where("s.type = ?", "story").
				Where("s.fromDemand = ? AND s.product = ? AND s.isParent = ?", req.DemandID, req.ProductID, "0")
		}
		return applyLinkableScope(q, req)
	}

	var total int64
	if err := base().Distinct("s.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []linkableStoryRow
	err := base().Select(linkableStorySelect).
		Order("s.id DESC").
		Limit(req.Limit).
		Offset(req.Offset).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// applyLinkableScope 分支与已排除 ID。分支为空不过滤；有分支时含主干 0。
func applyLinkableScope(q *gorm.DB, req RepoFindLinkableStoriesReq) *gorm.DB {
	branchIDs := branchFilterIDs(req.BranchCSV)
	if len(branchIDs) > 0 {
		q = q.Where("s.branch IN ?", branchIDs)
	}
	if len(req.ExcludeIDs) > 0 {
		q = q.Where("s.id NOT IN ?", req.ExcludeIDs)
	}
	return q
}

// branchFilterIDs 对齐禅道 byBranch：包含主干 0 + 版本 branch；空串表示不过滤。
func branchFilterIDs(branchCSV string) []uint {
	branchCSV = strings.TrimSpace(branchCSV)
	if branchCSV == "" {
		return nil
	}
	parts := strings.Split(branchCSV, ",")
	seen := map[uint]struct{}{0: {}}
	out := []uint{0}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			continue
		}
		id := uint(n)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// BuildExcludeIDList 合并父需求与已关联需求（对齐 getExcludeStoryIdList）。
func BuildExcludeIDList(parentIDs, linkedIDs []uint) []uint {
	seen := make(map[uint]struct{}, len(parentIDs)+len(linkedIDs))
	out := make([]uint, 0, len(parentIDs)+len(linkedIDs))
	appendUnique := func(id uint) {
		if id == 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range parentIDs {
		appendUnique(id)
	}
	for _, id := range linkedIDs {
		appendUnique(id)
	}
	return out
}
