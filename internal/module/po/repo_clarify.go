// =============================================================================
// 文件: internal/module/po/repo_clarify.go
// 模块: PO 工作台
// 类型: repo
// 职责: 需求澄清表单数据读取（demand / demandclarify / demanduserstory / product / user）。
// =============================================================================

package po

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// demandClarifyRawRow 需求主表字段。
type demandClarifyRawRow struct {
	ID                    uint   `gorm:"column:id"`
	Name                  string `gorm:"column:name"`
	Status                string `gorm:"column:status"`
	Category              string `gorm:"column:category"`
	BRA                   string `gorm:"column:BRA"`
	QD                    string `gorm:"column:QD"`
	RD                    string `gorm:"column:RD"`
	Desc                  string `gorm:"column:desc"`
	ClarifyDesc           string `gorm:"column:clarifyDesc"`
	MainSystem            string `gorm:"column:mainSystem"`
	ScaleEstimation       string `gorm:"column:scaleEstimation"`
	IsNewProduct          string `gorm:"column:isNewProduct"`
	IsRelatedAccounts     string `gorm:"column:isRelatedAccounts"`
	IsNewFunction         string `gorm:"column:isNewFunction"`
	IsOtherImportantOrder string `gorm:"column:isOtherImportantOrder"`
	MultiLegalPersonLogo  string `gorm:"column:multiLegalPersonLogo"`
	Deleted               string `gorm:"column:deleted"`
}

// demandClarifyDBItem 需求涉及系统。
type demandClarifyDBItem struct {
	ID                   uint   `gorm:"column:id"`
	Demand               uint   `gorm:"column:demand"`
	Product              string `gorm:"column:product"`
	PM                   string `gorm:"column:PM"`
	DemandCompletionDate string `gorm:"column:demandCompletionDate"`
	SystemClarifyDesc    string `gorm:"column:systemClarifyDesc"`
	IsAdditionalInfo     string `gorm:"column:isAdditionalInfo"`
	AdditionalInfo       string `gorm:"column:additionalInfo"`
}

// demandUserStoryDBItem 需求用户故事。
type demandUserStoryDBItem struct {
	ID         uint   `gorm:"column:id"`
	Demand     uint   `gorm:"column:demand"`
	Role       string `gorm:"column:role"`
	GV         string `gorm:"column:gv"`
	Product    string `gorm:"column:product"`
	Point      int    `gorm:"column:point"`
	Revpoint   int    `gorm:"column:revpoint"`
	AICode     any    `gorm:"column:aiCode"`
	SourceType string `gorm:"column:source_type"`
}

// FindDemandForClarify 从主库读取需求主表。
func (r *Repo) FindDemandForClarify(ctx context.Context, id int64) (*demandClarifyRawRow, error) {
	db, err := r.writer()
	if err != nil {
		return nil, err
	}
	var row demandClarifyRawRow
	err = db.WithContext(ctx).Table("zt_demand").
		Select("id, name, status, category, BRA, QD, RD, `desc`, clarifyDesc, mainSystem, scaleEstimation, isNewProduct, isRelatedAccounts, isNewFunction, isOtherImportantOrder, multiLegalPersonLogo, deleted").
		Where("id = ?", id).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load demand clarify %d: %w", id, err)
	}
	return &row, nil
}

// FindDemandClarifies 读取涉及系统表。
func (r *Repo) FindDemandClarifies(ctx context.Context, demandID int64) ([]demandClarifyDBItem, error) {
	db, err := r.writer()
	if err != nil {
		return nil, err
	}
	var items []demandClarifyDBItem
	err = db.WithContext(ctx).Table("zt_demandclarify").
		Where("demand = ?", demandID).
		Order("id ASC").
		Find(&items).Error
	return items, err
}

// FindDemandUserStories 读取需求用户故事条目。
func (r *Repo) FindDemandUserStories(ctx context.Context, demandID int64) ([]demandUserStoryDBItem, error) {
	db, err := r.writer()
	if err != nil {
		return nil, err
	}
	var items []demandUserStoryDBItem
	err = db.WithContext(ctx).Table("zt_demanduserstory").
		Where("demand = ?", demandID).
		Order("id ASC").
		Find(&items).Error
	return items, err
}

