// =============================================================================
// 文件: internal/module/schedule/repo_story_link.go
// 模块: 排期工作台
// 类型: repo
// 职责: 创建/编辑任务时把研发需求关联到项目和执行（zt_projectstory + zt_action）。
// 依赖: internal/module/schedule/repo.go (Repo)
//       internal/module/schedule/repo_scheduling_write.go (CreateAction)
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"fmt"
)

// ZtProjectStoryInsert 用于向 zt_projectstory 插入项目-需求关联。
type ZtProjectStoryInsert struct {
	Project uint
	Product uint
	Branch  string
	Story   uint
	Version int
	Order   int
}

// GetStoryBranchAndVersion 查研发需求的 branch 和 version。
// 等价于禅道 story/model.php:3140 getVersions（select id,version from zt_story）。
func (r *Repo) GetStoryBranchAndVersion(ctx context.Context, storyID uint) (string, int, error) {
	if storyID == 0 {
		return "", 0, errors.New("需求 ID 无效")
	}
	var row struct {
		Branch  string `gorm:"column:branch"`
		Version int    `gorm:"column:version"`
	}
	const query = `SELECT branch, version FROM zt_story WHERE id = ? AND deleted = '0' LIMIT 1`
	if err := r.db.WithContext(ctx).Raw(query, storyID).Scan(&row).Error; err != nil {
		return "", 0, err
	}
	return row.Branch, row.Version, nil
}

// ProjectStoryExists 判断 zt_projectstory 是否已存在该 (project, story) 关联。
// 对应禅道 execution/model.php:2849 的存在性守卫 isset($linkedStories[$storyID])。
func (r *Repo) ProjectStoryExists(ctx context.Context, projectID, storyID uint) (bool, error) {
	var row struct {
		OK int `gorm:"column:ok"`
	}
	const query = `SELECT 1 AS ok FROM zt_projectstory WHERE project = ? AND story = ? LIMIT 1`
	if err := r.db.WithContext(ctx).Raw(query, projectID, storyID).Scan(&row).Error; err != nil {
		return false, err
	}
	return row.OK == 1, nil
}

// GetProjectStoryMaxOrder 取 zt_projectstory 在某 project 下的最大 order。
// 对应禅道 execution/model.php:2867 的 $lastOrder 累加逻辑。
func (r *Repo) GetProjectStoryMaxOrder(ctx context.Context, projectID uint) (int, error) {
	var row struct {
		MaxOrder int `gorm:"column:maxOrder"`
	}
	const query = "SELECT COALESCE(MAX(`order`), 0) AS maxOrder FROM zt_projectstory WHERE project = ?"
	if err := r.db.WithContext(ctx).Raw(query, projectID).Scan(&row).Error; err != nil {
		return 0, err
	}
	return row.MaxOrder, nil
}

// ReplaceProjectStory 插入或更新 zt_projectstory。
// 用 ON DUPLICATE KEY UPDATE 而非 REPLACE INTO —— csrcb20 为 zt_projectstory 加了
// PKID 自增主键，REPLACE 会消耗自增值并打乱顺序。
func (r *Repo) ReplaceProjectStory(ctx context.Context, row *ZtProjectStoryInsert) error {
	if row == nil {
		return errors.New("project story row is nil")
	}
	const query = "INSERT INTO zt_projectstory (project, product, branch, story, version, `order`) VALUES (?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE product = VALUES(product), branch = VALUES(branch), version = VALUES(version), `order` = VALUES(`order`)"
	return r.db.WithContext(ctx).Exec(query, row.Project, row.Product, row.Branch, row.Story, row.Version, row.Order).Error
}

// LinkStoryToProjectAndExecution 把研发需求关联到项目与执行。
//   - 项目关联：projectID > 0 时，写一条 zt_projectstory(project=projectID)。
//   - 执行关联：executionID > 0 且与 projectID 不同时，再写一条 zt_projectstory(project=executionID)。
//   - M1 存在性守卫：仅在该 (project, story) 首次建立关联时写 zt_action，
//     避免 edit 反复保存导致 action 刷屏（对齐禅道 execution/model.php:2849）。
//   - action 名使用小写 "linked2project"/"linked2execution"，对齐禅道 lang key。
//
// 原生任务调用已提交；本地关联失败由调用方提示部分完成。
func (r *Repo) LinkStoryToProjectAndExecution(ctx context.Context, storyID, productID, projectID, executionID uint, account string) error {
	if storyID == 0 {
		return errors.New("需求 ID 无效")
	}

	branch, version, err := r.GetStoryBranchAndVersion(ctx, storyID)
	if err != nil {
		return fmt.Errorf("get story branch and version: %w", err)
	}

	links := []struct {
		id        uint
		action    string
		execution uint
	}{{projectID, "linked2project", 0}, {executionID, "linked2execution", executionID}}
	for i, link := range links {
		if link.id == 0 || (i == 1 && executionID == projectID) {
			continue
		}
		existed, err := r.ProjectStoryExists(ctx, link.id, storyID)
		if err != nil {
			return err
		}
		order, err := r.GetProjectStoryMaxOrder(ctx, link.id)
		if err != nil {
			return err
		}
		if err := r.ReplaceProjectStory(ctx, &ZtProjectStoryInsert{
			Project: link.id, Product: productID, Branch: branch,
			Story: storyID, Version: version, Order: order + 1,
		}); err != nil {
			return err
		}
		if !existed {
			if err := r.CreateAction(ctx, "story", storyID, link.action, account, productID, projectID, link.execution, fmt.Sprint(link.id)); err != nil {
				return err
			}
		}
	}

	return nil
}

