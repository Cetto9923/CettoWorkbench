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
	// 超管 / PMO 对全部需求都有关系，直接全置真并返回，避免多余查询。
	orgWide, err := r.isOrgWideRelated(ctx, actor)
	if err != nil {
		return nil, err
	}
	if orgWide {
		batch.markAll()
		return batch, nil
	}
	if err := r.markDirectRelated(ctx, batch, demandIDs, actor.Account); err != nil {
		return nil, err
	}
	if err := r.markLeaderManaged(ctx, batch, demandIDs, actor.Account); err != nil {
		return nil, err
	}
	return batch, nil
}

// markAll 把整批标记为有关系，供超管 / PMO 提前返回时使用。
func (b *DemandRelationBatch) markAll() {
	for id := range b.Related {
		b.Related[id] = true
	}
}

// isOrgWideRelated 判定 actor 是否对全部需求都有关系：超管直接成立；
// PMO 通过 demandauthz 既有判定（1 条 SQL）。
func (r *Repo) isOrgWideRelated(ctx context.Context, actor *model.User) (bool, error) {
	if actor.IsSuperAdmin {
		return true, nil
	}
	authz := newDemandAuthzRepo(r.db)
	if authz == nil {
		return false, nil
	}
	isPMO, err := authz.IsPMORole(ctx, actor.ID)
	if err != nil {
		return false, err
	}
	return isPMO, nil
}

// markDirectRelated 标记「本人直接相关」的需求：一条批量 IN 查询覆盖整批，
// 关系列与 demandauthz.CheckDemandVisibility 一致。
func (r *Repo) markDirectRelated(ctx context.Context, batch *DemandRelationBatch, demandIDs []uint, account string) error {
	direct, err := r.DemandIDsRelatedToAccount(ctx, demandIDs, account)
	if err != nil {
		return err
	}
	for _, id := range direct {
		batch.Related[id] = true
	}
	return nil
}

// markLeaderManaged 标记「团队长管辖部门内」的需求：管辖部门树每次调用算一次，
// 再用一条批量查询筛出落在树内的需求。
func (r *Repo) markLeaderManaged(ctx context.Context, batch *DemandRelationBatch, demandIDs []uint, account string) error {
	authz := newDemandAuthzRepo(r.db)
	if authz == nil {
		return nil
	}
	deptIDs, err := authz.ListLeaderDeptTreeIDs(ctx, account)
	if err != nil {
		return err
	}
	if len(deptIDs) == 0 {
		return nil
	}
	managed, err := r.DemandIDsManagedByDepts(ctx, demandIDs, deptIDs)
	if err != nil {
		return err
	}
	for _, id := range managed {
		batch.Related[id] = true
	}
	return nil
}
