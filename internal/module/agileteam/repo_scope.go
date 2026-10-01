// =============================================================================
// 文件: internal/module/agileteam/repo_scope.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 团队视角可见范围（按团队 / 按部室）。
// 依赖: 无
// =============================================================================

package agileteam

import (
	"context"
	"html"
	"regexp"
	"strings"

	"workbench/internal/constants"
)

// HasActiveRole 仅认可未删除的用户角色关联与启用且未删除的角色，
// 避免失效角色关联继续授予全局看板范围。
func (r *Repo) HasActiveRole(ctx context.Context, userID int64, roleCode string) (bool, error) {
	var count int64
	err := r.read().WithContext(ctx).Raw(`SELECT COUNT(*) FROM zt_gf_user_roles ur JOIN zt_roles r ON r.id=ur.roleId
WHERE ur.userId=? AND ur.deleted='0' AND r.deleted='0' AND r.isActive=1 AND r.code=?`, userID, roleCode).Scan(&count).Error
	return count > 0, err
}

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

// ListManagedTeamgroupIDs 返回登记该账号为敏捷教练的团队范围。
// 父级教练覆盖父级及直接子级；仅登记在子级的教练只获得该子级。
func (r *Repo) ListManagedTeamgroupIDs(ctx context.Context, account string) ([]uint, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return []uint{}, nil
	}
	var ids []uint
	err := r.read().WithContext(ctx).Raw(`
SELECT DISTINCT scoped.id
FROM zt_teamgroup managed
INNER JOIN zt_teamgroup scoped ON scoped.deleted = '0'
  AND (scoped.id = managed.id OR ((managed.type = 'parent' OR managed.parent = 0) AND scoped.parent = managed.id))
WHERE managed.deleted = '0' AND COALESCE(managed.manager, '') REGEXP ?
ORDER BY scoped.id ASC`, accountTokenRegexp(account)).Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint{}
	}
	return ids, nil
}

func accountTokenRegexp(account string) string {
	return `(^|[[:space:],;])` + regexp.QuoteMeta(strings.TrimSpace(account)) + `([[:space:],;]|$)`
}

// IsDeptManager 判断账号是否为某部室负责人。
// 规则与 ListDeptTreeIDs 保持完全一致：优先查补缺表（deleted='0'），有记录以补缺表为准；
// 没有记录再看 zt_dept.manager；仅对 constants.DeptTechHQID 及以下部门生效。
func (r *Repo) IsDeptManager(ctx context.Context, account string) (bool, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return false, nil
	}
	ids, err := r.ListDeptTreeIDs(ctx, account)
	if err != nil {
		return false, err
	}
	return len(ids) > 0, nil
}

// ListDeptTreeIDs 返回账号负责的部门及其下级部门 ID。
// 针对 constants.DeptTechHQID 科技部本部及其子孙部门，优先查补缺表
// zt_wb_dept_manager_override；未命中 fallback 到 zt_dept.manager。
func (r *Repo) ListDeptTreeIDs(ctx context.Context, account string) ([]uint, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return nil, nil
	}
	type managedDept struct {
		ID   uint   `gorm:"column:id"`
		Path string `gorm:"column:path"`
	}
	var managed []managedDept
	if err := r.read().WithContext(ctx).Raw(`
SELECT d.id, COALESCE(d.path, '') AS path FROM zt_dept d
LEFT JOIN zt_wb_dept_manager_override o
  ON o.dept = d.id AND o.deleted = '0' AND (d.id = ? OR d.path LIKE ?)
WHERE COALESCE(
  CASE
    WHEN (d.id = ? OR d.path LIKE ?) AND o.id IS NOT NULL THEN o.account
    ELSE d.manager
  END, ''
) REGEXP ? ORDER BY d.id ASC`,
		constants.DeptTechHQID, constants.DeptTechHQPathPattern(),
		constants.DeptTechHQID, constants.DeptTechHQPathPattern(),
		accountTokenRegexp(account)).Scan(&managed).Error; err != nil {
		return nil, err
	}
	if len(managed) == 0 {
		return []uint{}, nil
	}
	deptIDs := make([]uint, 0, len(managed))
	query := `SELECT DISTINCT id FROM zt_dept WHERE id IN ?`
	args := []interface{}{make([]uint, 0, len(managed))}
	for _, dept := range managed {
		deptIDs = append(deptIDs, dept.ID)
		if path := strings.TrimSpace(dept.Path); path != "" {
			query += ` OR path LIKE ?`
			args = append(args, path+"%")
		}
	}
	args[0] = deptIDs
	query += ` ORDER BY id ASC`
	var ids []uint
	if err := r.read().WithContext(ctx).Raw(query, args...).Scan(&ids).Error; err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint{}
	}
	return ids, nil
}

