// =============================================================================
// 文件: internal/pkg/zentao/client_do.go
// 模块: 基础设施
// 类型: infra
// 职责: 统一禅道 REST 出站：用户态 DoAs、服务账号 Do、401 重试与 APILog。
// 依赖: internal/pkg/zentao（client / apilog）
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
)

const userTokenTTL = 2 * time.Hour

type userTokenEntry struct {
	token string
	at    time.Time
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

var (
	userTokenMu    sync.Mutex
	userTokenCache = map[string]userTokenEntry{}
	doNow          = time.Now
)

// DoAs 以指定账号的用户 Token 调用禅道 REST；401 时清缓存并重试一次。
func (c *Client) DoAs(ctx context.Context, account, method, path string, body any, out any) error {
	account = strings.TrimSpace(account)
	if account == "" {
		return fmt.Errorf("%w: account is empty", ErrZentaoAuthFailed)
	}
	tok, err := c.cachedUserToken(ctx, account)
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
	clearUserToken(account)
	tok, err = c.refreshUserToken(ctx, account)
	if err != nil {
		return err
	}
	return c.doRaw(ctx, method, path, tok, body, out)
}

// Do 使用配置中的服务账号调用禅道 REST（内部走 DoAs）。
// 仅当禅道接口必须服务账号时使用；LinkStory/建版本默认优先 DoAs(当前登录 account)。
func (c *Client) Do(ctx context.Context, method, path string, body any, out any) error {
	account := strings.TrimSpace(zentaoCfg.Account)
	if account == "" {
		return fmt.Errorf("%w: zentao account is empty", ErrZentaoAuthFailed)
	}
	return c.DoAs(ctx, account, method, path, body, out)
}

func (c *Client) cachedUserToken(ctx context.Context, account string) (string, error) {
	userTokenMu.Lock()
	if entry, ok := userTokenCache[account]; ok && entry.token != "" && !entry.at.IsZero() {
		if doNow().Sub(entry.at) < userTokenTTL {
			tok := entry.token
			userTokenMu.Unlock()
			return tok, nil
		}
		delete(userTokenCache, account)
	}
	userTokenMu.Unlock()
	return c.refreshUserToken(ctx, account)
}

func (c *Client) refreshUserToken(ctx context.Context, account string) (string, error) {
	tok, err := c.GetUserToken(ctx, account)
	if err != nil {
		return "", err
	}
	userTokenMu.Lock()
	userTokenCache[account] = userTokenEntry{token: tok, at: doNow()}
	userTokenMu.Unlock()
	return tok, nil
}

func clearUserToken(account string) {
	userTokenMu.Lock()
	delete(userTokenCache, account)
	userTokenMu.Unlock()
}

func isUnauthorized(err error) bool {
	ae, ok := err.(*apiError)
	return ok && ae.status == http.StatusUnauthorized
}

func (c *Client) doRaw(ctx context.Context, method, path, token string, body any, out any) error {
	if c == nil {
		return fmt.Errorf("zentao client is nil")
	}
	if c.baseURL == "" {
		return fmt.Errorf("禅道 API 未配置")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	url := c.apiURL(path)
	start := doNow()

	var (
		status   int
		respBody []byte
		callErr  error
	)
	defer func() {
		elapsed := doNow().Sub(start)
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		callErr = fmt.Errorf("%w: %v", ErrZentaoUnreachable, err)
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
		callErr = parseZentaoAPIError(respBody, resp.StatusCode)
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
