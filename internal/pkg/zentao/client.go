// =============================================================================
// 文件: internal/pkg/zentao/client.go
// 模块: 基础设施
// 类型: infra
// 职责: 封装禅道原生 REST API 客户端，负责身份凭证换取与原生业务接口代理。
// 依赖: net/http
// =============================================================================

package zentao

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrZentaoUnreachable = errors.New("无法连接禅道服务器")
	ErrZentaoAuthFailed  = errors.New("禅道鉴权失败")
	ErrZentaoAPIError    = errors.New("禅道接口执行异常")
)

// Client 禅道原生 API 客户端。
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建客户端实例。
func NewClient(baseURL string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(zentaoCfg.API, "/")
		if baseURL == "" {
			baseURL = strings.TrimRight(zentaoCfg.URL, "/")
		}
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// DefaultClient 返回基于全局配置的默认客户端。
func DefaultClient() *Client {
	return NewClient(zentaoCfg.API)
}

type tokenResponse struct {
	Token   string `json:"token"`
	Message string `json:"message"`
	Ret     []struct {
		ReturnCode string `json:"ReturnCode"`
		ReturnMsg  string `json:"ReturnMsg"`
	} `json:"Ret"`
}

// GetUserToken 为指定账号获取/换取禅道 API Session Token。
func (c *Client) GetUserToken(ctx context.Context, account string) (string, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return "", fmt.Errorf("%w: account is empty", ErrZentaoAuthFailed)
	}

	payload := map[string]string{
		"account": account,
		"type":    "app", // 禅道原生内部免密身份识别模式
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	reqURL := fmt.Sprintf("%s/api.php/v1/tokens", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrZentaoUnreachable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("%w: parse token response failed: %v", ErrZentaoAPIError, err)
	}

	if tokenResp.Token == "" {
		errMsg := tokenResp.Message
		if len(tokenResp.Ret) > 0 && tokenResp.Ret[0].ReturnMsg != "" {
			errMsg = tokenResp.Ret[0].ReturnMsg
		}
		if errMsg == "" {
			errMsg = string(respBody)
		}
		return "", fmt.Errorf("%w: %s", ErrZentaoAuthFailed, errMsg)
	}

	return tokenResp.Token, nil
}

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

	reqURL := fmt.Sprintf("%s/api.php/v1/demand/%d/review", c.baseURL, p.DemandID)
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
		var errObj struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.Unmarshal(respBytes, &errObj)
		errMsg := errObj.Message
		if errMsg == "" {
			errMsg = errObj.Error
		}
		if errMsg == "" {
			errMsg = string(respBytes)
		}
		return fmt.Errorf("%w (%d): %s", ErrZentaoAPIError, resp.StatusCode, errMsg)
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

	reqURL := fmt.Sprintf("%s/api.php/v1/demand/%d/withdrawReview", c.baseURL, p.DemandID)
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
		var errObj struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.Unmarshal(respBytes, &errObj)
		errMsg := errObj.Message
		if errMsg == "" {
			errMsg = errObj.Error
		}
		if errMsg == "" {
			errMsg = string(respBytes)
		}
		return fmt.Errorf("%w (%d): %s", ErrZentaoAPIError, resp.StatusCode, errMsg)
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

	reqURL := fmt.Sprintf("%s/api.php/v1/demand/%d/submitReview", c.baseURL, p.DemandID)
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
		var errObj struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.Unmarshal(respBytes, &errObj)
		errMsg := errObj.Message
		if errMsg == "" {
			errMsg = errObj.Error
		}
		if errMsg == "" {
			errMsg = string(respBytes)
		}
		return fmt.Errorf("%w (%d): %s", ErrZentaoAPIError, resp.StatusCode, errMsg)
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

	reqURL := fmt.Sprintf("%s/api.php/v1/demand/%d/clarify", c.baseURL, p.DemandID)
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

	reqURL := fmt.Sprintf("%s/api.php/v1/demand/%d/aiGenerate", c.baseURL, p.DemandID)
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

func parseZentaoAPIError(respBytes []byte, statusCode int) error {
	var errObj struct {
		Message string `json:"message"`
		Error   any    `json:"error"`
	}
	_ = json.Unmarshal(respBytes, &errObj)
	errMsg := strings.TrimSpace(errObj.Message)
	if errMsg == "" && errObj.Error != nil {
		switch e := errObj.Error.(type) {
		case string:
			errMsg = strings.TrimSpace(e)
		case []any:
			var parts []string
			for _, item := range e {
				parts = append(parts, fmt.Sprintf("%v", item))
			}
			errMsg = strings.Join(parts, "; ")
		case map[string]any:
			var parts []string
			for k, v := range e {
				parts = append(parts, fmt.Sprintf("%s: %v", k, v))
			}
			errMsg = strings.Join(parts, "; ")
		default:
			errMsg = fmt.Sprintf("%v", e)
		}
	}
	if errMsg == "" {
		errMsg = string(respBytes)
	}
	if errMsg == "Array" {
		errMsg = "数据校验失败，请检查表单各项必填项与输入格式"
	}
	return fmt.Errorf("%w (%d): %s", ErrZentaoAPIError, statusCode, errMsg)
}