// ListMappedTeamgroupIDsByDepts 返回指定禅道部门范围内有正式挂靠的敏捷团队。
// 父级挂靠覆盖其直接子级；子级挂靠只授权该子级，不扩展到兄弟小组。
func (r *Repo) ListMappedTeamgroupIDsByDepts(ctx context.Context, deptIDs []uint) ([]uint, error) {
	if len(deptIDs) == 0 {
		return []uint{}, nil
	}
	var ids []uint
	err := r.read().WithContext(ctx).Raw(`
SELECT DISTINCT scoped.id
FROM zt_wb_agileteam_orgmap m
INNER JOIN zt_teamgroup mapped ON mapped.id = m.teamgroupId AND mapped.deleted = '0'
INNER JOIN zt_teamgroup scoped ON scoped.deleted = '0'
  AND (scoped.id = mapped.id OR ((mapped.type = 'parent' OR mapped.parent = 0) AND scoped.parent = mapped.id))
WHERE m.status = 'active' AND m.deptId IN ?
ORDER BY scoped.id ASC`, deptIDs).Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint{}
	}
	return ids, nil
}

// ListTeamgroupOptionsByIDs 按授权 ID 返回团队/小组选项。
func (r *Repo) ListTeamgroupOptionsByIDs(ctx context.Context, ids []uint) ([]ScopeOption, error) {
	if len(ids) == 0 {
		return []ScopeOption{}, nil
	}
	var rows []ScopeOption
	err := r.read().WithContext(ctx).Raw(`
SELECT id, name,
       CASE WHEN type = 'parent' OR parent = 0 THEN 'team' ELSE 'subteam' END AS type
FROM zt_teamgroup
WHERE deleted = '0' AND id IN ?
ORDER BY id ASC`, ids).Scan(&rows).Error
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

// ListOrgTeamOptions 返回禅道组织树中的最底层部门，作为组织团队挂靠候选。
func (r *Repo) ListOrgTeamOptions(ctx context.Context) ([]ScopeOption, error) {
	var rows []ScopeOption
	err := r.read().WithContext(ctx).Raw(`
SELECT d.id, d.name
FROM zt_dept d
WHERE NOT EXISTS (SELECT 1 FROM zt_dept child WHERE child.parent = d.id)
ORDER BY d.id ASC`).Scan(&rows).Error
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
SELECT account, COALESCE(NULLIF(realname, ''), account) AS name, pinyin
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

// SearchPeerTeamUsers 返回当前小组及同一父级下兄弟小组的成员。
// 空 q 用于团队维护弹窗的默认候选列表，非空 q 仍支持姓名、账号和禅道拼音搜索。
func (r *Repo) SearchPeerTeamUsers(ctx context.Context, teamgroupID uint, q string, limit int) ([]CandidateItem, error) {
	if teamgroupID == 0 {
		return []CandidateItem{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q = strings.TrimSpace(q)
	like := "%" + q + "%"
	var rows []CandidateItem
	err := r.read().WithContext(ctx).Raw(`
SELECT DISTINCT t.account, COALESCE(NULLIF(u.realname, ''), t.account) AS name, u.pinyin
FROM zt_teamgroup current_tg
INNER JOIN zt_teamgroup peer_tg
  ON peer_tg.deleted = '0'
 AND ((current_tg.parent <> 0 AND peer_tg.parent = current_tg.parent)
      OR (current_tg.parent = 0 AND peer_tg.id = current_tg.id))
INNER JOIN zt_team t ON t.root = peer_tg.id AND t.type = 'teamgroup' AND t.account <> ''
INNER JOIN zt_user u ON u.account = t.account AND u.deleted = '0'
WHERE current_tg.id = ? AND current_tg.deleted = '0'
  AND (? = '' OR t.account LIKE ? OR u.realname LIKE ? OR u.pinyin LIKE ?)
ORDER BY t.account ASC
LIMIT ?`, teamgroupID, q, like, like, like, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []CandidateItem{}
	}
	return rows, nil
}
