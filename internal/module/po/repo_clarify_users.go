// =============================================================================
// 文件: internal/module/po/repo_clarify_users.go
// 模块: PO 工作台
// 类型: repo
// 职责: 需求澄清候选人员读取。
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm/clause"
	"workbench/internal/pkg/personlabel"
)

// findFirstLevelDeptIDs 查找操作人所在的一级部门（例如总行-金融科技总部这个级别）及其所有下属部门 ID。
func (r *Repo) findFirstLevelDeptIDs(ctx context.Context, actorAccount string) []uint {
	if strings.TrimSpace(actorAccount) == "" {
		return nil
	}
	db, err := r.reader()
	if err != nil {
		return nil
	}
	var u struct {
		Dept uint `gorm:"column:dept"`
	}
	if err := db.WithContext(ctx).Table("zt_user").
		Select("dept").Where("account = ? AND deleted = '0'", actorAccount).
		Take(&u).Error; err != nil || u.Dept == 0 {
		return nil
	}

	var actorDept struct {
		ID     uint   `gorm:"column:id"`
		Parent uint   `gorm:"column:parent"`
		Path   string `gorm:"column:path"`
		Grade  uint   `gorm:"column:grade"`
	}
	if err := db.WithContext(ctx).Table("zt_dept").
		Select("id, parent, path, grade").Where("id = ?", u.Dept).
		Take(&actorDept).Error; err != nil {
		return nil
	}

	// 禅道 path 如 ",1,5,18,"，找出对应的一级部门（总行下一级，如金融科技总部）。
	parts := strings.Split(strings.Trim(actorDept.Path, ","), ",")
	var targetDeptID uint
	if len(parts) >= 2 {
		if id, parseErr := strconv.ParseUint(parts[1], 10, 32); parseErr == nil {
			targetDeptID = uint(id)
		} else {
			targetDeptID = actorDept.ID
		}
	} else if len(parts) == 1 && parts[0] != "" {
		if id, parseErr := strconv.ParseUint(parts[0], 10, 32); parseErr == nil {
			targetDeptID = uint(id)
		} else {
			targetDeptID = actorDept.ID
		}
	} else {
		targetDeptID = actorDept.ID
	}

	if targetDeptID == 0 {
		return nil
	}

	var targetDept struct {
		ID   uint   `gorm:"column:id"`
		Path string `gorm:"column:path"`
	}
	if err := db.WithContext(ctx).Table("zt_dept").
		Select("id, path").Where("id = ?", targetDeptID).
		Take(&targetDept).Error; err != nil {
		return []uint{targetDeptID}
	}

	var deptIDs []uint
	targetPath := targetDept.Path
	if targetPath == "" {
		targetPath = fmt.Sprintf(",%d,", targetDeptID)
	} else {
		if !strings.HasPrefix(targetPath, ",") {
			targetPath = "," + targetPath
		}
		if !strings.HasSuffix(targetPath, ",") {
			targetPath = targetPath + ","
		}
	}
	_ = db.WithContext(ctx).Table("zt_dept").
		Where("id = ? OR path LIKE ?", targetDeptID, targetPath+"%").
		Pluck("id", &deptIDs).Error
	if len(deptIDs) == 0 {
		return []uint{targetDeptID}
	}
	return deptIDs
}

// FindCandidateUsers 读取全部可选人员（优先把本一级部门排在前面，且工号倒序）。
func (r *Repo) FindCandidateUsers(ctx context.Context, actorAccount string) ([]ClarifyOption, error) {
	db, err := r.reader()
	if err != nil {
		return nil, err
	}
	firstLevelDeptIDs := r.findFirstLevelDeptIDs(ctx, actorAccount)

	query := db.WithContext(ctx).Table("zt_user AS u").
		Joins("LEFT JOIN zt_dept AS d ON d.id = u.dept").
		Where("u.deleted = '0'")
	if len(firstLevelDeptIDs) > 0 {
		query = query.Order(clause.OrderBy{Expression: clause.Expr{
			SQL:  "CASE WHEN u.dept IN ? THEN 0 ELSE 1 END, u.account DESC",
			Vars: []any{firstLevelDeptIDs},
		}})
	} else {
		query = query.Order("u.account DESC")
	}

	var rows []struct {
		Account  string `gorm:"column:account"`
		Realname string `gorm:"column:realname"`
		Dept     string `gorm:"column:dept"`
		Pinyin   string `gorm:"column:pinyin"`
	}
	if err := query.Select("u.account, u.realname, COALESCE(d.name, '') AS dept, u.pinyin").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ClarifyOption, 0, len(rows))
	for _, row := range rows {
		account := strings.TrimSpace(row.Account)
		if account == "" {
			continue
		}
		out = append(out, ClarifyOption{
			Value:  account,
			Label:  personlabel.Format(account, row.Realname),
			Dept:   strings.TrimSpace(row.Dept),
			Pinyin: strings.TrimSpace(row.Pinyin),
		})
	}
	return out, nil
}
