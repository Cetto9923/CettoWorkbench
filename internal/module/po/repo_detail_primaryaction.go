// Primary-action facts shared by demand detail and list services.
package po

import (
	"context"
	"strings"
)

// StoryTestTaskCountRow 研发需求关联的测试单张数与首选 ID。
// Stage 5 §4-4：test_link 必须消费 zt_testtask join 真实事实。
type StoryTestTaskCountRow struct {
	StoryID uint `gorm:"column:story"`
	Count   int  `gorm:"column:count"`
	FirstID uint `gorm:"column:first_id"`
}

// DemandPrimaryActionRow 是 primaryAction 派生所需的最小业务需求事实投影。
// 单行详情与列表共用：列表场景走 FindDemandsPrimaryActions 批量 IN 查询。
//
// 字段选择依据：
//   - accepter / assignedTo：验收与受理阶段授权判断的事实基础。
//   - stage / status：派生价值流阶段 key。
//   - parent / fromDemandExists：与排期、提测路由相关；当前 primaryAction 不直接消费。
type DemandPrimaryActionRow struct {
	ID         uint   `gorm:"column:id"`
	Stage      string `gorm:"column:stage"`
	Status     string `gorm:"column:status"`
	AssignedTo string `gorm:"column:assignedTo"`
	Accepter   string `gorm:"column:accepter"`
	CreatedBy  string `gorm:"column:createdBy"`
}

// FindDemandPrimaryActions 批量查询一批业务需求的 primaryAction 事实。
//
// 入参 ids 为业务需求 ID（来自 /demands 列表或 /board/demand 行）。
// IN (?) 一次性查询，避免行内 N+1。空切片直接返回空 map。
func (r *DemandDetailRepo) FindDemandPrimaryActions(ctx context.Context, ids []uint) (map[uint]DemandPrimaryActionRow, error) {
	out := make(map[uint]DemandPrimaryActionRow, len(ids))
	if r == nil || r.db == nil || len(ids) == 0 {
		return out, nil
	}
	var rows []DemandPrimaryActionRow
	err := r.db.WithContext(ctx).Raw(`
SELECT id, stage, status, assignedTo, accepter, createdBy
FROM zt_demand
WHERE id IN ? AND deleted = '0'`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	// 默认填 0 值避免调用方 nil 检查。
	for _, id := range ids {
		if _, ok := out[id]; !ok {
			out[id] = DemandPrimaryActionRow{ID: id}
		}
	}
	return out, nil
}

// DemandEvaluateStatusRow 业务需求的评价状态聚合（Stage 5 §4-3 / PLAN §4-3 评价统计口径）。
//
//   - HasPendingEvaluateForAccount：当前账号在该需求下存在 zt_demandappraise
//     且 appraiseBy = account 但 appraiseTime 为空 / 0 / 0000-00-00（"未评任务"）。
//   - HasAnyEvaluate：任何评价存在 appraiseTime（"已有评价可读"）。
//
// 注意：本函数不修改 release 阶段计数（PLAN §4-3 明示禁止）；只读取事实用于
// primaryAction.Enabled 派生，调用方按 Input.HasPendingEvaluateTask /
//
//	Input.HasHistoricalEvaluate 传入。
type DemandEvaluateStatusRow struct {
	DemandID                     uint `gorm:"column:demand_id"`
	HasPendingEvaluateForAccount bool `gorm:"column:has_pending"`
	HasAnyEvaluate               bool `gorm:"column:has_any"`
}

// FindDemandEvaluateStatus 批量查询一批业务需求的评价状态。
//
// 入参 demandIDs 为当前页业务需求 ID；account 为当前用户账号。
//
// SQL：
//
//	LEFT JOIN zt_demandappraise：appraiseTime IS NULL 或 = '0000-00-00'
//	  且 appraiseBy = account → "本账号有未评"。
//	appraiseTime IS NOT NULL 且 != '0000-00-00' → "任一已评"。
func (r *DemandDetailRepo) FindDemandEvaluateStatus(ctx context.Context, account string, demandIDs []uint) (map[uint]DemandEvaluateStatusRow, error) {
	out := make(map[uint]DemandEvaluateStatusRow, len(demandIDs))
	if r == nil || r.db == nil || len(demandIDs) == 0 {
		return out, nil
	}
	if strings.TrimSpace(account) == "" {
		for _, id := range demandIDs {
			out[id] = DemandEvaluateStatusRow{DemandID: id}
		}
		return out, nil
	}
	var rows []DemandEvaluateStatusRow
	err := r.db.WithContext(ctx).Raw(`
SELECT d.id AS demand_id,
       EXISTS(
         SELECT 1 FROM zt_demandappraise da
         WHERE da.demand = d.id
           AND da.appraiseBy = ?
           AND (da.appraiseTime IS NULL OR da.appraiseTime = '0000-00-00')
       ) AS has_pending,
       EXISTS(
         SELECT 1 FROM zt_demandappraise da
         WHERE da.demand = d.id
           AND da.appraiseTime IS NOT NULL
           AND da.appraiseTime <> '0000-00-00'
       ) AS has_any
FROM zt_demand d
WHERE d.id IN ? AND d.deleted = '0'`, account, demandIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.DemandID] = row
	}
	for _, id := range demandIDs {
		if _, ok := out[id]; !ok {
			out[id] = DemandEvaluateStatusRow{DemandID: id}
		}
	}
	return out, nil
}

