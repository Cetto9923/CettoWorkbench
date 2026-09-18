// =============================================================================
// 文件: internal/module/po/servicedetail_relations.go
// 模块: PO 工作台
// 类型: action
// 职责: 组装业需详情关联块（转化研发需求/用户故事/评审信息/工单）。
// 依赖: internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"

	"workbench/internal/pkg/zentao"
)

var htmlTagPattern = regexp.MustCompile(`(?s)<[^>]*>`)

func (s *Service) loadDemandDetailRelations(
	ctx context.Context,
	demandID int64,
	displayMap map[string]string,
) (stories []DemandStoryItem, userStories []DemandUserStoryItem, reviews []DemandReviewRecordItem, tickets []DemandTicketItem, err error) {
	storyRows, err := s.repo.FindDemandStoriesByDemandID(ctx, demandID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	usRows, err := s.repo.FindDemandUserStoriesByDemandID(ctx, demandID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	reviewRows, err := s.repo.FindDemandReviewRecordsByDemandID(ctx, demandID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	ticketRows, err := s.repo.FindDemandTicketsByDemandID(ctx, demandID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return buildDemandStoryItems(storyRows),
		buildDemandUserStoryItems(usRows),
		buildDemandReviewRecordItems(reviewRows, displayMap),
		buildDemandTicketItems(ticketRows),
		nil
}

func buildDemandStoryItems(rows []demandStoryDetailRow) []DemandStoryItem {
	out := make([]DemandStoryItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, DemandStoryItem{
			ID:          row.ID,
			Title:       row.Title,
			Stage:       strings.TrimSpace(row.Stage),
			StageLabel:  storyStageLabel(row.Stage),
			ProductName: dashOr(row.ProductName),
			ReleaseDate: formatDateYMD(row.ReleaseDate),
			ZentaoURL:   zentao.StoryViewURL(row.ID),
		})
	}
	return out
}

func buildDemandUserStoryItems(rows []demandUserStoryDetailRow) []DemandUserStoryItem {
	out := make([]DemandUserStoryItem, 0, len(rows))
	for i, row := range rows {
		point := row.Point
		if row.Revpoint > 0 {
			point = row.Revpoint
		}
		out = append(out, DemandUserStoryItem{
			NO:          i + 1,
			Role:        dashOr(row.Role),
			GV:          dashOr(row.GV),
			ProductName: dashOr(row.ProductName),
			PointLabel:  storyPointKeywordLabel(point),
		})
	}
	return out
}

func buildDemandReviewRecordItems(rows []demandReviewRecordRow, displayMap map[string]string) []DemandReviewRecordItem {
	out := make([]DemandReviewRecordItem, 0, len(rows))
	for _, row := range rows {
		account := strings.TrimSpace(row.CreatedBy)
		out = append(out, DemandReviewRecordItem{
			ReviewType:        strings.TrimSpace(row.ReviewType),
			ReviewTypeLabel:   demandReviewTypeLabel(row.ReviewType),
			ReviewDate:        formatDateYMD(row.ReviewDate),
			ReviewResult:      plainReviewResult(row.ReviewResult),
			CreatedBy:         account,
			CreatedByName:     dashOr(lookupAccountDisplay(displayMap, account)),
			CreatedDate:       formatDateYMD(row.CreatedDate),
			ReviewStatus:      strings.TrimSpace(row.ReviewStatus),
			ReviewStatusLabel: demandReviewStatusLabel(row.ReviewStatus),
		})
	}
	return out
}

func buildDemandTicketItems(rows []demandTicketRow) []DemandTicketItem {
	out := make([]DemandTicketItem, 0, len(rows))
	for _, row := range rows {
		pri := ""
		if row.Pri > 0 {
			pri = fmt.Sprintf("%d", row.Pri)
		}
		out = append(out, DemandTicketItem{
			ID:          row.ID,
			Title:       row.Title,
			Pri:         dashOr(pri),
			Status:      strings.TrimSpace(row.Status),
			StatusLabel: ticketStatusLabel(row.Status),
			ZentaoURL:   zentao.TicketViewURL(row.ID),
		})
	}
	return out
}

func plainReviewResult(raw string) string {
	text := html.UnescapeString(strings.TrimSpace(raw))
	text = htmlTagPattern.ReplaceAllString(text, "")
	text = strings.Join(strings.Fields(text), " ")
	return text
}
