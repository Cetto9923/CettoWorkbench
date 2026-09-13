// =============================================================================
// 文件: internal/pkg/zentao/client_site.go
// 包:   zentao
// 职责: 站点控制层（PATH_INFO）出站。部分禅道页面动作只在站点侧存在控制层方法，
//       未被 api.php/v1 注册为 REST entry（版本关联需求即属此类），
//       走这里而不是臆造一个不存在的 REST 路径。
// =============================================================================

package zentao

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DoSiteForm 以用户 Token 调用禅道站点控制层动作（PATH_INFO + 表单提交）。
//
// 为什么走站点而不是 REST：这些动作在禅道原生路由表 config/routes.php 中没有对应
// entry，只有 module/<m>/control.php 里的控制层方法；强行拼一个 REST 路径只会得到
// 404 not found。
//
// 为什么必须带同源 Referer：framework/base/router.class.php 在非 api 模式下会把
// "非同源 POST" 的 $_POST 整体清空（CSRF 保护），随后页面照常渲染并返回 HTTP 200。
// 即缺 Referer 时写入会静默丢失，因此这里强制附带，且调用方必须依据响应体判定结果。
func (c *Client) DoSiteForm(ctx context.Context, account, method, path string, form url.Values) error {
	if c == nil {
		return fmt.Errorf("zentao client is nil")
	}
	if c.baseURL == "" {
		return fmt.Errorf("禅道 API 未配置")
	}
	account = strings.TrimSpace(account)
	if account == "" {
		return fmt.Errorf("%w: account is empty", ErrZentaoAuthFailed)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if form == nil {
		form = url.Values{}
	}
	body := form.Encode()

	token, err := c.GetUserToken(ctx, account)
	if err != nil {
		return err
	}

	reqURL := c.webURL(path)
	start := doNow()

	var (
		status   int
		respBody []byte
		callErr  error
	)
	defer func() {
		entry := apiLogEntry{
			Time:     formatAPILogTime(start),
			Elapsed:  formatAPILogDuration(doNow().Sub(start)),
			Method:   method,
			Path:     path,
			URL:      reqURL,
			Status:   status,
			Success:  status >= 200 && status < 300 && callErr == nil,
			Request:  encodeRequestBody(formAsLogValue(form)),
			Response: encodeResponseBody(respBody),
		}
		if callErr != nil {
			entry.Error = callErr.Error()
		}
		logAPICall(entry)
	}()

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bytes.NewReader([]byte(body)))
	if err != nil {
		callErr = err
		return callErr
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", c.siteOrigin())
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
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

	callErr = validateSiteActionResponse(respBody, status)
	return callErr
}

// siteOrigin 返回站点侧同源前缀（scheme://host/），用于构造 Referer。
func (c *Client) siteOrigin() string {
	root := c.webURL("/")
	if !strings.HasSuffix(root, "/") {
		root += "/"
	}
	return root
}

// validateSiteActionResponse 判定站点控制层动作结果。
//
// 站点侧被 CSRF 拦截时返回的是 HTTP 200 + 页面 HTML（静默失败），必须显式识别，
// 否则会把「什么都没写」当成「写入成功」。
func validateSiteActionResponse(body []byte, status int) error {
	if status >= 200 && status < 300 && looksLikeHTML(body) {
		return fmt.Errorf("%w (%d): 禅道站点拒绝该动作（Referer/会话校验未通过）", ErrZentaoAPIError, status)
	}
	return validateDemandReviewResponse(body, status)
}

// looksLikeHTML 判断响应体是否为页面而非动作 JSON。
// 站点页面可达数百 KB，只取前缀判定，避免整段 rune 转换。
func looksLikeHTML(body []byte) bool {
	head := body
	if len(head) > 4096 {
		head = head[:4096]
	}
	text := strings.ToLower(strings.TrimSpace(string(head)))
	if text == "" {
		return false
	}
	return strings.HasPrefix(text, "<") &&
		(strings.Contains(text, "<html") || strings.Contains(text, "<!doctype") || strings.Contains(text, "<div") || strings.Contains(text, "<style"))
}

// formAsLogValue 把表单转为可落盘结构；同名字段保留为数组，避免多值被截断成单个。
func formAsLogValue(form url.Values) any {
	if len(form) == 0 {
		return nil
	}
	out := make(map[string]any, len(form))
	for k, vs := range form {
		if len(vs) == 1 {
			out[k] = vs[0]
			continue
		}
		out[k] = vs
	}
	return out
}
