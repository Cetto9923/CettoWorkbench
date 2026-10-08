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
	errStoryNotFound       = errors.New("研发需求不存在")
	errHomeActionForbidden = errors.New("无权办理该需求")
	errHomeActionConflict  = errors.New("需求状态已变化，请刷新后重试")
	errDeliverBlocked      = errors.New("存在严重缺陷未关闭，禁止发起交付")
	errUrgeNoRecipient     = errors.New("未找到验收责任人")
	errUrgeChannel         = errors.New("不支持的催办渠道")
	errUrgeType            = errors.New("催办类型无效")
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
	case "acceptance":
		if row.Status != "testing" && row.Status != "waitacceptance" {
			return errHomeActionConflict
		}
		return acceptDemandViaZentao(ctx, s.ztAPI, acceptDemandViaZentaoReq{DemandID: int64(id), Acceptance: "yes", AssignedTo: row.AssignedTo, Comment: homeActionComment(comment, "工作台验收完成")})
	case "deliver":
		if row.Status != "acceptanced" {
			return errHomeActionConflict
		}
		if err := s.checkDemandDeliverBlockers(ctx, id); err != nil {
			return err
		}
		detail, err := s.repo.FindDeliverDemand(ctx, id)
		if err != nil {
			return err
		}
		return s.DeliverDemand(ctx, actor, DemandDeliverReq{ID: id, DeliverDate: detail.DeliverDate, IsCarReview: detail.IsCarReview,
			IsGrayVerifyPlan: detail.IsGrayVerifyPlan, VerifyDate: detail.VerifyDate, VerifyPlan: detail.VerifyPlan, Verifier: detail.VeriFier, Comment: comment})
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
	DemandID       uint                   `json:"demandId"`
	Title          string                 `json:"title"`
	Status         string                 `json:"status"`
	StatusLabel    string                 `json:"statusLabel"`
	StageLabel     string                 `json:"stageLabel"`
	Reason         string                 `json:"reason"`
	RecipientTip   string                 `json:"recipientTip"`
	RecipientSrc   string                 `json:"recipientSrc"`
	Recipients     []UrgePreviewRecipient `json:"recipients"`
	MessagePreview string                 `json:"messagePreview"`
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
	preview := buildAcceptanceUrgeMessage(row, labelList, "")
	return &UrgePreview{
		DemandID:       row.ID,
		Title:          strings.TrimSpace(row.Name),
		Status:         row.Status,
		StatusLabel:    homeAcceptanceStatusLabel(row.Status),
		StageLabel:     "验收",
		Reason:         "请尽快完成验收",
		RecipientSrc:   string(source),
		Recipients:     recs,
		MessagePreview: preview,
	}, nil
}

func (s *Service) UrgeHomeDemand(ctx context.Context, actor *model.User, id uint, req UrgeHomeDemandReq) (bool, error) {
	if s == nil || s.repo == nil || actor == nil || strings.TrimSpace(actor.Account) == "" {
		return false, errHomeActionForbidden
	}
	if req.UrgeType != "accept" {
		return false, errUrgeType
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

// canUrgeHomeAcceptance 与首页一致：最近一次发起验收的人催办，验收负责人不催自己。
func (r *Repo) canUrgeHomeAcceptance(row *homeActionDemandRow, account string) bool {
	if row == nil || row.Status != "waitacceptance" || !hasHomeAccount(row.AcceptanceInitiator, account) {
		return false
	}
	owner := acceptanceOwner(row.RD, row.AssignedTo)
	return owner != "" && !hasHomeAccount(owner, account)
}

type homeAcceptanceRecipientSource string

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

// homeAcceptanceRecipients 取 RD，未填时取 assignedTo。
func homeAcceptanceRecipients(row *homeActionDemandRow) []string {
	accounts, _ := homeAcceptanceRecipientsWithSource(row)
	return accounts
}

func homeAcceptanceRecipientsWithSource(row *homeActionDemandRow) ([]string, homeAcceptanceRecipientSource) {
	if row == nil {
		return nil, ""
	}
	if rd := strings.TrimSpace(row.RD); rd != "" {
		return []string{rd}, "RD"
	}
	if assigned := strings.TrimSpace(row.AssignedTo); assigned != "" {
		return []string{assigned}, "assignedTo"
	}
	return nil, ""
}
