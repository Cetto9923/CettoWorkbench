// =============================================================================
// 文件: internal/module/po/service_clarify.go
// 模块: PO 工作台
// 类型: service
// 职责: 需求澄清表单组装、提交处理与 AI 生成。
// =============================================================================

package po

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

// GetDemandClarifyForm 获取需求澄清初始化表单数据。
func (s *Service) GetDemandClarifyForm(ctx context.Context, actor *model.User, demandID int64) (*DemandClarifyFormResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	account := strings.TrimSpace(actor.Account)

	demand, err := s.repo.FindDemandForClarify(ctx, demandID)
	if err != nil {
		return nil, err
	}
	if demand == nil || demand.Deleted != "0" {
		return nil, errorx.New(errorx.ErrCodeNotFound, "需求不存在")
	}

	products, err := s.repo.FindCandidateProducts(ctx, account)
	if err != nil {
		return nil, err
	}
	prodMap := make(map[string]string, len(products))
	for _, p := range products {
		prodMap[strconv.FormatInt(p.ID, 10)] = p.Name
	}

	users, err := s.repo.FindCandidateUsers(ctx, account)
	if err != nil {
		return nil, err
	}
	userMap := make(map[string]string, len(users))
	for _, u := range users {
		userMap[u.Value] = u.Label
	}

	cfg := s.repo.LoadClarifyConfig(ctx)

	clarifies, err := s.repo.FindDemandClarifies(ctx, demandID)
	if err != nil {
		return nil, err
	}

	clarifyItems := make([]ClarifyProductItem, 0, len(clarifies))
	for _, c := range clarifies {
		pName := prodMap[c.Product]
		if pName == "" {
			pName = c.Product
		}
		pmName := userMap[c.PM]
		if pmName == "" {
			pmName = c.PM
		}
		isMain := demand.MainSystem != "" && (demand.MainSystem == c.Product)
		clarifyItems = append(clarifyItems, ClarifyProductItem{
			ID:                   c.ID,
			ProductID:            c.Product,
			ProductName:          pName,
			PM:                   c.PM,
			PMName:               pmName,
			DemandCompletionDate: c.DemandCompletionDate,
			SystemClarifyDesc:    c.SystemClarifyDesc,
			IsAdditionalInfo:     c.IsAdditionalInfo,
			AdditionalInfo:       c.AdditionalInfo,
			IsMainSystem:         isMain,
		})
	}

	userStories, err := s.repo.FindDemandUserStories(ctx, demandID)
	if err != nil {
		return nil, err
	}
	storyItems := make([]ClarifyUserStoryItem, 0, len(userStories))
	for _, us := range userStories {
		ptStr := strconv.Itoa(us.Point)
		kw := cfg.PointToKeyword[ptStr]
		rev := us.Revpoint
		if rev == 0 {
			rev = us.Point
		}
		storyItems = append(storyItems, ClarifyUserStoryItem{
			ID:           us.ID,
			Role:         us.Role,
			GV:           us.GV,
			ProductID:    us.Product,
			Point:        us.Point,
			PointKeyword: kw,
			Revpoint:     rev,
			SourceType:   us.SourceType,
			AICode:       us.AICode,
			Checked:      true,
		})
	}

	scaleEst, _ := strconv.Atoi(demand.ScaleEstimation)

	code := fmt.Sprintf("US%d", demand.ID)

	frequentProducts, _ := s.repo.FindFrequentProducts(ctx, account, 8)

	prodIDs := make([]int64, 0, len(products))
	for _, p := range products {
		prodIDs = append(prodIDs, p.ID)
	}
	productMembers, _ := s.repo.FindProductMembers(ctx, prodIDs)

	resp := &DemandClarifyFormResp{
		ID:                    demand.ID,
		Code:                  code,
		Name:                  demand.Name,
		Category:              demand.Category,
		BRA:                   demand.BRA,
		BRAName:               userMap[demand.BRA],
		QD:                    demand.QD,
		QDName:                userMap[demand.QD],
		RD:                    demand.RD,
		RDName:                userMap[demand.RD],
		Desc:                  demand.Desc,
		ClarifyDesc:           demand.ClarifyDesc,
		ScaleEstimation:       scaleEst,
		IsNewProduct:          demand.IsNewProduct,
		IsRelatedAccounts:     demand.IsRelatedAccounts,
		IsNewFunction:         demand.IsNewFunction,
		IsOtherImportantOrder: demand.IsOtherImportantOrder,
		MultiLegalPersonLogo:  demand.MultiLegalPersonLogo,
		Products:              clarifyItems,
		UserStories:           storyItems,
		CategoryOptions:       cfg.CategoryOptions,
		ProductOptions:        products,
		FrequentProducts:      frequentProducts,
		ProductMembers:        productMembers,
		UserOptions:           users,
		NoAICategories:        cfg.NoAICategories,
		AICategories:          cfg.AICategories,
		PointToKeyword:        cfg.PointToKeyword,
		RevpointList:          cfg.RevpointList,
		AllPointList:          cfg.AllPointList,
	}

	return resp, nil
}

