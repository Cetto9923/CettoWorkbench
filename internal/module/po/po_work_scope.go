// =============================================================================
// 文件: internal/module/po/po_work_scope.go
// 模块: PO 工作台
// 类型: readonly
// 职责: POWorkScope 唯一真源 —— 当前 PO 仍具有效业务关系的需求范围。
//       语义（2026-08-30 用户冻结）：
//         纳入：我负责（assignedTo/distributedBy/QD/RD/accepter/clarify.PM 六字段）
//               + 我关注（zt_starinfo 主动 follow）
//               独立研发需求由各页面独立口径处理，不在本 demand 范围。
//         父子去重：存在子业务需求时父退出（父子都有时看子），子需求按各自关系进入。
//         排除：仅历史评论/审批/变更/下游 story/task 上卷形成的参与痕迹。
//       三页面（首页价值流 / 验收交付 / 版本跟进）统一消费本真源，禁止各自拼接范围 SQL。
// 依赖: gorm
// =============================================================================

package po

import (
	"context"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// DemandScopeRole POWorkScope 内需求的归属角色。
type DemandScopeRole struct {
	Owner bool `json:"owner"` // 我负责（六字段命中）
	Watch bool `json:"watch"` // 我关注（主动 follow）
}

// poworkScopeSQL 返回 POWorkScope 的 (id, role) 联合查询。
// 内部对 zt_demand 无表别名依赖，可独立执行；外层仅按 s.id 收敛。
func poworkScopeSQL(account string) (string, []interface{}) {
	const sql = `
SELECT s.id, s.role FROM (
	SELECT d.id, 'owner' AS role FROM zt_demand d
	WHERE d.deleted = '0'
	  AND (
		d.assignedTo = ?
		OR d.distributedBy = ?
		OR d.QD = ?
		OR d.RD = ?
		OR d.accepter = ?
		OR EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = d.id AND dc.PM = ?)
	  )
	  AND NOT EXISTS (SELECT 1 FROM zt_demand child WHERE child.parent = d.id AND child.deleted = '0')
	UNION ALL
	SELECT st.objectID, 'watch' AS role FROM zt_starinfo st
	WHERE st.objectType = 'demand' AND st.account = ? AND st.followed = '1'
	  AND NOT EXISTS (SELECT 1 FROM zt_demand child WHERE child.parent = st.objectID AND child.deleted = '0')
) s WHERE s.id > 0`
	return sql, []interface{}{account, account, account, account, account, account, account}
}

// POWorkScope 返回当前账号 POWorkScope 的 demand id → 归属角色映射。
func POWorkScope(ctx context.Context, db *gorm.DB, account string) (map[uint]DemandScopeRole, error) {
	out := map[uint]DemandScopeRole{}
	if db == nil || strings.TrimSpace(account) == "" {
		return out, nil
	}
	sql, args := poworkScopeSQL(account)
	var rows []struct {
		ID   uint   `gorm:"column:id"`
		Role string `gorm:"column:role"`
	}
	if err := db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		sr := out[row.ID]
		if row.Role == "owner" {
			sr.Owner = true
		} else if row.Role == "watch" {
			sr.Watch = true
		}
		out[row.ID] = sr
	}
	return out, nil
}

// POWorkScopeDemandIDs 返回 POWorkScope 的 demand id 列表（升序），供 IN 子查询 / 列表构建。
func POWorkScopeDemandIDs(ctx context.Context, db *gorm.DB, account string) ([]uint, error) {
	scope, err := POWorkScope(ctx, db, account)
	if err != nil {
		return nil, err
	}
	if len(scope) == 0 {
		return nil, nil
	}
	ids := make([]uint, 0, len(scope))
	for id := range scope {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}