// FindCandidateProducts 读取全部可选产品（过滤禅道项目生成的影子产品 shadow = 0）。
// 当前用户作为产品、需求、发布、反馈负责人或产品白名单成员时，该产品排在前面。
func (r *Repo) FindCandidateProducts(ctx context.Context, account string) ([]ClarifyProductOption, error) {
	db, err := r.reader()
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID            int64  `gorm:"column:id"`
		Name          string `gorm:"column:name"`
		PO            string `gorm:"column:PO"`
		Participating bool   `gorm:"column:participating"`
	}
	err = db.WithContext(ctx).Table("zt_product").
		Where("deleted = '0' AND status != 'closed' AND shadow = '0'").
		Order("participating DESC, `order` DESC, id DESC").
		Select(`id, name, PO,
			(PO = ? OR QD = ? OR RD = ? OR feedback = ?
				OR CONCAT(',', whitelist, ',') LIKE CONCAT('%,', ?, ',%')) AS participating`,
			account, account, account, account, account).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]ClarifyProductOption, 0, len(rows))
	for _, row := range rows {
		out = append(out, ClarifyProductOption{
			ID:            row.ID,
			Name:          row.Name,
			PO:            row.PO,
			Participating: row.Participating,
		})
	}
	return out, nil
}

// FindFrequentProducts 读取当前登录用户近期最常澄清的产品（用于快捷点击胶囊）。
func (r *Repo) FindFrequentProducts(ctx context.Context, account string, limit int) ([]ClarifyProductOption, error) {
	if limit <= 0 {
		limit = 8
	}
	db, err := r.reader()
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID           int64  `gorm:"column:id"`
		Name         string `gorm:"column:name"`
		PO           string `gorm:"column:PO"`
		ClarifyCount int    `gorm:"column:clarify_count"`
	}
	const q = `SELECT p.id, p.name, p.PO, COUNT(dc.id) AS clarify_count
FROM zt_product p
JOIN zt_demandclarify dc ON dc.product = CAST(p.id AS CHAR)
JOIN zt_demand d ON d.id = dc.demand AND d.deleted = '0'
WHERE p.deleted = '0' AND p.status != 'closed' AND p.shadow = '0'
  AND (dc.PM = ? OR d.BRA = ? OR p.PO = ?)
GROUP BY p.id, p.name, p.PO
ORDER BY clarify_count DESC, p.id DESC
LIMIT ?`
	if err := db.WithContext(ctx).Raw(q, account, account, account, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ClarifyProductOption, 0, len(rows))
	for _, row := range rows {
		out = append(out, ClarifyProductOption{
			ID:            row.ID,
			Name:          row.Name,
			PO:            row.PO,
			Participating: true,
			ClarifyCount:  row.ClarifyCount,
		})
	}
	return out, nil
}

// FindProductMembers 查询指定产品相关参与人员（PO、QD、RD、白名单用户、历史曾任PM等）。
func (r *Repo) FindProductMembers(ctx context.Context, productIDs []int64) (map[string][]ProductMemberOption, error) {
	result := make(map[string][]ProductMemberOption)
	if len(productIDs) == 0 {
		return result, nil
	}
	db, err := r.reader()
	if err != nil {
		return result, nil
	}

	var prodRows []struct {
		ID        int64  `gorm:"column:id"`
		PO        string `gorm:"column:PO"`
		QD        string `gorm:"column:QD"`
		RD        string `gorm:"column:RD"`
		Feedback  string `gorm:"column:feedback"`
		Ticket    string `gorm:"column:ticket"`
		CreatedBy string `gorm:"column:createdBy"`
		Whitelist string `gorm:"column:whitelist"`
	}
	if err := db.WithContext(ctx).Table("zt_product").
		Where("id IN ? AND deleted = '0'", productIDs).
		Select("id, PO, QD, RD, feedback, ticket, createdBy, whitelist").
		Find(&prodRows).Error; err != nil {
		return nil, err
	}

	var pmRows []struct {
		ProductID string `gorm:"column:product"`
		PM        string `gorm:"column:PM"`
	}
	idStrs := make([]string, len(productIDs))
	for i, id := range productIDs {
		idStrs[i] = fmt.Sprintf("%d", id)
	}
	_ = db.WithContext(ctx).Table("zt_demandclarify").
		Where("product IN ? AND PM != ''", idStrs).
		Select("DISTINCT product, PM").
		Find(&pmRows).Error

	// 收集全部待解析账号
	accountSet := make(map[string]bool)
	for _, p := range prodRows {
		for _, acc := range []string{p.PO, p.QD, p.RD, p.Feedback, p.Ticket, p.CreatedBy} {
			if a := strings.TrimSpace(acc); a != "" {
				accountSet[a] = true
			}
		}
		for _, w := range strings.Split(p.Whitelist, ",") {
			if a := strings.TrimSpace(w); a != "" {
				accountSet[a] = true
			}
		}
	}
	for _, pm := range pmRows {
		if a := strings.TrimSpace(pm.PM); a != "" {
			accountSet[a] = true
		}
	}

	// 批量查询真实姓名
	nameMap := make(map[string]string)
	if len(accountSet) > 0 {
		accounts := make([]string, 0, len(accountSet))
		for a := range accountSet {
			accounts = append(accounts, a)
		}
		var users []struct {
			Account  string `gorm:"column:account"`
			Realname string `gorm:"column:realname"`
		}
		if err := db.WithContext(ctx).Table("zt_user").
			Where("account IN ?", accounts).
			Select("account, realname").
			Find(&users).Error; err == nil {
			for _, u := range users {
				nameMap[u.Account] = u.Realname
			}
		}
	}

	// 按产品聚合去重人员列表
	for _, p := range prodRows {
		pKey := fmt.Sprintf("%d", p.ID)
		var members []ProductMemberOption
		seen := make(map[string]bool)

		addMember := func(account, role string) {
			a := strings.TrimSpace(account)
			if a == "" || seen[a] {
				return
			}
			seen[a] = true
			rn := nameMap[a]
			if rn == "" {
				rn = a
			}
			members = append(members, ProductMemberOption{
				Account:  a,
				Realname: rn,
				Role:     role,
			})
		}

		addMember(p.PO, "产品负责人 (PO)")
		addMember(p.QD, "测试负责人 (QD)")
		addMember(p.RD, "研发负责人 (RD)")
		for _, w := range strings.Split(p.Whitelist, ",") {
			addMember(w, "白名单成员")
		}
		for _, pm := range pmRows {
			if pm.ProductID == pKey {
				addMember(pm.PM, "曾任需求分析人 (PM)")
			}
		}
		addMember(p.Feedback, "反馈负责人")
		addMember(p.Ticket, "工单负责人")
		addMember(p.CreatedBy, "创建人")

		if len(members) > 0 {
			result[pKey] = members
		}
	}

	return result, nil
}

