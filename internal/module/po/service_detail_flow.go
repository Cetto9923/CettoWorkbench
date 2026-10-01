// =============================================================================
// 文件: internal/module/po/service_detail_flow.go
// 模块: PO 工作台
// 类型: service
// 职责: 装配需求详情 B3 流程与审批（变更记录、挂起日志、评审记录、主管部门审批）及
//       管理信息（7 项重要检查项、8 项实际时间）。
// 依赖: internal/model
// =============================================================================

package po

import (
	"context"
	"strings"
	"time"

	"workbench/internal/pkg/datefmt"
)

// populateFlowApproval 装配流程与审批数据块。
// 四块均无数据时返回 nil，确保在关闭或无数据场景下 omitempty 完全省略。
func (s *DetailService) populateFlowApproval(ctx context.Context, demandID uint) (*DetailFlowApproval, error) {
	if s == nil || s.repo == nil || demandID == 0 {
		return nil, nil
	}

	mgrReviews, err := s.repo.FindDemandManagerReviews(ctx, demandID)
	if err != nil {
		return nil, err
	}

	changes, err := s.repo.FindDemandChanges(ctx, demandID)
	if err != nil {
		return nil, err
	}

	hangLogs, err := s.repo.FindDemandHangLogs(ctx, demandID)
	if err != nil {
		return nil, err
	}

	reviewRecords, err := s.repo.FindDemandReviewRecords(ctx, demandID)
	if err != nil {
		return nil, err
	}

	if len(mgrReviews) == 0 && len(changes) == 0 && len(hangLogs) == 0 && len(reviewRecords) == 0 {
		return nil, nil
	}

	return &DetailFlowApproval{
		ManagerReviews: mgrReviews,
		DemandChanges:  changes,
		HangLogs:       hangLogs,
		ReviewRecords:  reviewRecords,
	}, nil
}

// populateManagementInfo 装配管理信息数据块（重要检查项与实际时间）。
// 无数据时返回 nil，确保逐字兼容旧接口输出。
func (s *DetailService) populateManagementInfo(ctx context.Context, demandID uint) (*DetailManagementInfo, error) {
	if s == nil || s.repo == nil || demandID == 0 {
		return nil, nil
	}

	row, err := s.repo.FindDemandManagementRow(ctx, demandID)
	if err != nil || row == nil {
		return nil, err
	}

	checks := buildImportantChecks(row)
	times, err := s.buildActualTimes(ctx, demandID, row)
	if err != nil {
		return nil, err
	}

	if checks == nil && times == nil {
		return nil, nil
	}

	return &DetailManagementInfo{
		ImportantChecks: checks,
		ActualTimes:     times,
	}, nil
}

func buildImportantChecks(row *DemandManagementRow) *DemandImportantChecks {
	legalLogo, legalLogoLabel, has1 := parseCheckField(row.MultiLegalPersonLogo, "-1", mapMultiLegalPersonLogoLabel)
	relatedAcc, relatedAccLabel, has2 := parseCheckField(row.IsRelatedAccounts, "-1", mapYesNoLabel)
	importantOrder, importantOrderLabel, has3 := parseCheckField(row.IsImportantOrder, "0", mapYesNoLabel)
	needReview, needReviewLabel, has4 := parseCheckField(row.IsNeedReview, "0", mapYesNoLabel)
	onetime, onetimeLabel, has5 := parseCheckField(row.OnetimeAcceptance, "", mapOnetimeLabel)
	carReview, carReviewLabel, has6 := parseCheckField(row.IsCarReview, "0", mapYesNoLabel)
	verifyDate, verifyDateLabel, has7 := parseCheckField(row.VerifyDate, "0", mapVerifyDateLabel)

	if !has1 && !has2 && !has3 && !has4 && !has5 && !has6 && !has7 {
		return nil
	}

	return &DemandImportantChecks{
		MultiLegalPersonLogo:      legalLogo,
		MultiLegalPersonLogoLabel: legalLogoLabel,
		IsRelatedAccounts:         relatedAcc,
		IsRelatedAccountsLabel:    relatedAccLabel,
		IsImportantOrder:          importantOrder,
		IsImportantOrderLabel:     importantOrderLabel,
		IsNeedReview:              needReview,
		IsNeedReviewLabel:         needReviewLabel,
		OnetimeAcceptance:         onetime,
		OnetimeAcceptanceLabel:    onetimeLabel,
		IsCarReview:               carReview,
		IsCarReviewLabel:          carReviewLabel,
		VerifyDate:                verifyDate,
		VerifyDateLabel:           verifyDateLabel,
	}
}

