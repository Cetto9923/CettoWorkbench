// =============================================================================
// 文件: internal/pkg/zentao/demand_client.go
// 模块: 基础设施
// 类型: infra
// 职责: 封装禅道需求相关业务接口调用（澄清与 AI 故事生成）。
// 依赖: 无
// =============================================================================

package zentao

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DemandReviewParams 需求评审请求参数。
type DemandReviewParams struct {
	DemandID    uint   `json:"demandId"`
	Account     string `json:"account"`
	Result      string `json:"result"`      // pass | refuse
	IsNeedFocus string `json:"isNeedFocus"` // 是否重点关注 0 | 1
	Comment     string `json:"comment"`     // 评审意见
	Mailto      string `json:"mailto"`      // 抄送通知人
}

// ReviewDemand 调用禅道原生 demandReview 接口完成业务需求评审。
func (c *Client) ReviewDemand(ctx context.Context, p DemandReviewParams) error {
	if c == nil {
		return fmt.Errorf("zentao client is nil")
	}
	if AccountFrom(ctx) == "" && p.Account != "" {
		ctx = WithAccount(ctx, p.Account)
	}
	postData := map[string]string{
		"result":      p.Result,
		"isNeedFocus": p.IsNeedFocus,
		"comment":     p.Comment,
	}
	if strings.TrimSpace(p.Mailto) != "" {
		postData["mailto"] = strings.TrimSpace(p.Mailto)
	}
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/demand/%d/review", p.DemandID), postData, nil)
}

// WithdrawDemandReviewParams 撤回需求评审参数。
type WithdrawDemandReviewParams struct {
	DemandID uint   `json:"demandId"`
	Account  string `json:"account"`
	Comment  string `json:"comment"`
}

// WithdrawDemandReview 调用禅道原生 demandWithdrawReview 接口撤回评审。
func (c *Client) WithdrawDemandReview(ctx context.Context, p WithdrawDemandReviewParams) error {
	if c == nil {
		return fmt.Errorf("zentao client is nil")
	}
	if AccountFrom(ctx) == "" && p.Account != "" {
		ctx = WithAccount(ctx, p.Account)
	}
	postData := map[string]string{
		"comment": p.Comment,
	}
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/demand/%d/withdrawReview", p.DemandID), postData, nil)
}

// SubmitDemandReviewParams 提交需求评审参数。
type SubmitDemandReviewParams struct {
	DemandID uint     `json:"demandId"`
	Account  string   `json:"account"`
	Reviewer []string `json:"reviewer"`
	Comment  string   `json:"comment"`
}

// SubmitDemandReview 调用禅道原生 demandSubmitReview 接口提交评审。
func (c *Client) SubmitDemandReview(ctx context.Context, p SubmitDemandReviewParams) error {
	if c == nil {
		return fmt.Errorf("zentao client is nil")
	}
	if AccountFrom(ctx) == "" && p.Account != "" {
		ctx = WithAccount(ctx, p.Account)
	}
	postData := map[string]any{
		"comment": p.Comment,
	}
	if len(p.Reviewer) > 0 {
		postData["reviewer"] = p.Reviewer
	}
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/demand/%d/submitReview", p.DemandID), postData, nil)
}

// DemandClarifyParams 需求澄清请求参数。
type DemandClarifyParams struct {
	DemandID              uint              `json:"demandId"`
	Account               string            `json:"account"`
	Category              string            `json:"category"`
	BRA                   string            `json:"BRA"`
	QD                    string            `json:"QD"`
	RD                    string            `json:"RD"`
	ClarifyDesc           string            `json:"clarifyDesc"`
	ScaleEstimation       int               `json:"scaleEstimation"`
	IsNewProduct          string            `json:"isNewProduct"`
	IsRelatedAccounts     string            `json:"isRelatedAccounts"`
	IsNewFunction         string            `json:"isNewFunction"`
	IsOtherImportantOrder string            `json:"isOtherImportantOrder"`
	MultiLegalPersonLogo  string            `json:"multiLegalPersonLogo"`
	Status                string            `json:"status"`
	Comment               string            `json:"comment"`
	Products              []string          `json:"products"`
	PM                    []string          `json:"PM"`
	DemandCompletionDate  []string          `json:"demandCompletionDate"`
	SystemClarifyDesc     []string          `json:"systemClarifyDesc"`
	IsAdditionalInfo      [][]string        `json:"isAdditionalInfo"`
	AdditionalInfo        []string          `json:"additionalInfo"`
	IsMainSystem          map[string]string `json:"isMainSystem"`
	ClarifyIDList         []string          `json:"id"`
	UserStoryNO           []int             `json:"userStoryNO"`
	UserStoryChecked      map[string]string `json:"userStoryChecked"`
	UserStoryID           []string          `json:"userStoryID"`
	Role                  []string          `json:"role"`
	GV                    []string          `json:"gv"`
	EntryProductID        []string          `json:"entryProductID"`
	Point                 []string          `json:"point"`
	Revpoint              []string          `json:"revpoint"`
	SourceType            []string          `json:"sourceType"`
	AICode                []any             `json:"aiCode"`
}