// ClarifyConfigData 配置数据。
type ClarifyConfigData struct {
	CategoryOptions []ClarifyOption
	NoAICategories  []string
	AICategories    []string
	PointToKeyword  map[string]string
	RevpointList    map[string][]int
	AllPointList    []int
}

// LoadClarifyConfig 读取类别、AI 类别与故事点配置。
func (r *Repo) LoadClarifyConfig(ctx context.Context) ClarifyConfigData {
	cfg := ClarifyConfigData{
		CategoryOptions: []ClarifyOption{
			{Value: "feature", Label: "新增需求"},
			{Value: "experience", Label: "体验优化"},
			{Value: "BUG", Label: "BUG"},
			{Value: "tecopt", Label: "技术优化（科技内部提单使用）"},
			{Value: "performance", Label: "性能（技术测试团队提单使用）"},
			{Value: "safe", Label: "安全（安全&合规团队提单使用）"},
			{Value: "DATACG", Label: "数据变更（研发迭代：由研发团队执行）"},
			{Value: "datachange", Label: "数据变更（快速变更：由运维团队执行）"},
			{Value: "dataexport", Label: "数据导出"},
			{Value: "other", Label: "其他（配合、测试、排查）"},
		},
		NoAICategories: []string{"datachange", "dataexport", "safe"},
		AICategories:   []string{"feature", "feature_1", "feature_2", "feature_3", "experience", "tecopt"},
		PointToKeyword: map[string]string{"2": "微型", "3": "小型", "5": "中型", "8": "大型"},
		RevpointList:   map[string][]int{"2": {2, 3}, "3": {2, 3, 5}, "5": {3, 5, 8}, "8": {5, 8}},
		AllPointList:   []int{2, 3, 5, 8},
	}

	db, err := r.reader()
	if err != nil {
		return cfg
	}

	// 尝试从 zt_config 读取系统后台配置
	var confRows []struct {
		Key   string `gorm:"column:key"`
		Value string `gorm:"column:value"`
	}
	if err := db.WithContext(ctx).Table("zt_config").
		Where("module = 'custom' AND section = 'clarifyCategoryAIConfig'").
		Find(&confRows).Error; err == nil {
		for _, cr := range confRows {
			var list []string
			if json.Unmarshal([]byte(cr.Value), &list) == nil && len(list) > 0 {
				if cr.Key == "aiCategory" {
					cfg.AICategories = list
				} else if cr.Key == "noAiCategory" {
					cfg.NoAICategories = list
				}
			}
		}
	}

	return cfg
}
