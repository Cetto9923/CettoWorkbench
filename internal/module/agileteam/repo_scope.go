// =============================================================================
// 文件: internal/module/agileteam/repo_scope.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 团队管理视图可见范围（按团队 / 按部室）。
// 依赖: 无
// =============================================================================

package agileteam

import (
	"context"
	"html"
	"strings"
)

// ListParentTeamOptions 返回可作为父级的小组（type=parent 或 parent=0）。
func (r *Repo) ListParentTeamOptions(ctx context.Context, excludeID uint) ([]ParentOption, error) {
	var rows []ParentOption
	err := r.read().WithContext(ctx).Raw(`
SELECT id, name
FROM zt_teamgroup
WHERE deleted = '0' AND (type = 'parent' OR parent = 0) AND id <> ?
ORDER BY id ASC`, excludeID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []ParentOption{}
	}
	for i := range rows {
		rows[i].Name = html.UnescapeString(rows[i].Name)
	}
	return rows, nil
}

// ListUserParentTeamIDs 当前账号所属小组对应的父级团队 ID。
func (r *Repo) ListUserParentTeamIDs(ctx context.Context, account string) ([]uint, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return nil, nil
	}
	var ids []uint
	err := r.read().WithContext(ctx).Raw(`
SELECT DISTINCT COALESCE(NULLIF(tg.parent, 0), tg.id) AS id
FROM zt_teamgroup tg
LEFT JOIN zt_team t ON t.root = tg.id AND t.type = 'teamgroup' AND t.account = ?
WHERE tg.deleted = '0' AND (
  t.account = ? OR tg.PO = ? OR tg.manager = ?
  OR tg.manager LIKE ? OR tg.manager LIKE ? OR tg.manager LIKE ?
)`, account, account, account, account,
		account+",%", "%,"+account+",%", "%,"+account).Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint{}
	}
	return ids, nil
}

// ListFamilyIDs 返回指定父级及其全部子级 ID。
func (r *Repo) ListFamilyIDs(ctx context.Context, parentIDs []uint) ([]uint, error) {
	if len(parentIDs) == 0 {
		return []uint{}, nil
	}
	var ids []uint
	err := r.read().WithContext(ctx).Raw(`
SELECT id FROM zt_teamgroup
WHERE deleted = '0' AND (id IN ? OR parent IN ?)`, parentIDs, parentIDs).Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint{}
	}
	return ids, nil
}

// IsDeptManager 判断账号是否为某部室负责人（zt_dept.manager）。
func (r *Repo) IsDeptManager(ctx context.Context, account string) (bool, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return false, nil
	}
	var n int64
	err := r.read().WithContext(ctx).Raw(`
SELECT COUNT(*) FROM zt_dept WHERE manager = ?`, account).Scan(&n).Error
	return n > 0, err
}

// ListDeptTreeIDs 返回账号所属部门及其下级部门 ID。
func (r *Repo) ListDeptTreeIDs(ctx context.Context, account string) ([]uint, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return nil, nil
	}
	var deptID uint
	if err := r.read().WithContext(ctx).Raw(`
SELECT dept FROM zt_user WHERE account = ? AND deleted = '0' LIMIT 1`, account).Scan(&deptID).Error; err != nil {
		return nil, err
	}
	if deptID == 0 {
		return []uint{}, nil
	}
	var path string
	if err := r.read().WithContext(ctx).Raw(`
SELECT path FROM zt_dept WHERE id = ? LIMIT 1`, deptID).Scan(&path).Error; err != nil {
		return nil, err
	}
	var ids []uint
	q := r.read().WithContext(ctx).Raw(`SELECT id FROM zt_dept WHERE id = ?`, deptID)
	if strings.TrimSpace(path) != "" {
		q = r.read().WithContext(ctx).Raw(`
SELECT id FROM zt_dept WHERE id = ? OR path LIKE ?`, deptID, strings.TrimSpace(path)+"%")
	}
	if err := q.Scan(&ids).Error; err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint{}
	}
	return ids, nil
}

// ListParentTeamIDsByDepts 部室成员所在敏捷小组对应的父级团队。
func (r *Repo) ListParentTeamIDsByDepts(ctx context.Context, deptIDs []uint) ([]uint, error) {
	if len(deptIDs) == 0 {
		return []uint{}, nil
	}
	var ids []uint
	err := r.read().WithContext(ctx).Raw(`
SELECT DISTINCT COALESCE(NULLIF(tg.parent, 0), tg.id) AS id
FROM zt_teamgroup tg
INNER JOIN zt_team t ON t.root = tg.id AND t.type = 'teamgroup'
INNER JOIN zt_user u ON u.account = t.account AND u.deleted = '0'
WHERE tg.deleted = '0' AND u.dept IN ?`, deptIDs).Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint{}
	}
	return ids, nil
}

// ListParentOptionsByIDs 按 ID 返回父级选项（下拉）。
func (r *Repo) ListParentOptionsByIDs(ctx context.Context, ids []uint) ([]ScopeOption, error) {
	if len(ids) == 0 {
		return []ScopeOption{}, nil
	}
	var rows []ScopeOption
	err := r.read().WithContext(ctx).Raw(`
SELECT id, name FROM zt_teamgroup WHERE deleted = '0' AND id IN ? ORDER BY id ASC`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []ScopeOption{}
	}
	for i := range rows {
		rows[i].Name = html.UnescapeString(rows[i].Name)
	}
	return rows, nil
}

// SearchUsers 按账号 / 姓名 / 拼音模糊搜索用户。
//
// 禅道 zt_user.pinyin（VARCHAR(255)）由禅道原生维护，含真实姓名全拼与首字母（如
// "maoyuzhang(001196) myz0"），直接复用为权威拼音索引源，比前端 pinyin-lite
// 高频字典更完整。匹配按 (account OR realname OR pinyin) 三路 LIKE + ?
// 参数化执行；订单按 account ASC；limit 上限 50。
func (r *Repo) SearchUsers(ctx context.Context, q string, limit int) ([]CandidateItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return []CandidateItem{}, nil
	}
	like := "%" + q + "%"
	var rows []CandidateItem
	err := r.read().WithContext(ctx).Raw(`
SELECT account, COALESCE(NULLIF(realname, ''), account) AS name
FROM zt_user
WHERE deleted = '0' AND (account LIKE ? OR realname LIKE ? OR pinyin LIKE ?)
ORDER BY account ASC
LIMIT ?`, like, like, like, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []CandidateItem{}
	}
	return rows, nil
}