// DemandTestTaskCountRow 业务需求维度的测试单统计（Stage 5 §4-4）。
//
// 业务需求不直接关联测试单，需经 zt_story.fromDemand 找到故事，再通过
// zt_testrun + zt_case.story 找到测试单。本聚合在业务需求视角给出
// "需求 → 故事 → 测试单" 的真实张数 + 首选测试单 ID，供 primaryAction.test_link 阶段消费。
//
// DemandID = 0 时表示该需求无任何故事 / 测试单。
type DemandTestTaskCountRow struct {
	DemandID uint `gorm:"column:demand_id"`
	Count    int  `gorm:"column:count"`
	FirstID  uint `gorm:"column:first_id"`
}

// CountDemandTestTasks 批量统计每个业务需求关联的测试单张数与首选 ID。
//
// 一次性 IN (?) 查询，避免行内 N+1。空切片直接返回空 map。
func (r *DemandDetailRepo) CountDemandTestTasks(ctx context.Context, demandIDs []uint) (map[uint]DemandTestTaskCountRow, error) {
	out := make(map[uint]DemandTestTaskCountRow, len(demandIDs))
	if r == nil || r.db == nil || len(demandIDs) == 0 {
		return out, nil
	}
	var rows []DemandTestTaskCountRow
	err := r.db.WithContext(ctx).Raw(`
SELECT d.id AS demand_id,
       COALESCE(tt_summary.cnt, 0) AS count,
       COALESCE(tt_summary.first_id, 0) AS first_id
FROM (
  SELECT id FROM zt_demand WHERE id IN ? AND deleted = '0'
) d
LEFT JOIN (
  SELECT s.fromDemand AS demand_id,
         COUNT(DISTINCT tt.id) AS cnt,
         SUBSTRING_INDEX(GROUP_CONCAT(DISTINCT tt.id ORDER BY tt.id DESC), ',', 1) AS first_id
  FROM zt_testtask tt
  JOIN zt_testrun tr ON tr.task = tt.id
  JOIN zt_case c ON c.id = tr.case
  JOIN zt_story s ON s.id = c.story AND s.deleted = '0'
  WHERE s.fromDemand IN ? AND tt.deleted = '0'
  GROUP BY s.fromDemand
) tt_summary ON tt_summary.demand_id = d.id`, demandIDs, demandIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.DemandID] = row
	}
	for _, id := range demandIDs {
		if _, ok := out[id]; !ok {
			out[id] = DemandTestTaskCountRow{DemandID: id, Count: 0, FirstID: 0}
		}
	}
	return out, nil
}

// CountStoryTestTasks 批量统计每个研发需求关联的测试单张数与首选 ID。
//
// SQL：先按 (case.story, testtask.id) 取每个 story 下最新的 testtask.id，
// 再 LEFT JOIN 得到总数（保留 0 测试单的故事）。
//
// 入参 storyIDs 用于 service 层 IN (...) 批量驱动，避免行内 N+1。
func (r *DemandDetailRepo) CountStoryTestTasks(ctx context.Context, storyIDs []uint) (map[uint]StoryTestTaskCountRow, error) {
	out := make(map[uint]StoryTestTaskCountRow, len(storyIDs))
	if len(storyIDs) == 0 {
		return out, nil
	}
	var rows []StoryTestTaskCountRow
	err := r.db.WithContext(ctx).Raw(`
SELECT s.id AS story,
       COALESCE(tt_summary.cnt, 0) AS count,
       COALESCE(tt_summary.first_id, 0) AS first_id
FROM (
  SELECT id FROM zt_story WHERE id IN ? AND deleted = '0'
) s
LEFT JOIN (
  SELECT c.story AS story,
         COUNT(DISTINCT tt.id) AS cnt,
         SUBSTRING_INDEX(GROUP_CONCAT(DISTINCT tt.id ORDER BY tt.id DESC), ',', 1) AS first_id
  FROM zt_testtask tt
  JOIN zt_testrun tr ON tr.task = tt.id
  JOIN zt_case c ON c.id = tr.case
  WHERE c.story IN ? AND tt.deleted = '0'
  GROUP BY c.story
) tt_summary ON tt_summary.story = s.id`, storyIDs, storyIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.StoryID] = row
	}
	// 为没有出现在结果里的 story 也填默认 0，避免调用方再 nil 检查。
	for _, id := range storyIDs {
		if _, ok := out[id]; !ok {
			out[id] = StoryTestTaskCountRow{StoryID: id, Count: 0, FirstID: 0}
		}
	}
	return out, nil
}

// StoryMetaForAction 研发需求主操作派生所需最小列。
type StoryMetaForAction struct {
	ID     uint   `gorm:"column:id"`
	Status string `gorm:"column:status"`
	Stage  string `gorm:"column:stage"`
}

// FindStoryMetaForAction 批量读取故事 status/stage，供 primaryAction 派生（单次 IN，禁止行循环）。
func (r *DemandDetailRepo) FindStoryMetaForAction(ctx context.Context, storyIDs []uint) (map[uint]StoryMetaForAction, error) {
	out := make(map[uint]StoryMetaForAction, len(storyIDs))
	if len(storyIDs) == 0 {
		return out, nil
	}
	var rows []StoryMetaForAction
	err := r.db.WithContext(ctx).
		Table("zt_story").
		Select("id, status, stage").
		Where("id IN ? AND deleted = ?", storyIDs, "0").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}
