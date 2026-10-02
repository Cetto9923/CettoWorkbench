// =============================================================================
// 文件: internal/module/agileteam/repo_basic_atomic.go
// 模块: 敏捷小组治理
// 类型: repository
// 职责: 原子更新敏捷小组基本信息、父子层级及历史记录。
// =============================================================================

package agileteam

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const defaultTeamgroupTable = "zt_teamgroup"

// clauseLockForUpdate 是 gorm 提供的行级 X 锁 clause。
var clauseLockForUpdate = clause.Locking{Strength: "UPDATE"}

var (
	errTeamgroupMissing       = errors.New("agile teamgroup missing")
	errTeamgroupSelfParent    = errors.New("agile teamgroup self parent")
	errTeamgroupParentMissing = errors.New("agile teamgroup parent missing")
	errTeamgroupParentCycle   = errors.New("agile teamgroup parent cycle")
	errTeamgroupCorruptTree   = errors.New("agile teamgroup tree corrupt")
)

// teamgroupLockRow holds the minimum fields needed to lock + compute topology.
type teamgroupLockRow struct {
	ID     uint   `gorm:"column:id"`
	Parent uint   `gorm:"column:parent"`
	Grade  int    `gorm:"column:grade"`
	Path   string `gorm:"column:path"`
}

// teamgroupTable 返回 Repo 当前使用的 teamgroup 表名。
// 默认生产为 zt_teamgroup；测试可通过 WithTeamgroupTable 注入临时表。
func (r *Repo) teamgroupTable() string {
	if r.tableOverride != "" {
		return r.tableOverride
	}
	return defaultTeamgroupTable
}

// WithTeamgroupTable 返回一个使用指定表名的 Repo 副本，用于隔离测试。
// 不修改调用方传入的 Repo。
func (r *Repo) WithTeamgroupTable(name string) *Repo {
	if r == nil {
		return nil
	}
	cp := *r
	if name != "" {
		cp.tableOverride = name
	}
	return &cp
}

// UpdateTeamgroupBasicAtomic 基本保存只锁目标行；拓扑变更沿用一致性快照协议。
func (r *Repo) UpdateTeamgroupBasicAtomic(
	ctx context.Context,
	id uint,
	name, slogan, declaration, logo string,
	parentID *uint,
	history *History,
) error {
	if r == nil || r.db == nil || id == 0 {
		return errTeamgroupMissing
	}
	table := r.teamgroupTable()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var observed teamgroupLockRow
		if err := tx.Table(table).Where("id = ? AND deleted = ?", id, "0").Take(&observed).Error; err != nil {
			return teamgroupRowError(err)
		}
		if parentID == nil || *parentID == observed.Parent {
			var current teamgroupLockRow
			if err := tx.Clauses(clauseLockForUpdate).Table(table).Where("id = ? AND deleted = ?", id, "0").Take(&current).Error; err != nil {
				return teamgroupRowError(err)
			}
			if parentID != nil && *parentID != current.Parent {
				return errors.New("小组层级已变化，请刷新后重试")
			}
			return updateTeamgroupFields(tx, table, id, name, slogan, declaration, logo, history)
		}
		// 第一步：整树一致性锁（升序 id 列表 + IN FOR UPDATE）。
		rows, err := lockActiveTreeOrdered(tx, table)
		if err != nil {
			return err
		}
		if err := validateTreeIntegrity(rows); err != nil {
			return err
		}
		byID := indexTree(rows)
		current, ok := byID[id]
		if !ok {
			return errTeamgroupMissing
		}

		// topology：基于同一份已锁快照计算 + 写回。
		return applyTopologyChange(tx, table, byID, current, name, slogan, declaration, logo, *parentID, history)
	})
}

