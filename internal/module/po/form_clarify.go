// =============================================================================
// 文件: internal/module/po/form_clarify.go
// 模块: PO 工作台
// 类型: form
// 职责: 需求澄清表单读写结构体与校验规则。
// =============================================================================

package po

import (
	"strconv"
	"strings"
)

// ClarifyProductItem 涉及产品/系统行。
type ClarifyProductItem struct {
	ID                   uint   `json:"id"`
	ProductID            string `json:"productId"`
	ProductName          string `json:"productName"`
	PM                   string `json:"pm"`
	PMName               string `json:"pmName"`
	DemandCompletionDate string `json:"demandCompletionDate"`
	SystemClarifyDesc    string `json:"systemClarifyDesc"`
	IsAdditionalInfo     string `json:"isAdditionalInfo"`
	AdditionalInfo       string `json:"additionalInfo"`
	IsMainSystem         bool   `json:"isMainSystem"`
	Disabled             bool   `json:"disabled"`
}

// ClarifyUserStoryItem 用户故事条目行。
type ClarifyUserStoryItem struct {
	ID           uint   `json:"id"`
	Role         string `json:"role"`
	GV           string `json:"gv"`
	ProductID    string `json:"productId"`
	Point        int    `json:"point"`
	PointKeyword string `json:"pointKeyword"`
	Revpoint     int    `json:"revpoint"`
	SourceType   string `json:"sourceType"`
	AICode       any    `json:"aiCode"`
	Checked      bool   `json:"checked"`
}

// ClarifyOption 下拉选项。
type ClarifyOption struct {
	Value  string `json:"value"`
	Label  string `json:"label"`
	Dept   string `json:"dept,omitempty"`
	Pinyin string `json:"pinyin,omitempty"`
}

// ClarifyProductOption 候选产品选项。
type ClarifyProductOption struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	PO            string `json:"po"`
	Participating bool   `json:"participating"`
	ClarifyCount  int    `json:"clarifyCount,omitempty"`
}

// ProductMemberOption 产品相关参与人员。
type ProductMemberOption struct {
	Account  string `json:"account"`
	Realname string `json:"realname"`
	Role     string `json:"role"`
}

// DemandClarifyFormResp 需求澄清表单初始化数据。
type DemandClarifyFormResp struct {
	ID                    uint                             `json:"id"`
	Code                  string                           `json:"code"`
	Name                  string                           `json:"name"`
	Category              string                           `json:"category"`
	BRA                   string                           `json:"bra"`
	BRAName               string                           `json:"braName"`
	QD                    string                           `json:"qd"`
	QDName                string                           `json:"qdName"`
	RD                    string                           `json:"rd"`
	RDName                string                           `json:"rdName"`
	Desc                  string                           `json:"desc"`
	ClarifyDesc           string                           `json:"clarifyDesc"`
	ScaleEstimation       int                              `json:"scaleEstimation"`
	IsNewProduct          string                           `json:"isNewProduct"`
	IsRelatedAccounts     string                           `json:"isRelatedAccounts"`
	IsNewFunction         string                           `json:"isNewFunction"`
	IsOtherImportantOrder string                           `json:"isOtherImportantOrder"`
	MultiLegalPersonLogo  string                           `json:"multiLegalPersonLogo"`
	Products              []ClarifyProductItem             `json:"products"`
	UserStories           []ClarifyUserStoryItem           `json:"userStories"`
	CategoryOptions       []ClarifyOption                  `json:"categoryOptions"`
	ProductOptions        []ClarifyProductOption           `json:"productOptions"`
	FrequentProducts      []ClarifyProductOption           `json:"frequentProducts"`
	ProductMembers        map[string][]ProductMemberOption `json:"productMembers"`
	UserOptions           []ClarifyOption                  `json:"userOptions"`
	NoAICategories        []string                         `json:"noAiCategories"`
	AICategories          []string                         `json:"aiCategories"`
	PointToKeyword        map[string]string                `json:"pointToKeyword"`
	RevpointList          map[string][]int                 `json:"revpointList"`
	AllPointList          []int                            `json:"allPointList"`
}

// DemandClarifySubmitReq 需求澄清表单提交请求。
type DemandClarifySubmitReq struct {
	ID                    int64             `json:"id"`
	Category              string            `json:"category"`
	BRA                   string            `json:"bra"`
	QD                    string            `json:"qd"`
	RD                    string            `json:"rd"`
	ClarifyDesc           string            `json:"clarifyDesc"`
	ScaleEstimation       int               `json:"scaleEstimation"`
	IsNewProduct          string            `json:"isNewProduct"`
	IsRelatedAccounts     string            `json:"isRelatedAccounts"`
	IsNewFunction         string            `json:"isNewFunction"`
	IsOtherImportantOrder string            `json:"isOtherImportantOrder"`
	MultiLegalPersonLogo  string            `json:"multiLegalPersonLogo"`
	Comment               string            `json:"comment"`
	Products              []string          `json:"products"`
	PM                    []string          `json:"pm"`
	DemandCompletionDate  []string          `json:"demandCompletionDate"`
	SystemClarifyDesc     []string          `json:"systemClarifyDesc"`
	IsAdditionalInfo      []string          `json:"isAdditionalInfo"`
	AdditionalInfo        []string          `json:"additionalInfo"`
	IsMainSystem          map[string]string `json:"isMainSystem"`
	ClarifyIDList         []string          `json:"clarifyIds"`
	UserStoryNO           []int             `json:"userStoryNo"`
	UserStoryChecked      map[string]string `json:"userStoryChecked"`
	UserStoryID           []string          `json:"userStoryId"`
	Role                  []string          `json:"role"`
	GV                    []string          `json:"gv"`
	EntryProductID        []string          `json:"entryProductID"`
	Point                 []string          `json:"point"`
	Revpoint              []string          `json:"revpoint"`
	SourceType            []string          `json:"sourceType"`
	AICode                []any             `json:"aiCode"`
}

