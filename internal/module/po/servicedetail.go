// =============================================================================
// 文件: internal/module/po/servicedetail.go
// 模块: PO 工作台
// 类型: action
// 职责: 组装业需评审抽屉详情（对齐禅道 demand-view 字段取值）。
// 依赖: internal/model
//       internal/model/zentao
//       internal/pkg/errorx
//       internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
	ztmodel "workbench/internal/model/zentao"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

// GetDemandDetail 查询业需详情；不可见返回 403，不存在返回 404。
func (s *Service) GetDemandDetail(ctx context.Context, actor *model.User, req DemandDetailReq) (*DemandDetailResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	id := req.ExtractDemandID()
	if id <= 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "需求 ID 无效")
	}

	row, err := s.repo.FindDemandDetailByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.New(errorx.ErrCodeNotFound, "需求不存在")
		}
		return nil, err
	}

	displayMap, mapErr := s.loadAccountDisplayMap(ctx, actor)
	if mapErr != nil {
		return nil, mapErr
	}
	files, fileErr := s.repo.FindDemandFiles(ctx, id)
	if fileErr != nil {
		return nil, fileErr
	}
	resp := buildDemandDetailResp(row, displayMap, files)
	return &resp, nil
}

func buildDemandDetailResp(row *DemandDetailRow, displayMap map[string]string, files []ztmodel.ZtFile) DemandDetailResp {
	if row == nil {
		return DemandDetailResp{}
	}
	atts := make([]DemandAttachment, 0, len(files))
	for _, f := range files {
		atts = append(atts, DemandAttachment{
			ID:       f.ID,
			Title:    f.Title,
			Size:     formatFileSize(f.Size),
			Download: zentao.URL("file", "download", fmt.Sprintf("fileID=%d", f.ID)),
		})
	}
	return DemandDetailResp{
		ID:                fmt.Sprintf("US%d", row.ID),
		DemandID:          row.ID,
		Title:             row.Name,
		Pri:               formatDemandPri(row.Pri),
		Category:          demandCategoryLabel(row.Category),
		Source:            demandSourceLabel(row.Source),
		PoolName:          dashOr(row.PoolName),
		Deadline:          formatDateYMD(row.Deadline),
		ProposerName:      dashOr(lookupAccountDisplay(displayMap, row.Originator)),
		ProposerDept:      dashOr(row.ProposeDept),
		OwnerName:         dashOr(lookupAccountDisplay(displayMap, row.BRA)),
		Reviewer:          dashOr(lookupAccountsDisplay(displayMap, row.Reviewer)),
		CreatedName:       dashOr(lookupAccountDisplay(displayMap, row.CreatedBy)),
		CurrentOwner:      dashOr(lookupAccountDisplay(displayMap, row.AssignedTo)), // UI：指派给
		ZentaoStatus:      strings.TrimSpace(row.Status),
		ZentaoStatusLabel: demandStatusLabel(row.Status),
		ValueStageLabel:   demandValueStageLabel(row.Status),
		SpecHtml:          row.Desc,
		VerifyHtml:        row.VerifyPlan,
		ZentaoURL:         zentao.DemandViewURL(uint(row.ID)),
		ZentaoEditURL:     zentao.URL("demand", "edit", fmt.Sprintf("demandID=%d", row.ID)),
		Attachments:       atts,
	}
}

func formatDateYMD(t *time.Time) string {
	if t == nil || t.IsZero() || t.Year() < 1971 {
		return "—"
	}
	return t.Format("2006-01-02")
}

func formatFileSize(size int) string {
	if size <= 0 {
		return "0 B"
	}
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	return fmt.Sprintf("%.1f KB", float64(size)/1024.0)
}
