package po

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"workbench/internal/model"
)

var (
	errHomeActionNotFound  = errors.New("需求不存在")
	errHomeActionForbidden = errors.New("无权办理该需求")
	errHomeActionConflict  = errors.New("需求状态已变化，请刷新后重试")
	errUrgeNoRecipient     = errors.New("未找到验收责任人")
	errUrgeChannel         = errors.New("不支持的催办渠道")
	errUrgeDuplicate       = errors.New("催办重复")
)

func (s *Service) homeAction(ctx context.Context, actor *model.User, id uint, action string, comment string) error {
	if s == nil || s.repo == nil || id == 0 {
		return errHomeActionNotFound
	}
	account := ""
	if actor != nil {
		account = strings.TrimSpace(actor.Account)
	}
	if account == "" {
		return errHomeActionForbidden
	}
	row, err := s.repo.findHomeActionDemand(ctx, id)
	if err != nil {
		return err
	}
	if !s.repo.homeActionAuthorized(row, account, action) {
		return errHomeActionForbidden
	}
	switch action {
	case "clarify":
		if row.Status != "active" {
			return errHomeActionConflict
		}
		return s.repo.updateHomeDemandStatus(ctx, id, "active", "clarified", account, "clarify", homeActionComment(comment, "工作台澄清完成"), false)
	case "acceptance":
		if row.Status != "testing" && row.Status != "waitacceptance" {
			return errHomeActionConflict
		}
		return s.repo.updateHomeDemandStatus(ctx, id, row.Status, "acceptanced", account, "verified", homeActionComment(comment, "工作台验收完成"), true)
	case "deliver":
		if row.Status != "acceptanced" {
			return errHomeActionConflict
		}
		return s.repo.updateHomeDemandStatus(ctx, id, "acceptanced", "waitdeliver", account, "deliver", homeActionComment(comment, "工作台发起交付"), false)
	case "urge":
		return s.repo.insertHomeDemandAction(ctx, id, account, "reminded", homeActionComment(comment, "工作台催办验收"))
	default:
		return errHomeActionConflict
	}
}

func homeActionComment(comment, fallback string) string {
	if strings.TrimSpace(comment) == "" {
		return fallback
	}
	return strings.TrimSpace(comment)
}

func (s *Service) ClarifyHomeDemand(ctx context.Context, actor *model.User, id uint, comment string) error {
	return s.homeAction(ctx, actor, id, "clarify", comment)
}
func (s *Service) AcceptHomeDemand(ctx context.Context, actor *model.User, id uint, comment string) error {
	return s.homeAction(ctx, actor, id, "acceptance", comment)
}
func (s *Service) DeliverHomeDemand(ctx context.Context, actor *model.User, id uint, comment string) error {
	return s.homeAction(ctx, actor, id, "deliver", comment)
}

// UrgePreviewRecipient 催办预览收件人展示。
type UrgePreviewRecipient struct {
	Account string `json:"account"`
	Label   string `json:"label"`
}

// UrgePreview 催办弹窗预填数据。
type UrgePreview struct {
	DemandID      uint                   `json:"demandId"`
	Title         string                 `json:"title"`
	Status        string                 `json:"status"`
	StatusLabel   string                 `json:"statusLabel"`
	StageLabel    string                 `json:"stageLabel"`
	Reason        string                 `json:"reason"`
	Recipients    []UrgePreviewRecipient `json:"recipients"`
	MessagePreview string                `json:"messagePreview"`
}

func homeAcceptanceStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "testing":
		return "测试中"
	case "waitacceptance":
		return "待验收"
	case "acceptanced":
		return "已验收"
	default:
		return strings.TrimSpace(status)
	}
}

func buildAcceptanceUrgeMessage(row *homeActionDemandRow, recipientLabels []string, note string) string {
	if row == nil {
		return "请尽快完成验收"
	}
	title := strings.TrimSpace(row.Name)
	if title == "" {
		title = "（未命名需求）"
	}
	statusLabel := homeAcceptanceStatusLabel(row.Status)
	who := "验收责任人"
	if len(recipientLabels) > 0 {
		who = strings.Join(recipientLabels, "、")
	}
	msg := fmt.Sprintf("【催办验收】业务需求 US%d《%s》：当前状态「%s」，请 %s 尽快完成业务验收（核对测试结论与交付条件），以免影响后续交付上线。", row.ID, title, statusLabel, who)
	note = strings.TrimSpace(note)
	if note != "" {
		msg += "\n补充说明：" + note
	}
	return msg
}

func (s *Service) UrgeHomeDemandPreview(ctx context.Context, actor *model.User, id uint) (*UrgePreview, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errHomeActionForbidden
	}
	row, err := s.repo.findHomeActionDemand(ctx, id)
	if err != nil {
		return nil, err
	}
	if !s.repo.canUrgeHomeAcceptance(row, actor.Account) {
		return nil, errHomeActionForbidden
	}
	accounts, source := homeAcceptanceRecipientsWithSource(row)
	accounts = excludeHomeAccount(accounts, actor.Account)
	if len(accounts) == 0 {
		return nil, errUrgeNoRecipient
	}
	labels, err := s.repo.homeUserLabels(ctx, accounts)
	if err != nil {
		return nil, err
	}
	recs := make([]UrgePreviewRecipient, 0, len(accounts))
	labelList := make([]string, 0, len(accounts))
	for _, a := range accounts {
		lb := labels[a]
		if lb == "" {
			lb = a
		}
		recs = append(recs, UrgePreviewRecipient{Account: a, Label: lb})
		labelList = append(labelList, lb)
	}
	reason := homeAcceptanceUrgeReason(source)
	preview := buildAcceptanceUrgeMessage(row, labelList, "")
	return &UrgePreview{
		DemandID:       row.ID,
		Title:          strings.TrimSpace(row.Name),
		Status:         row.Status,
		StatusLabel:    homeAcceptanceStatusLabel(row.Status),
		StageLabel:     "验收",
		Reason:         reason,
		Recipients:     recs,
		MessagePreview: preview,
	}, nil
}