// =============================================================================
// story ↔ plan 关联（zt_planstory + zt_action）
// 用于排期创建/编辑时把研发需求关联到计划，并补写操作历史。
// 对齐 LinkStoryToProjectAndExecution 的 M1 存在性守卫：仅首次关联/解除断言时
// 写 action，避免 edit 反复保存导致 action 刷屏。
// =============================================================================

// PlanStoryExists 判断 zt_planstory 是否已存在该 (plan, story) 关联。
func (r *Repo) PlanStoryExists(ctx context.Context, planID, storyID uint) (bool, error) {
	var row struct {
		OK int `gorm:"column:ok"`
	}
	const query = `SELECT 1 AS ok FROM zt_planstory WHERE plan = ? AND story = ? LIMIT 1`
	if err := r.db.WithContext(ctx).Raw(query, planID, storyID).Scan(&row).Error; err != nil {
		return false, err
	}
	return row.OK == 1, nil
}

// ListOtherPlansOfStory 查询 story 所关联的「除 keepPlanID 外」的所有 planID。
// 用于 edit plan 场景：摘除 story 在其他计划中的关联。
func (r *Repo) ListOtherPlansOfStory(ctx context.Context, storyID, keepPlanID uint) ([]uint, error) {
	var planIDs []uint
	const query = `SELECT plan FROM zt_planstory WHERE story = ? AND plan <> ?`
	if err := r.db.WithContext(ctx).Raw(query, storyID, keepPlanID).Scan(&planIDs).Error; err != nil {
		return nil, err
	}
	return planIDs, nil
}

// EnsurePlanStoryRelation 幂等地把 story 关联到 plan。
// 用 INSERT IGNORE 应对 (plan, story) 复合主键冲突，已存在则不报错。
// `order` 是 MySQL 保留字，必须反引号。
func (r *Repo) EnsurePlanStoryRelation(ctx context.Context, planID, storyID uint) error {
	if planID == 0 || storyID == 0 {
		return errors.New("plan or story id is invalid")
	}
	const query = "INSERT IGNORE INTO zt_planstory (plan, story, `order`) VALUES (?, ?, 0)"
	return r.db.WithContext(ctx).Exec(query, planID, storyID).Error
}

// LinkStoryToPlan 把研发需求关联到计划。
//   - 幂等保证：先 EnsurePlanStoryRelation（INSERT IGNORE）。
//   - M1 存在性守卫：仅在该 (plan, story) 首次建立关联时写 zt_action，
//     避免 edit 反复保存导致 action 刷屏（对齐 LinkStoryToProjectAndExecution）。
//   - productID 传 story 的 product（本系统 plan 由 resolvePlanForProduct 按 product
//     创建，plan.product == story.product）。
//
// 注意：调用方需在事务内调用（txRepo），与排期创建/更新同事务，失败回滚。
func (r *Repo) LinkStoryToPlan(ctx context.Context, storyID, productID, planID uint, account string) error {
	if storyID == 0 {
		return errors.New("需求 ID 无效")
	}

	existed, err := r.PlanStoryExists(ctx, planID, storyID)
	if err != nil {
		return fmt.Errorf("check plan story exists: %w", err)
	}

	if err := r.EnsurePlanStoryRelation(ctx, planID, storyID); err != nil {
		return fmt.Errorf("ensure plan story relation: %w", err)
	}

	// M1 存在性守卫：仅首次关联写 action，避免 edit 刷屏。
	if !existed {
		if err := r.CreateAction(ctx, "story", storyID, "linked2plan", account, productID, 0, 0, fmt.Sprintf("%d", planID)); err != nil {
			return fmt.Errorf("create linked2plan action: %w", err)
		}
		if err := r.CreateAction(ctx, "productplan", planID, "linkstory", account, productID, 0, 0, fmt.Sprintf("%d", storyID)); err != nil {
			return fmt.Errorf("create linkstory action: %w", err)
		}
	}

	return nil
}
