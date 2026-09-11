// =============================================================================
// 文件: internal/pkg/zentao/client.go
// 模块: 基础设施
// 类型: infra
// 职责: 封装禅道原生 REST API 客户端核心，负责身份凭证换取、URL 构建与错误解析。
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

// SiteClient 返回面向禅道站点页（zentao.url）的客户端，用于 PATH_INFO 控制层动作。
// DefaultClient 面向 REST API（zentao.api）；关注切换等自定义 ajax 只在站点侧可用。
func SiteClient() *Client {
	return NewClient(zentaoCfg.URL)
}

// apiURL 构造禅道原生 REST API 完整 URL，自动防御 baseURL 重复包含 /api.php/v1 或 /v1 的情况。
func (c *Client) apiURL(path string) string {
	base := strings.TrimRight(c.baseURL, "/")
	base = strings.TrimSuffix(base, "/api.php/v1")
	base = strings.TrimSuffix(base, "/v1")
	base = strings.TrimSuffix(base, "/api.php")
	base = strings.TrimRight(base, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + "/api.php/v1" + path
}

// webURL 构造禅道 Web 页面完整 URL，去除可能混入的 API 后缀。
func (c *Client) webURL(path string) string {
	base := strings.TrimRight(c.baseURL, "/")
	base = strings.TrimSuffix(base, "/api.php/v1")
	base = strings.TrimSuffix(base, "/v1")
	base = strings.TrimSuffix(base, "/api.php")
	base = strings.TrimRight(base, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
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

	reqURL := c.apiURL("/tokens")
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

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