func (s *Service) UrgeHomeDemand(ctx context.Context, actor *model.User, id uint, req UrgeHomeDemandReq) (bool, error) {
	if s == nil || s.repo == nil || actor == nil || strings.TrimSpace(actor.Account) == "" {
		return false, errHomeActionForbidden
	}
	if req.UrgeType != "accept" {
		return false, errHomeActionConflict
	}
	if len(req.Channels) != 1 || req.Channels[0] != "inapp" {
		return false, errUrgeChannel
	}
	row, err := s.repo.findHomeActionDemand(ctx, id)
	if err != nil {
		return false, err
	}
	if !s.repo.canUrgeHomeAcceptance(row, actor.Account) {
		return false, errHomeActionForbidden
	}
	recipients, _ := homeAcceptanceRecipientsWithSource(row)
	recipients = excludeHomeAccount(recipients, actor.Account)
	if len(recipients) == 0 {
		return false, errUrgeNoRecipient
	}
	labels, _ := s.repo.homeUserLabels(ctx, recipients)
	labelList := make([]string, 0, len(recipients))
	for _, a := range recipients {
		if lb := labels[a]; lb != "" {
			labelList = append(labelList, lb)
		} else {
			labelList = append(labelList, a)
		}
	}
	message := buildAcceptanceUrgeMessage(row, labelList, req.Comment)
	duplicate, err := s.repo.insertAcceptanceUrge(ctx, row, actor.Account, recipients, message, time.Minute)
	if errors.Is(err, errUrgeDuplicate) {
		return true, nil
	}
	return duplicate, err
}

// canUrgeHomeAcceptance keeps the API authorization identical to the homepage:
// a participant may remind the acceptance owner, but cannot remind themselves.
func (r *Repo) canUrgeHomeAcceptance(row *homeActionDemandRow, account string) bool {
	if row == nil || (row.Status != "testing" && row.Status != "waitacceptance") {
		return false
	}
	if hasHomeAccount(row.Accepter, account) || hasHomeCSV(row.VeriFier, account) {
		return false
	}
	if !r.homeActionAuthorized(row, account, "urge") {
		return false
	}
	// 与预览/提交一致：解析后无人可催（含只剩自己）则不允许开催办。
	recipients, _ := homeAcceptanceRecipientsWithSource(row)
	return len(excludeHomeAccount(recipients, account)) > 0
}

// homeAcceptanceRecipientSource 标记催办对象来源，用于预览提示。
type homeAcceptanceRecipientSource string

const (
	homeAcceptSrcOwner     homeAcceptanceRecipientSource = "accepter"
	homeAcceptSrcOriginator homeAcceptanceRecipientSource = "originator"
	homeAcceptSrcCreatedBy homeAcceptanceRecipientSource = "createdBy"
)

func homeAcceptanceUrgeReason(src homeAcceptanceRecipientSource) string {
	switch src {
	case homeAcceptSrcOriginator:
		return "未配置验收人，已按提出人推荐催办"
	case homeAcceptSrcCreatedBy:
		return "未配置验收人/提出人，已按创建人推荐催办"
	default:
		return "验收待办理，请及时处理"
	}
}

func excludeHomeAccount(accounts []string, account string) []string {
	account = strings.TrimSpace(account)
	if account == "" || len(accounts) == 0 {
		return accounts
	}
	out := make([]string, 0, len(accounts))
	for _, a := range accounts {
		if strings.TrimSpace(a) == account {
			continue
		}
		out = append(out, a)
	}
	return out
}

func appendHomeCSVAccounts(out []string, seen map[string]bool, raw string) []string {
	for _, account := range strings.Split(strings.ReplaceAll(raw, " ", ""), ",") {
		account = strings.TrimSpace(account)
		if account == "" || seen[account] {
			continue
		}
		seen[account] = true
		out = append(out, account)
	}
	return out
}

// homeAcceptanceRecipients 与老仓 resolveUrgeRecipients(acceptance) 对齐：
// 验收人/验证人 → 提出人 originator → 创建人 createdBy。
func homeAcceptanceRecipients(row *homeActionDemandRow) []string {
	accounts, _ := homeAcceptanceRecipientsWithSource(row)
	return accounts
}

func homeAcceptanceRecipientsWithSource(row *homeActionDemandRow) ([]string, homeAcceptanceRecipientSource) {
	if row == nil {
		return nil, ""
	}
	seen := map[string]bool{}
	out := []string{}
	out = appendHomeCSVAccounts(out, seen, row.Accepter)
	out = appendHomeCSVAccounts(out, seen, row.VeriFier)
	if len(out) > 0 {
		return out, homeAcceptSrcOwner
	}
	if originator := strings.TrimSpace(row.Originator); originator != "" {
		return []string{originator}, homeAcceptSrcOriginator
	}
	if created := strings.TrimSpace(row.CreatedBy); created != "" {
		return []string{created}, homeAcceptSrcCreatedBy
	}
	return nil, ""
}
