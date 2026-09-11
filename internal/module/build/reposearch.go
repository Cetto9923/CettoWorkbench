// =============================================================================
// 文件: internal/module/build/reposearch.go
// 模块: 版本管理
// 类型: action
// 职责: 关联需求 bySearch 查询及搜索选项（模块/计划/产品类型）。
// 依赖: internal/model/zentao
// =============================================================================

package build

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"gorm.io/gorm"

	ztmodel "workbench/internal/model/zentao"
)

// RepoFindStoriesBySearchReq bySearch 分页查询入参。
type RepoFindStoriesBySearchReq struct {
	ProductID   uint
	BranchCSV   string
	ExcludeIDs  []uint
	UserWhere   string
	UserArgs    []any
	Limit       int
	Offset      int
}

// FindProductType 读取产品类型（normal 等）。
func (r *Repo) FindProductType(ctx context.Context, productID uint) (string, error) {
	if r == nil || r.db == nil || productID == 0 {
		return "normal", nil
	}
	var m ztmodel.ZtProduct
	err := r.db.WithContext(ctx).Select("id", "type").Where("id = ? AND deleted = ?", productID, "0").Limit(1).Take(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "normal", nil
		}
		return "", err
	}
	if strings.TrimSpace(m.Type) == "" {
		return "normal", nil
	}
	return m.Type, nil
}

// FindModuleOptions 产品下 story 模块选项（path 展示）。
func (r *Repo) FindModuleOptions(ctx context.Context, productID uint) ([]SearchOption, error) {
	if r == nil || r.db == nil || productID == 0 {
		return []SearchOption{{Value: "", Label: ""}}, nil
	}
	var rows []ztmodel.ZtModule
	err := r.db.WithContext(ctx).
		Where("root = ? AND type = ? AND deleted = ?", productID, "story", "0").
		Order("grade ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]ztmodel.ZtModule, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	out := []SearchOption{{Value: "", Label: ""}}
	for _, row := range rows {
		label := modulePathLabel(row, byID)
		out = append(out, SearchOption{Value: strconv.FormatUint(uint64(row.ID), 10), Label: label})
	}
	return out, nil
}

func modulePathLabel(m ztmodel.ZtModule, byID map[uint]ztmodel.ZtModule) string {
	parts := []string{m.Name}
	parent := m.Parent
	seen := map[uint]struct{}{m.ID: {}}
	for parent > 0 {
		if _, ok := seen[parent]; ok {
			break
		}
		seen[parent] = struct{}{}
		p, ok := byID[parent]
		if !ok {
			break
		}
		parts = append([]string{p.Name}, parts...)
		parent = p.Parent
	}
	return strings.Join(parts, "/")
}

// FindModuleChildIDs 对齐 tree::getAllChildId：含自身及 path 子孙。
func (r *Repo) FindModuleChildIDs(ctx context.Context, moduleID uint) ([]uint, error) {
	if r == nil || r.db == nil || moduleID == 0 {
		return nil, nil
	}
	var self ztmodel.ZtModule
	err := r.db.WithContext(ctx).Where("id = ? AND deleted = ?", moduleID, "0").Limit(1).Take(&self).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return []uint{moduleID}, nil
		}
		return nil, err
	}
	ids := []uint{self.ID}
	path := strings.TrimSpace(self.Path)
	if path == "" {
		return ids, nil
	}
	var children []uint
	like := path + "%"
	err = r.db.WithContext(ctx).Model(&ztmodel.ZtModule{}).
		Where("deleted = ? AND path LIKE ? AND id <> ?", "0", like, self.ID).
		Pluck("id", &children).Error
	if err != nil {
		return nil, err
	}
	return append(ids, children...), nil
}

// FindPlanOptions 产品下未关闭计划。
func (r *Repo) FindPlanOptions(ctx context.Context, productID uint) ([]SearchOption, error) {
	if r == nil || r.db == nil || productID == 0 {
		return []SearchOption{{Value: "", Label: ""}}, nil
	}
	var rows []ztmodel.ZtProductplan
	err := r.db.WithContext(ctx).
		Select("id", "title").
		Where("product = ? AND deleted = ? AND status <> ?", productID, "0", "closed").
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := []SearchOption{{Value: "", Label: ""}}
	for _, row := range rows {
		out = append(out, SearchOption{
			Value: strconv.FormatUint(uint64(row.ID), 10),
			Label: row.Title,
		})
	}
	return out, nil
}