// ClarifyDemand 提交需求澄清。
func (s *Service) ClarifyDemand(ctx context.Context, actor *model.User, req DemandClarifySubmitReq) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	account := strings.TrimSpace(actor.Account)

	demand, err := s.repo.FindDemandForClarify(ctx, req.ID)
	if err != nil {
		return err
	}
	if demand == nil || demand.Deleted != "0" {
		return errorx.New(errorx.ErrCodeNotFound, "需求不存在")
	}
	if strings.TrimSpace(demand.Status) != "active" {
		return errorx.New(errorx.ErrCodeConflict, "当前需求不是待澄清状态")
	}

	cfg := s.repo.LoadClarifyConfig(ctx)
	if fieldErrs := req.Validate(cfg.NoAICategories, cfg.AICategories); len(fieldErrs) > 0 {
		var msgs []string
		for _, v := range fieldErrs {
			msgs = append(msgs, v)
		}
		return errorx.New(errorx.ErrCodeInvalidParam, strings.Join(msgs, "; "))
	}

	isAddNested := make([][]string, len(req.IsAdditionalInfo))
	for i, v := range req.IsAdditionalInfo {
		isAddNested[i] = []string{v}
	}

	ztClient := zentao.DefaultClient()
	ztErr := ztClient.ClarifyDemand(ctx, zentao.DemandClarifyParams{
		DemandID:              uint(req.ID),
		Account:               account,
		Category:              req.Category,
		BRA:                   req.BRA,
		QD:                    req.QD,
		RD:                    req.RD,
		ClarifyDesc:           req.ClarifyDesc,
		ScaleEstimation:       req.ScaleEstimation,
		IsNewProduct:          req.IsNewProduct,
		IsRelatedAccounts:     req.IsRelatedAccounts,
		IsNewFunction:         req.IsNewFunction,
		IsOtherImportantOrder: req.IsOtherImportantOrder,
		MultiLegalPersonLogo:  req.MultiLegalPersonLogo,
		Status:                "clarified",
		Comment:               req.Comment,
		Products:              req.Products,
		PM:                    req.PM,
		DemandCompletionDate:  req.DemandCompletionDate,
		SystemClarifyDesc:     req.SystemClarifyDesc,
		IsAdditionalInfo:      isAddNested,
		AdditionalInfo:        req.AdditionalInfo,
		IsMainSystem:          req.IsMainSystem,
		ClarifyIDList:         req.ClarifyIDList,
		UserStoryNO:           req.UserStoryNO,
		UserStoryChecked:      req.UserStoryChecked,
		UserStoryID:           req.UserStoryID,
		Role:                  req.Role,
		GV:                    req.GV,
		EntryProductID:        req.EntryProductID,
		Point:                 req.Point,
		Revpoint:              req.Revpoint,
		SourceType:            req.SourceType,
		AICode:                req.AICode,
	})
	if ztErr != nil {
		return errorx.New(errorx.ErrCodeInternal, "禅道澄清保存失败: "+ztErr.Error())
	}

	return nil
}

// GenerateAIUserStory 代理调用禅道 AI 生成用户故事。
func (s *Service) GenerateAIUserStory(ctx context.Context, actor *model.User, req AIUserStoryReq) (*zentao.AIUserStoryResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}

	prodOptions, _ := s.repo.FindCandidateProducts(ctx, strings.TrimSpace(actor.Account))
	prodNameMap := make(map[string]string)
	for _, p := range prodOptions {
		prodNameMap[strconv.FormatInt(p.ID, 10)] = p.Name
	}

	type prodData struct {
		Name string `json:"name"`
		ID   int64  `json:"id"`
	}
	var pList []prodData
	for _, pidStr := range req.Products {
		pid, err := strconv.ParseInt(pidStr, 10, 64)
		if err == nil && pid > 0 {
			pList = append(pList, prodData{
				Name: prodNameMap[pidStr],
				ID:   pid,
			})
		}
	}

	inputMap := map[string]any{
		"product": pList,
		"desc":    req.Desc,
	}
	if strings.TrimSpace(req.ClarifyDesc) != "" {
		inputMap["clarifyDesc"] = strings.TrimSpace(req.ClarifyDesc)
	}
	bytes, _ := json.Marshal(inputMap)

	ztClient := zentao.DefaultClient()
	return ztClient.GenerateAIUserStory(ctx, zentao.GenerateAIUserStoryParams{
		DemandID:     req.DemandID,
		Account:      actor.Account,
		InputContent: string(bytes),
		Source:       "clarify",
	})
}