// ClarifyDemand 调用禅道原生 demandClarify 接口完成需求澄清。
func (c *Client) ClarifyDemand(ctx context.Context, p DemandClarifyParams) error {
	if c == nil {
		return fmt.Errorf("zentao client is nil")
	}
	if AccountFrom(ctx) == "" && p.Account != "" {
		ctx = WithAccount(ctx, p.Account)
	}
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/demand/%d/clarify", p.DemandID), p, nil)
}

// GenerateAIUserStoryParams AI 用户故事生成参数。
type GenerateAIUserStoryParams struct {
	DemandID     uint   `json:"demandId"`
	Account      string `json:"account"`
	InputContent string `json:"inputContent"`
	Source       string `json:"source"`
}

// AIUserStoryResp AI 生成用户故事响应。
type AIUserStoryResp struct {
	Result       string `json:"result"`
	Content      string `json:"content"`
	ThinkContent string `json:"thinkContent"`
	AICodes      []int  `json:"aiCodes"`
}

// GenerateAIUserStory 调用禅道原生 demandAIGenerate 接口生成用户故事。
func (c *Client) GenerateAIUserStory(ctx context.Context, p GenerateAIUserStoryParams) (*AIUserStoryResp, error) {
	if c == nil {
		return nil, fmt.Errorf("zentao client is nil")
	}
	payload := map[string]string{
		"inputContent": p.InputContent,
		"source":       p.Source,
	}
	if payload["source"] == "" {
		payload["source"] = "clarify"
	}
	var out AIUserStoryResp
	if err := c.Do(ctx, http.MethodPost, fmt.Sprintf("/demand/%d/aiGenerate", p.DemandID), payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// FollowDemandObject 调用禅道 demand::ajaxFollowObject（common::followObject）。
func (c *Client) FollowDemandObject(ctx context.Context, account string, demandID int64) error {
	return c.toggleDemandFollow(ctx, account, demandID, true)
}

// UnfollowDemandObject 调用禅道 demand::ajaxUnfollowObject（common::unfollowObject）。
func (c *Client) UnfollowDemandObject(ctx context.Context, account string, demandID int64) error {
	return c.toggleDemandFollow(ctx, account, demandID, false)
}

func (c *Client) toggleDemandFollow(ctx context.Context, account string, demandID int64, follow bool) error {
	if c == nil {
		return fmt.Errorf("zentao client is nil")
	}
	account = strings.TrimSpace(account)
	if account == "" || demandID <= 0 {
		return fmt.Errorf("invalid follow request")
	}
	token, err := c.getToken(ctx, account)
	if err != nil {
		return err
	}
	method := "ajaxUnfollowObject"
	if follow {
		method = "ajaxFollowObject"
	}
	base := strings.TrimRight(strings.TrimSpace(zentaoCfg.URL), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(c.apiBase), "/")
		base = strings.TrimSuffix(base, "/api.php/v1")
		base = strings.TrimSuffix(base, "/v1")
		base = strings.TrimSuffix(base, "/api.php")
		base = strings.TrimRight(base, "/")
	}
	reqURL := fmt.Sprintf("%s/demand-%s-demand-%d.html", base, method, demandID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Token", token)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("无法连接禅道服务器: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("禅道 API 错误: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	out := strings.TrimSpace(string(body))
	if out != "0" && out != "1" {
		if strings.Contains(out, "<html") || strings.Contains(out, "<!DOCTYPE") {
			return fmt.Errorf("禅道鉴权失败: session rejected follow ajax")
		}
		return fmt.Errorf("禅道关注接口异常响应: %s", out)
	}
	return nil
}
