// =============================================================================
// 文件: internal/module/po/repo_primaryaction_rel.go
// 模块: PO 工作台
// 类型: repo
// 职责: 主操作派生所需的对象级关系批量判定。排期按钮必须「有排期能力 + 与该需求有关系」。
//       关系口径直接复用 demandauthz（超管 / PMO / 个人干系人 / 团队长管辖），
//       本文件只做批量封装，不另立一套关系定义。
// =============================================================================

package po

import (
	"context"

	"workbench/internal/model"
)

// DemandRelationBatch 是「actor 对一批需求是否有对象级关系」的批量结果。
type DemandRelationBatch struct {
	// Related 命中集合：超管、PMO、个人干系人、团队长管辖任一成立即为 true。
	Related map[uint]bool
}

// HasRelated 返回 actor 对该需求是否有对象级关系。
func (b *DemandRelationBatch) HasRelated(demandID uint) bool {
	if b == nil || b.Related == nil {
		return false
	}
	return b.Related[demandID]
}

// BatchDemandRelations 批量判定 actor 对 demandIDs 的对象级关系。
//
// 口径与 demandauthz.Repo.Evaluate 完全一致（超管 → PMO → 个人干系人 → 团队长管辖），
// 但把「管辖部门树」压缩成每次调用算一次，再对整批需求做一次集合查询，
// 避免按需求逐行 Evaluate 造成 N+1。
func (r *Repo) BatchDemandRelations(ctx context.Context, actor *model.User, demandIDs []uint) (*DemandRelationBatch, error) {
	batch := &DemandRelationBatch{Related: make(map[uint]bool, len(demandIDs))}
	if r == nil || r.db == nil || actor == nil || len(demandIDs) == 0 {
		return batch, nil
	}
	for _, id := range demandIDs {
		if id != 0 {
			batch.Related[id] = false
		}
	}
	// 超管对全部需求都有关系，直接全置真，避免多余查询。
	if actor.IsSuperAdmin {
		for id := range batch.Related {
			batch.Related[id] = true
		}
		return batch, nil
	}
	authz := newDemandAuthzRepo(r.db)
	if authz == nil {
		return batch, nil
	}
	if isPMO, err := authz.IsPMORole(ctx, actor.ID); err == nil && isPMO {
		for id := range batch.Related {
			batch.Related[id] = true
		}
		return batch, nil
	}
	// 个人干系人：一条批量 IN 查询覆盖整批，关系列与 demandauthz.CheckDemandVisibility 一致。
	direct, err := r.DemandIDsRelatedToAccount(ctx, demandIDs, actor.Account)
	if err != nil {
		return nil, err
	}
	for _, id := range direct {
		batch.Related[id] = true
	}
	// 团队长管辖：先取管辖部门树（每次调用算一次），再批量筛出落在树内的需求。
	deptIDs, err := authz.ListLeaderDeptTreeIDs(ctx, actor.Account)
	if err != nil {
		return nil, err
	}
	if len(deptIDs) == 0 {
		return batch, nil
	}
	managed, err := r.DemandIDsManagedByDepts(ctx, demandIDs, deptIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range managed {
		batch.Related[id] = true
	}
	return batch, nil
}