// applyTopologyChange 在事务内基于已锁定的整树快照计算 + 写回 topology 变更。
//
// 调用方必须已持有整树 X 锁；本函数不再发起新的锁请求，仅做计算与 UPDATE。
func applyTopologyChange(
	tx *gorm.DB,
	table string,
	byID map[uint]teamgroupLockRow,
	current teamgroupLockRow,
	name, slogan, declaration, logo string,
	newParentID uint,
	history *History,
) error {
	if newParentID == current.ID {
		return errTeamgroupSelfParent
	}

	// 计算 current 的新 parent/type/grade/path。
	nextType, nextGrade, nextPath, err := computeRelocation(byID, current.ID, newParentID)
	if err != nil {
		return err
	}

	if err := updateTeamgroupFields(tx, table, current.ID, name, slogan, declaration, logo, nil); err != nil {
		return err
	}
	// 2) parent/type/grade/path。
	res := tx.Table(table).
		Where("id = ? AND deleted = ?", current.ID, "0").
		Updates(map[string]any{
			"parent": newParentID,
			"type":   nextType,
			"grade":  nextGrade,
			"path":   nextPath,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errTeamgroupMissing
	}
	// 3) iterative refresh: 用快照建 parent→children 邻接表，BFS 重算所有子孙的 grade/path。
	if err := refreshDescendantsIterative(tx, table, byID, current.ID, nextPath, nextGrade); err != nil {
		return err
	}
	// 4) 历史。
	if history != nil {
		if history.TeamgroupID == 0 {
			history.TeamgroupID = current.ID
		}
		if err := tx.Create(history).Error; err != nil {
			return err
		}
	}
	return nil
}

func updateTeamgroupFields(tx *gorm.DB, table string, id uint, name, slogan, declaration, logo string, history *History) error {
	res := tx.Table(table).Where("id = ? AND deleted = ?", id, "0").Updates(map[string]any{
		"name": name, "slogan": slogan, "declaration": declaration, "logo": logo,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 || history == nil {
		return nil
	}
	if history.TeamgroupID == 0 {
		history.TeamgroupID = id
	}
	return tx.Create(history).Error
}

// lockActiveTreeOrdered 仅用于拓扑变更；锁顺序按主键固定。
func lockActiveTreeOrdered(tx *gorm.DB, table string) ([]teamgroupLockRow, error) {
	// 第一步：无锁读取 active id 列表（按 id 升序）。
	var ids []uint
	if err := tx.Table(table).
		Select("id").
		Where("deleted = ?", "0").
		Order("id ASC").
		Scan(&ids).Error; err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	// 第二步：按 id 升序逐行 FOR UPDATE。
	// 使用 IN 子句 + ORDER BY 让 InnoDB 按主键顺序加锁，避免全表扫描 next-key gap lock 互锁。
	var rows []teamgroupLockRow
	if err := tx.Table(table).
		Clauses(clauseLockForUpdate).
		Select("id, parent, grade, COALESCE(path, '') AS path").
		Where("id IN ? AND deleted = ?", ids, "0").
		Order("id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func indexTree(rows []teamgroupLockRow) map[uint]teamgroupLockRow {
	m := make(map[uint]teamgroupLockRow, len(rows))
	for _, r := range rows {
		m[r.ID] = r
	}
	return m
}

// validateTreeIntegrity 检查快照中是否存在结构异常：
//   - 任何节点的 parent 指向已删除或不存在节点；
//   - 任何节点是其自身的祖先（结构性 cycle）。
//
// 失败时回滚，fail-closed。
func validateTreeIntegrity(rows []teamgroupLockRow) error {
	byID := indexTree(rows)
	for _, r := range rows {
		if r.Parent == 0 {
			continue
		}
		if _, ok := byID[r.Parent]; !ok {
			return fmt.Errorf("%w: parent %d of node %d not in active tree", errTeamgroupCorruptTree, r.Parent, r.ID)
		}
	}
	// 走 parent 链校验结构性 cycle。
	for _, r := range rows {
		if err := walkAncestors(byID, r.ID, r.Parent); err != nil {
			return err
		}
	}
	return nil
}

// walkAncestors 由 start 向 parent 上溯，验证无结构性 cycle。
func walkAncestors(byID map[uint]teamgroupLockRow, start, parent uint) error {
	visited := make(map[uint]bool)
	cur := parent
	for cur != 0 {
		if visited[cur] {
			return fmt.Errorf("%w: cycle at node %d", errTeamgroupCorruptTree, cur)
		}
		visited[cur] = true
		node, ok := byID[cur]
		if !ok {
			return fmt.Errorf("%w: missing ancestor %d", errTeamgroupCorruptTree, cur)
		}
		cur = node.Parent
	}
	return nil
}

// computeRelocation 基于整树快照计算 current 改挂到 newParentID 后的 type/grade/path。
// 若 newParentID 是 current 的子孙 → 返回 cycle 错误。
// 若 newParentID=0 → 视为 root。
func computeRelocation(byID map[uint]teamgroupLockRow, currentID, newParentID uint) (string, int, string, error) {
	if newParentID == 0 {
		return "parent", 1, fmt.Sprintf(",%d,", currentID), nil
	}
	parent, ok := byID[newParentID]
	if !ok {
		return "", 0, "", errTeamgroupParentMissing
	}
	// current 是否出现在 parent 的 path 中 → parent 是 current 的子孙。
	if containsPathID(parent.Path, currentID) {
		return "", 0, "", errTeamgroupParentCycle
	}
	newPath := parent.Path
	if newPath == "" {
		newPath = fmt.Sprintf(",%d,", parent.ID)
	}
	if newPath[len(newPath)-1] != ',' {
		newPath += ","
	}
	newPath += fmt.Sprintf("%d,", currentID)
	return "child", parent.Grade + 1, newPath, nil
}

// containsPathID 检查 ",id," 形式的 path 字符串中是否包含目标 id。
func containsPathID(path string, id uint) bool {
	needle := fmt.Sprintf(",%d,", id)
	return strings.Contains(path, needle)
}

// refreshDescendantsIterative 用整树快照的 parent→children 邻接表 BFS 重算子孙 grade/path。
// visited 防：邻接表本身在通过 byID 构建时不重复，但若历史数据出现结构性 cycle，
// BFS 也可能再次到达 current 自身或已访问节点 → 视为 corrupt 失败。
func refreshDescendantsIterative(
	tx *gorm.DB,
	table string,
	byID map[uint]teamgroupLockRow,
	rootID uint,
	rootPath string,
	rootGrade int,
) error {
	childrenOf := make(map[uint][]uint, len(byID))
	for _, r := range byID {
		childrenOf[r.Parent] = append(childrenOf[r.Parent], r.ID)
	}

	visited := map[uint]bool{rootID: true}
	type item struct {
		id    uint
		path  string
		grade int
	}
	queue := []item{}
	for _, childID := range childrenOf[rootID] {
		queue = append(queue, item{id: childID, path: rootPath, grade: rootGrade})
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if visited[cur.id] {
			return fmt.Errorf("%w: revisits %d during refresh", errTeamgroupCorruptTree, cur.id)
		}
		visited[cur.id] = true

		childPath := cur.path
		if childPath == "" {
			childPath = fmt.Sprintf(",%d,", rootID)
		}
		if childPath[len(childPath)-1] != ',' {
			childPath += ","
		}
		childPath += fmt.Sprintf("%d,", cur.id)
		childGrade := cur.grade + 1

		res := tx.Table(table).
			Where("id = ? AND deleted = ?", cur.id, "0").
			Updates(map[string]any{
				"grade": childGrade,
				"path":  childPath,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errTeamgroupMissing
		}

		for _, grandchildID := range childrenOf[cur.id] {
			if visited[grandchildID] {
				return fmt.Errorf("%w: revisits %d during refresh", errTeamgroupCorruptTree, grandchildID)
			}
			queue = append(queue, item{id: grandchildID, path: childPath, grade: childGrade})
		}
	}
	return nil
}

func teamgroupRowError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errTeamgroupMissing
	}
	return err
}