func parseCheckField(val, ignoreVal string, mapper func(string) string) (string, string, bool) {
	cleaned := cleanCheckValue(val)
	if cleaned == "" || (ignoreVal != "" && cleaned == ignoreVal) {
		return cleaned, "", false
	}
	return cleaned, mapper(cleaned), true
}

func (s *DetailService) buildActualTimes(ctx context.Context, demandID uint, row *DemandManagementRow) (*DemandActualTimes, error) {
	clarify := datefmt.Date(row.ClarifyDate)
	actualDevStart := datefmt.Date(row.ActualDevStartDate)
	actualTestStart := datefmt.Date(row.ActualTestStartDate)
	submitAcceptance := datefmt.Date(row.SubmitAcceptanceDate)
	acceptanced := datefmt.Date(row.AcceptancedDate)
	reviewed := datefmt.Date(row.ReviewedDate)
	deliver := datefmt.Date(row.DeliverDate)

	realRelease, err := s.repo.FindRealReleaseDate(ctx, demandID)
	if err != nil {
		realRelease = ""
	}

	// 空判定针对底层日期是否设置，与展示文案解耦：datefmt 对零日期给「未设置」、nil 给「—」。
	allUnset := true
	for _, v := range []string{clarify, actualDevStart, actualTestStart, submitAcceptance, acceptanced, reviewed, deliver} {
		if v != datefmt.Unset && v != datefmt.Empty {
			allUnset = false
			break
		}
	}
	if allUnset && realRelease == "" {
		return nil, nil
	}

	return &DemandActualTimes{
		ClarifyDate:              clarify,
		FirstToStoryDate:         actualDevStart,
		ActualDevStartDate:       actualDevStart,
		ActualTestStartDate:      actualTestStart,
		ActualDevCompletionDate:  actualTestStart,
		SubmitAcceptanceDate:     submitAcceptance,
		ActualTestCompletionDate: submitAcceptance,
		AcceptancedDate:          acceptanced,
		ReviewedDate:             reviewed,
		DeliverDate:              deliver,
		RealReleaseDate:          realRelease,
	}, nil
}

func cleanCheckValue(val string) string {
	return strings.TrimSpace(val)
}

func mapMultiLegalPersonLogoLabel(val string) string {
	switch val {
	case "0":
		return "常熟"
	case "1":
		return "村镇"
	case "2":
		return "常熟&村镇"
	}
	return ""
}

func mapYesNoLabel(val string) string {
	switch val {
	case "1", "yes":
		return "是"
	case "0", "no":
		return "否"
	}
	return val
}

func mapOnetimeLabel(val string) string {
	switch val {
	case "yes", "1":
		return "是"
	case "no", "0":
		return "否"
	}
	return val
}

func mapVerifyDateLabel(val string) string {
	switch val {
	case "1":
		return "当日验证"
	case "2":
		return "次日验证"
	case "3":
		return "首笔验证"
	}
	return val
}

func formatTime(t *time.Time, layout string) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(layout)
}

func mapChangeTypeLabel(t string) string {
	switch t {
	case "changeDemandContent":
		return "需求内容变更"
	case "changePublishPlan":
		return "发布计划变更"
	case "changeDemandContent,changePublishPlan", "changePublishPlan,changeDemandContent":
		return "需求内容变更、发布计划变更"
	}
	return t
}

func mapReviewResultLabel(res string) string {
	switch res {
	case "pass":
		return "通过"
	case "refuse":
		return "拒绝"
	case "wait":
		return "待审批"
	case "withdraw":
		return "撤回"
	}
	return res
}

func mapHangUpTypeLabel(t string) string {
	switch t {
	case "business":
		return "业务原因"
	case "tech":
		return "技术原因"
	case "other":
		return "其他原因"
	}
	return t
}

func mapReviewTypeLabel(t string) string {
	switch t {
	case "demand":
		return "需求评审"
	case "design":
		return "设计评审"
	case "test":
		return "测试案例评审"
	}
	return t
}

func mapReviewStatusLabel(s string) string {
	switch s {
	case "reviewPassed":
		return "评审通过"
	case "noRequired":
		return "无需评审"
	}
	return s
}
