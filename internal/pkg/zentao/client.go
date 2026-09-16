// =============================================================================
// 文件: internal/pkg/zentao/client.go
// 模块: 基础设施
// 类型: infra
// 职责: 禅道 REST API 通用客户端：按当前账号 type=po 免密取 Token、带鉴权发请求，并落盘 api 请求日志。
// 依赖: internal/config
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
	"sync"
	"time"

	"workbench/internal/config"
)

// tokenTTL 默认 Token 缓存时长（对齐工作台 session 常见 2h）。
const tokenTTL = 2 * time.Hour

type tokenCacheEntry struct {
	token string
	at    time.Time
}

// Client 禅道 OpenAPI 客户端（api.php/v1）。
// 鉴权统一：从 ctx 取当前账号，POST /tokens {account, type:"po"} 免密登录。
type Client struct {
	apiBase string
	http    *http.Client

	mu     sync.Mutex
	tokens map[string]tokenCacheEntry // 按账号缓存 Token
	now    func() time.Time
}

// NewClient 根据配置创建客户端。api 为空时后续调用会报错。
func NewClient(cfg config.ZentaoConfig) *Client {
	return &Client{
		apiBase: strings.TrimRight(strings.TrimSpace(cfg.API), "/"),
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		tokens: map[string]tokenCacheEntry{},
		now:    time.Now,
	}
}

func (c *Client) currentTime() time.Time {
	if c != nil && c.now != nil {
		return c.now()
	}
	return time.Now()
}

func (c *Client) cachedTokenLocked(account string) (string, bool) {
	if c.tokens == nil {
		return "", false
	}
	e, ok := c.tokens[account]
	if !ok || e.token == "" || e.at.IsZero() {
		return "", false
	}
	if c.currentTime().Sub(e.at) >= tokenTTL {
		delete(c.tokens, account)
		return "", false
	}
	return e.token, true
}

// getToken 按账号取禅道 Token（type=po）；命中未过期缓存则直接返回。
func (c *Client) getToken(ctx context.Context, account string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("zentao client is nil")
	}
	c.mu.Lock()
	if tok, ok := c.cachedTokenLocked(account); ok {
		c.mu.Unlock()
		return tok, nil
	}
	c.mu.Unlock()
	return c.refreshTokenPO(ctx, account)
}

// refreshTokenPO POST /tokens，body 为 {account, type:"po"}。
func (c *Client) refreshTokenPO(ctx context.Context, account string) (string, error) {
	if c.apiBase == "" {
		return "", fmt.Errorf("zentao api 未配置")
	}
	account = strings.TrimSpace(account)
	if account == "" {
		return "", fmt.Errorf("未登录或禅道账号为空")
	}
	body := map[string]string{
		"account": account,
		"type":    "po",
	}
	var resp struct {
		Token   string `json:"token"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := c.doRaw(ctx, http.MethodPost, "/tokens", "", body, &resp); err != nil {
		return "", err
	}
	tok := strings.TrimSpace(resp.Token)
	if tok == "" {
		msg := strings.TrimSpace(resp.Message)
		if msg == "" {
			msg = strings.TrimSpace(resp.Error)
		}
		if msg == "" {
			msg = "获取禅道 Token 失败"
		}
		return "", fmt.Errorf("%s", msg)
	}

	c.mu.Lock()
	if c.tokens == nil {
		c.tokens = map[string]tokenCacheEntry{}
	}
	c.tokens[account] = tokenCacheEntry{token: tok, at: c.currentTime()}
	c.mu.Unlock()
	return tok, nil
}

func (c *Client) clearToken(account string) {
	c.mu.Lock()
	if c.tokens != nil {
		delete(c.tokens, account)
	}
	c.mu.Unlock()
}

// Do 带当前登录账号的 Token 调用禅道 API；path 以 / 开头（相对 apiBase）。
// 账号来自 ctx（RequireLogin 注入的 WithAccount）；401 时清该账号缓存并重试一次。
func (c *Client) Do(ctx context.Context, method, path string, body any, out any) error {
	account := AccountFrom(ctx)
	if account == "" {
		return fmt.Errorf("未登录或禅道账号为空")
	}
	tok, err := c.getToken(ctx, account)
	if err != nil {
		return err
	}
	err = c.doRaw(ctx, method, path, tok, body, out)
	if err == nil {
		return nil
	}
	if !isUnauthorized(err) {
		return err
	}
	c.clearToken(account)
	tok, err = c.refreshTokenPO(ctx, account)
	if err != nil {
		return err
	}
	return c.doRaw(ctx, method, path, tok, body, out)
}

type apiError struct {
	status int
	msg    string
}

func (e *apiError) Error() string {
	if e == nil {
		return ""
	}
	if e.msg != "" {
		return e.msg
	}
	return fmt.Sprintf("禅道 API 错误: HTTP %d", e.status)
}

func isUnauthorized(err error) bool {
	ae, ok := err.(*apiError)
	return ok && ae.status == http.StatusUnauthorized
}

func (c *Client) doRaw(ctx context.Context, method, path, token string, body any, out any) error {
	if c == nil {
		return fmt.Errorf("zentao client is nil")
	}
	if c.apiBase == "" {
		return fmt.Errorf("zentao api 未配置")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	url := c.apiBase + path
	start := c.currentTime()

	var (
		status   int
		respBody []byte
		callErr  error
	)
	defer func() {
		elapsed := c.currentTime().Sub(start)
		entry := apiLogEntry{
			Time:     formatAPILogTime(start),
			Elapsed:  formatAPILogDuration(elapsed),
			Method:   method,
			Path:     path,
			URL:      url,
			Status:   status,
			Success:  status >= 200 && status < 300 && callErr == nil,
			Request:  encodeRequestBody(body),
			Response: encodeResponseBody(respBody),
		}
		if callErr != nil {
			entry.Error = callErr.Error()
		}
		logAPICall(entry)
	}()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			callErr = fmt.Errorf("marshal body: %w", err)
			return callErr
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		callErr = err
		return callErr
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Token", token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		callErr = err
		return callErr
	}
	defer resp.Body.Close()
	status = resp.StatusCode

	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		callErr = err
		return callErr
	}

	if resp.StatusCode == http.StatusUnauthorized {
		callErr = &apiError{status: resp.StatusCode, msg: "Unauthorized"}
		return callErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := parseAPIErrorMessage(respBody)
		if msg == "" {
			msg = fmt.Sprintf("禅道 API 错误: HTTP %d", resp.StatusCode)
		}
		callErr = &apiError{status: resp.StatusCode, msg: msg}
		return callErr
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		callErr = fmt.Errorf("decode response: %w", err)
		return callErr
	}
	return nil
}

func parseAPIErrorMessage(raw []byte) string {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return strings.TrimSpace(string(raw))
	}
	for _, key := range []string{"error", "message", "msg"} {
		if v, ok := obj[key]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			default:
				b, _ := json.Marshal(t)
				if s := strings.TrimSpace(string(b)); s != "" && s != "null" {
					return s
				}
			}
		}
	}
	return ""
}