// Validate 校验澄清表单（100% 对齐禅道 changshu.php 规则）。
func (r *DemandClarifySubmitReq) Validate(noAiCategory, aiCategory []string) map[string]string {
	errs := make(map[string]string)
	if strings.TrimSpace(r.Category) == "" {
		errs["category"] = "需求类别不能为空"
	}
	if strings.TrimSpace(r.BRA) == "" {
		errs["bra"] = "需求负责人不能为空"
	}
	if len(r.IsMainSystem) == 0 {
		errs["isMainSystem"] = "主系统不能为空"
	} else {
		// 校验主系统所在行产品是否有效
		var mainIdx string
		for k, v := range r.IsMainSystem {
			if v == "1" || v == "true" {
				mainIdx = k
				break
			}
		}
		if mainIdx == "" {
			errs["isMainSystem"] = "主系统不能为空"
		} else {
			idx, err := strconv.Atoi(mainIdx)
			if err != nil || idx < 0 || idx >= len(r.Products) || strings.TrimSpace(r.Products[idx]) == "" {
				errs["isMainSystem"] = "主系统对应涉及产品不能为空"
			}
		}
	}

	if len(r.Products) == 0 {
		errs["products"] = "涉及产品不能为空"
	} else {
		prodSet := make(map[string]bool)
		for i, p := range r.Products {
			p = strings.TrimSpace(p)
			if p == "" || p == "0" {
				errs["products"] = "涉及产品不能为空"
				break
			}
			if prodSet[p] {
				errs["products"] = "涉及产品不能重复"
				break
			}
			prodSet[p] = true

			pm := ""
			if i < len(r.PM) {
				pm = strings.TrimSpace(r.PM[i])
			}
			if pm == "" {
				errs["pm"] = "需求分析人员不能为空"
				break
			}

			if i < len(r.IsAdditionalInfo) && r.IsAdditionalInfo[i] == "1" {
				addInfo := ""
				if i < len(r.AdditionalInfo) {
					addInfo = strings.TrimSpace(r.AdditionalInfo[i])
				}
				if addInfo == "" {
					errs["additionalInfo"] = "总领文档链接不能为空"
					break
				}
			}
		}
	}

	isNoAi := containsString(noAiCategory, r.Category)
	isAi := containsString(aiCategory, r.Category)

	if !isNoAi {
		if r.ScaleEstimation <= 0 {
			errs["scaleEstimation"] = "需求规模估算不能为空且大于0"
		}
		checkedCount := 0
		for k, v := range r.UserStoryChecked {
			if v == "1" || v == "true" {
				checkedCount++
				idx, err := strconv.Atoi(k)
				if err == nil && idx >= 0 {
					if idx < len(r.Role) && strings.TrimSpace(r.Role[idx]) == "" {
						errs["userStoryRole"] = "用户故事条目角色不能为空"
					}
					if idx < len(r.GV) && strings.TrimSpace(r.GV[idx]) == "" {
						errs["userStoryGV"] = "用户故事条目目标和价值不能为空"
					}
					if idx < len(r.EntryProductID) && strings.TrimSpace(r.EntryProductID[idx]) == "" {
						errs["userStoryProduct"] = "用户故事条目涉及产品不能为空"
					}
					if idx < len(r.Revpoint) && strings.TrimSpace(r.Revpoint[idx]) == "" {
						errs["userStoryRevpoint"] = "校准故事点不能为空"
					}
				}
			}
		}
		if isAi && checkedCount == 0 {
			errs["userStoryChecked"] = "当前需求类别必须至少采纳一条用户故事条目"
		}
	}

	if r.IsNewProduct == "" || r.IsNewProduct == "-1" ||
		r.IsRelatedAccounts == "" || r.IsRelatedAccounts == "-1" ||
		r.IsNewFunction == "" || r.IsNewFunction == "-1" ||
		r.IsOtherImportantOrder == "" || r.IsOtherImportantOrder == "-1" ||
		r.MultiLegalPersonLogo == "" || r.MultiLegalPersonLogo == "-1" {
		errs["importantOrder"] = "重要工单识别各项不能为空"
	}

	return errs
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, s) {
			return true
		}
	}
	return false
}

// AIUserStoryReq AI 生成故事请求。
type AIUserStoryReq struct {
	DemandID    uint     `json:"demandId"`
	Products    []string `json:"products"`
	Desc        string   `json:"desc"`
	ClarifyDesc string   `json:"clarifyDesc"`
}