// FindStoriesBySearch 对齐禅道 story::getBySearch（build linkStory）。
func (r *Repo) FindStoriesBySearch(ctx context.Context, req RepoFindStoriesBySearchReq) ([]linkableStoryRow, int64, error) {
	if r == nil || r.db == nil || req.ProductID == 0 {
		return nil, 0, nil
	}
	base := func() *gorm.DB {
		q := r.db.WithContext(ctx).Table("zt_story AS s").
			Where("s.deleted = ?", "0").
			Where("s.type = ?", "story").
			Where("s.product = ?", req.ProductID).
			Where("s.parent != ?", -1).
			Where("s.status != ?", "draft")
		if len(req.ExcludeIDs) > 0 {
			q = q.Where("s.id NOT IN ?", req.ExcludeIDs)
		}
		branchIDs := branchFilterIDs(req.BranchCSV)
		if len(branchIDs) > 0 {
			q = q.Where("s.branch IN ?", branchIDs)
		}
		if strings.TrimSpace(req.UserWhere) != "" {
			q = q.Where(req.UserWhere, req.UserArgs...)
		}
		return q
	}

	var total int64
	if err := base().Distinct("s.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []linkableStoryRow
	err := base().Select("DISTINCT s.id, s.pri, s.title, s.openedBy, s.assignedTo, s.estimate, s.status, s.stage").
		Order("s.id DESC").
		Limit(req.Limit).
		Offset(req.Offset).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// BuildUserSearchWhere 组装两组用户条件；需要时解析 module 子 ID。
func (r *Repo) BuildUserSearchWhere(ctx context.Context, req LinkStoryListReq, defs []SearchFieldDef) (string, []any, error) {
	resolveModuleIDs := func(cond SearchCond) ([]uint, error) {
		if cond.Field != "module" {
			return nil, nil
		}
		op := NormalizeOperator(cond.Operator)
		if op != "belong" && op != "include" && op != "notinclude" {
			return nil, nil
		}
		id, err := strconv.ParseUint(strings.TrimSpace(cond.Value), 10, 64)
		if err != nil || id == 0 {
			return nil, nil
		}
		return r.FindModuleChildIDs(ctx, uint(id))
	}

	cond1 := req.Cond1()
	cond2 := req.Cond2()
	ctrl1 := FieldControl(cond1.Field, defs)
	ctrl2 := FieldControl(cond2.Field, defs)

	ids1, err := resolveModuleIDs(cond1)
	if err != nil {
		return "", nil, err
	}
	ids2, err := resolveModuleIDs(cond2)
	if err != nil {
		return "", nil, err
	}

	sql1, args1, _ := BuildCondClause(cond1, ctrl1, ids1...)
	sql2, args2, _ := BuildCondClause(cond2, ctrl2, ids2...)
	sql, args := CombineCondSQL(sql1, args1, req.AndOr, sql2, args2)
	if sql == "" {
		return "", nil, nil
	}
	return sql, args, nil
}

// ReplaceMeToken 将 $@me 替换为当前账号。
func ReplaceMeToken(value, account string) string {
	if strings.Contains(value, "$@me") {
		return strings.ReplaceAll(value, "$@me", account)
	}
	return value
}

// QuerySuffixFromReq 生成分页附加查询串（以 & 开头，不含 page/pageSize）。
func QuerySuffixFromReq(req LinkStoryListReq) string {
	if !req.IsBySearch() {
		return ""
	}
	q := url.Values{}
	q.Set("browseType", "bySearch")
	q.Set("field1", req.Field1)
	q.Set("operator1", req.Operator1)
	q.Set("value1", req.Value1)
	q.Set("andOr", req.AndOr)
	q.Set("field2", req.Field2)
	q.Set("operator2", req.Operator2)
	q.Set("value2", req.Value2)
	return "&" + q.Encode()
}
