// =============================================================================
// 文件: internal/pkg/zentao/demand_client.go
// 模块: 基础设施
// 类型: infra
// 职责: 封装禅道需求相关业务接口调用（评审、撤回评审、提审、澄清、AI故事生成、关注）。
// 依赖: context, net/http
// =============================================================================

package zentao

import (
	"bytes"
	"context"
	"encoding/json"
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
	token, err := c.GetUserToken(ctx, p.Account)
	if err != nil {
		return err
	}

	reqURL := c.apiURL(fmt.Sprintf("/demand/%d/review", p.DemandID))
	postData := map[string]string{
		"result":      p.Result,
		"isNeedFocus": p.IsNeedFocus,
		"comment":     p.Comment,
	}
	if strings.TrimSpace(p.Mailto) != "" {
		postData["mailto"] = strings.TrimSpace(p.Mailto)
	}

	dataBytes, err := json.Marshal(postData)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(dataBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrZentaoUnreachable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseZentaoAPIError(respBytes, resp.StatusCode)
	}

	return nil
}

// WithdrawDemandReviewParams 撤回需求评审参数。
type WithdrawDemandReviewParams struct {
	DemandID uint   `json:"demandId"`
	Account  string `json:"account"`
	Comment  string `json:"comment"`
}

// WithdrawDemandReview 调用禅道原生 demandWithdrawReview 接口撤回评审。
func (c *Client) WithdrawDemandReview(ctx context.Context, p WithdrawDemandReviewParams) error {
	token, err := c.GetUserToken(ctx, p.Account)
	if err != nil {
		return err
	}

	reqURL := c.apiURL(fmt.Sprintf("/demand/%d/withdrawReview", p.DemandID))
	postData := map[string]string{
		"comment": p.Comment,
	}
	dataBytes, err := json.Marshal(postData)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(dataBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrZentaoUnreachable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseZentaoAPIError(respBytes, resp.StatusCode)
	}
	return nil
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
	token, err := c.GetUserToken(ctx, p.Account)
	if err != nil {
		return err
	}

	reqURL := c.apiURL(fmt.Sprintf("/demand/%d/submitReview", p.DemandID))
	postData := map[string]any{
		"comment": p.Comment,
	}
	if len(p.Reviewer) > 0 {
		postData["reviewer"] = p.Reviewer
	}
	dataBytes, err := json.Marshal(postData)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(dataBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrZentaoUnreachable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseZentaoAPIError(respBytes, resp.StatusCode)
	}
	return nil
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
	token, err := c.GetUserToken(ctx, p.Account)
	if err != nil {
		return err
	}

	reqURL := c.apiURL(fmt.Sprintf("/demand/%d/clarify", p.DemandID))
	dataBytes, err := json.Marshal(p)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(dataBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrZentaoUnreachable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseZentaoAPIError(respBytes, resp.StatusCode)
	}
	return nil
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
	token, err := c.GetUserToken(ctx, p.Account)
	if err != nil {
		return nil, err
	}

	reqURL := c.apiURL(fmt.Sprintf("/demand/%d/aiGenerate", p.DemandID))
	payload := map[string]string{
		"inputContent": p.InputContent,
		"source":       p.Source,
	}
	if payload["source"] == "" {
		payload["source"] = "clarify"
	}
	dataBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(dataBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrZentaoUnreachable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseZentaoAPIError(respBytes, resp.StatusCode)
	}

	var out AIUserStoryResp
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("decode ai resp failed: %w", err)
	}
	return &out, nil
}

// FollowDemandObject 调用禅道 demand::ajaxFollowObject（common::followObject），
// 会同步关注子需求，返回正文为 followed 状态 "0"|"1"。
func (c *Client) FollowDemandObject(ctx context.Context, account string, demandID int64) error {
	return c.toggleDemandFollow(ctx, account, demandID, true)
}

// UnfollowDemandObject 调用禅道 demand::ajaxUnfollowObject（common::unfollowObject）。
func (c *Client) UnfollowDemandObject(ctx context.Context, account string, demandID int64) error {
	return c.toggleDemandFollow(ctx, account, demandID, false)
}

func (c *Client) toggleDemandFollow(ctx context.Context, account string, demandID int64, follow bool) error {
	account = strings.TrimSpace(account)
	if account == "" || demandID <= 0 {
		return fmt.Errorf("%w: invalid follow request", ErrZentaoAPIError)
	}
	token, err := c.GetUserToken(ctx, account)
	if err != nil {
		return err
	}
	method := "ajaxUnfollowObject"
	if follow {
		method = "ajaxFollowObject"
	}
	// PATH_INFO: /demand-ajaxFollowObject-demand-{id}.html （objectType + objectID）
	reqURL := c.webURL(fmt.Sprintf("/demand-%s-demand-%d.html", method, demandID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Token", token)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrZentaoUnreachable, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w (%d): %s", ErrZentaoAPIError, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	// 成功正文通常为 "0" / "1"；登录页 HTML 视为失败。
	out := strings.TrimSpace(string(body))
	if out != "0" && out != "1" {
		if strings.Contains(out, "<html") || strings.Contains(out, "<!DOCTYPE") {
			return fmt.Errorf("%w: zenTao session rejected follow ajax", ErrZentaoAuthFailed)
		}
		return fmt.Errorf("%w: unexpected follow response: %s", ErrZentaoAPIError, truncateRunes(out, 120))
	}
	return nil
}
