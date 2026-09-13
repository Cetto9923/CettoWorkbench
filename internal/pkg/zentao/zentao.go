// Package zentao 根据禅道站点地址、requestType 与 m、f 等参数拼接页面链接。
package zentao

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"workbench/internal/config"
)

const indexPath = "/index.php"

var zentaoCfg config.ZentaoConfig

// SetConfig 注册禅道配置，应在应用启动时调用（bootstrap 中传入 cfg.Zentao）。
// URL 是浏览器跳转基址；API 是原生 API 基址，空时客户端回退 URL。
func SetConfig(cfg config.ZentaoConfig) {
	zentaoCfg = cfg
}

// URL 拼接禅道页面链接，站点前缀取自 config.Zentao.URL（zentao.url）。
func URL(m, f string, params ...string) string {
	return URLWithBase(strings.TrimRight(zentaoCfg.URL, "/"), m, f, params...)
}

// URLWithBase 使用指定站点前缀拼接禅道页面链接（base 为空时回退全局配置）。
//
// requestType=PATH_INFO 时优先输出伪静态 /m-f-N.html，以便未登录跳转登录时
// referer 能保留目标路径（GET 的 index.php?m=&f= 会被禅道收成 referer=/，登录后丢详情）。
// 业需/研发需求仍附加 #app=… 壳上下文。无法安全提取数字 ID 时回退 GET。
// requestType=GET（或未识别）时保持 index.php?m=&f= 形态。
func URLWithBase(base, m, f string, params ...string) string {
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = strings.TrimRight(zentaoCfg.URL, "/")
	}
	if base == "" || m == "" || f == "" {
		return ""
	}

	if usePathInfo() {
		if u := pathInfoURL(base, m, f, params...); u != "" {
			return u
		}
	}
	return getURL(base, m, f, params...)
}

func usePathInfo() bool {
	return strings.EqualFold(strings.TrimSpace(zentaoCfg.RequestType), "PATH_INFO")
}

func getURL(base, m, f string, params ...string) string {
	query := fmt.Sprintf("m=%s&f=%s", url.QueryEscape(m), url.QueryEscape(f))
	if len(params) > 0 && params[0] != "" {
		query += "&" + params[0]
		// 禅道 max5 需要 id 参数与业务 ID 参数并存（CRCBWorkbench buildZentaoURL 双设）。
		if val, ok := extractParamsValue(params[0]); ok {
			if !strings.Contains(params[0], "id=") {
				query += "&id=" + url.QueryEscape(val)
			}
			if m == "story" && !strings.Contains(params[0], "storyID=") {
				query += "&storyID=" + url.QueryEscape(val)
			}
		}
	}
	u := base + indexPath + "?" + query
	return appendAppHash(u, m, f)
}

// pathInfoURL 生成 base/m-f-N.html[#app=…]；ID 非纯数字时返回空以回退 GET。
func pathInfoURL(base, m, f string, params ...string) string {
	raw := ""
	if len(params) > 0 {
		raw = params[0]
	}
	id := extractObjectID(m, raw)
	if id == "" || !isAllDigits(id) {
		return ""
	}
	u := fmt.Sprintf("%s/%s-%s-%s.html", base, m, f, id)
	return appendAppHash(u, m, f)
}

func appendAppHash(u, m, f string) string {
	// 禅道 max5 需求池应用壳：demand view/edit 缺 #app=demandpool 会回落"地盘/首页"。
	if m == "demand" && (f == "view" || f == "edit") {
		return u + "#app=demandpool"
	}
	// 禅道 max5 项目应用壳：story view 缺 #app=project 会回落"地盘/首页"。
	if m == "story" && f == "view" {
		return u + "#app=project"
	}
	return u
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// extractObjectID 从查询参数串提取对象 ID，优先模块业务键（demandID 等），否则 id。
func extractObjectID(m, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		if val, ok := extractParamsValue(raw); ok {
			return val
		}
		return ""
	}
	keys := objectIDKeys(m)
	for _, k := range keys {
		if v := strings.TrimSpace(values.Get(k)); v != "" {
			return v
		}
	}
	// ParseQuery 对无编码的单对 key=value 仍可用；兜底扫一遍
	for _, k := range keys {
		for qk, vs := range values {
			if strings.EqualFold(qk, k) && len(vs) > 0 && strings.TrimSpace(vs[0]) != "" {
				return strings.TrimSpace(vs[0])
			}
		}
	}
	return ""
}

func objectIDKeys(m string) []string {
	switch strings.ToLower(strings.TrimSpace(m)) {
	case "demand":
		return []string{"demandID", "id"}
	case "story":
		return []string{"storyID", "id"}
	case "task":
		return []string{"taskID", "id"}
	case "bug":
		return []string{"bugID", "id"}
	case "product":
		return []string{"productID", "id"}
	case "todo":
		return []string{"todoID", "id"}
	case "project":
		return []string{"projectID", "id"}
	case "execution":
		return []string{"executionID", "id"}
	case "testtask":
		return []string{"testtaskID", "taskID", "id"}
	case "caselib", "testcase", "case":
		return []string{"caseID", "id"}
	case "doc":
		return []string{"docID", "id"}
	case "issue":
		return []string{"issueID", "id"}
	case "risk":
		return []string{"riskID", "id"}
	case "feedback":
		return []string{"feedbackID", "id"}
	case "ticket":
		return []string{"ticketID", "id"}
	case "review":
		return []string{"reviewID", "id"}
	case "approval":
		return []string{"approvalID", "id"}
	case "charter":
		return []string{"id", "projectID"}
	case "build":
		return []string{"buildID", "id"}
	case "release":
		return []string{"releaseID", "id"}
	case "productplan":
		return []string{"planID", "id"}
	default:
		return []string{"id", m + "ID"}
	}
}

// extractParamsValue 从 "key=value" 形式的参数串中取出第一个 value；无 "=" 时返回 false。
func extractParamsValue(raw string) (string, bool) {
	// 仅取第一对，兼容历史 getURL 双设逻辑
	part := raw
	if i := strings.IndexByte(raw, '&'); i >= 0 {
		part = raw[:i]
	}
	_, val, found := strings.Cut(part, "=")
	if !found {
		return "", false
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return "", false
	}
	return val, true
}

// DemandViewURL 业需详情页链接。
func DemandViewURL(demandID uint) string {
	if demandID == 0 {
		return ""
	}
	return URL("demand", "view", fmt.Sprintf("demandID=%d", demandID))
}

// DemandViewURLWithBase 使用指定站点前缀拼接业需详情页链接。
func DemandViewURLWithBase(base string, demandID uint) string {
	if demandID == 0 {
		return ""
	}
	return URLWithBase(base, "demand", "view", fmt.Sprintf("demandID=%d", demandID))
}

// DemandEditURL 业需编辑页链接。
func DemandEditURL(demandID uint) string {
	if demandID == 0 {
		return ""
	}
	return URL("demand", "edit", fmt.Sprintf("demandID=%d", demandID))
}

// DemandEditURLWithBase 使用指定站点前缀拼接业需编辑页链接。
func DemandEditURLWithBase(base string, demandID uint) string {
	if demandID == 0 {
		return ""
	}
	return URLWithBase(base, "demand", "edit", fmt.Sprintf("demandID=%d", demandID))
}

// DemandClarifyURL 业需澄清办理页链接（直达澄清办理，非详情页）。
func DemandClarifyURL(demandID uint) string {
	if demandID == 0 {
		return ""
	}
	return URL("demand", "clarify", fmt.Sprintf("demandID=%d", demandID))
}

// DemandClarifyURLWithBase 使用指定站点前缀拼接业需澄清办理页链接。
func DemandClarifyURLWithBase(base string, demandID uint) string {
	if demandID == 0 {
		return ""
	}
	return URLWithBase(base, "demand", "clarify", fmt.Sprintf("demandID=%d", demandID))
}

// StoryViewURL 研发需求详情页链接。
func StoryViewURL(storyID uint) string {
	if storyID == 0 {
		return ""
	}
	return URL("story", "view", fmt.Sprintf("storyID=%d", storyID))
}

// StoryViewURLWithBase 使用指定站点前缀拼接研发需求详情页链接。
func StoryViewURLWithBase(base string, storyID uint) string {
	if storyID == 0 {
		return ""
	}
	return URLWithBase(base, "story", "view", fmt.Sprintf("storyID=%d", storyID))
}

// ProductViewURL 产品概况页链接。
func ProductViewURL(productID uint) string {
	if productID == 0 {
		return ""
	}
	return URL("product", "view", fmt.Sprintf("productID=%d", productID))
}

// ProductViewURLWithBase 使用指定站点前缀拼接产品概况页链接。
func ProductViewURLWithBase(base string, productID uint) string {
	if productID == 0 {
		return ""
	}
	return URLWithBase(base, "product", "view", fmt.Sprintf("productID=%d", productID))
}

// TaskViewURL 任务详情页链接。
func TaskViewURL(taskID uint) string {
	if taskID == 0 {
		return ""
	}
	return URL("task", "view", fmt.Sprintf("taskID=%d", taskID))
}

// BugViewURL Bug 详情页链接。
func BugViewURL(bugID uint) string {
	if bugID == 0 {
		return ""
	}
	return URL("bug", "view", fmt.Sprintf("bugID=%d", bugID))
}

// CharterViewURL 项目章程详情页链接。禅道 charter::view 按项目打开，必须使用 projectID。
func CharterViewURL(objectID, projectID uint) string {
	if projectID == 0 {
		return ""
	}
	return URL("charter", "view", fmt.Sprintf("projectID=%d", projectID))
}

// CharterViewURLWithBase 使用指定站点前缀拼接项目章程详情页链接。
func CharterViewURLWithBase(base string, objectID, projectID uint) string {
	if projectID == 0 {
		return ""
	}
	return URLWithBase(base, "charter", "view", fmt.Sprintf("projectID=%d", projectID))
}

// BuildguidelineViewURL 项目建设指引详情页链接。禅道 buildguideline::view 按项目打开，必须使用 projectID。
func BuildguidelineViewURL(objectID, projectID uint) string {
	if projectID == 0 {
		return ""
	}
	return URL("buildguideline", "view", fmt.Sprintf("projectID=%d", projectID))
}

// BuildguidelineViewURLWithBase 使用指定站点前缀拼接项目建设指引详情页链接。
func BuildguidelineViewURLWithBase(base string, objectID, projectID uint) string {
	if projectID == 0 {
		return ""
	}
	return URLWithBase(base, "buildguideline", "view", fmt.Sprintf("projectID=%d", projectID))
}

// PlanchangeViewURL 计划变更详情页链接。
func PlanchangeViewURL(objectID uint) string {
	if objectID == 0 {
		return ""
	}
	return URL("planchange", "view", fmt.Sprintf("ID=%d", objectID))
}

// PlanchangeViewURLWithBase 使用指定站点前缀拼接计划变更详情页链接。
func PlanchangeViewURLWithBase(base string, objectID uint) string {
	if objectID == 0 {
		return ""
	}
	return URLWithBase(base, "planchange", "view", fmt.Sprintf("ID=%d", objectID))
}

// ReviewViewURL 项目评审详情页链接。
func ReviewViewURL(objectID uint) string {
	if objectID == 0 {
		return ""
	}
	return URL("review", "view", fmt.Sprintf("reviewID=%d", objectID))
}

// ReviewViewURLWithBase 使用指定站点前缀拼接项目评审详情页链接。
func ReviewViewURLWithBase(base string, objectID uint) string {
	if objectID == 0 {
		return ""
	}
	return URLWithBase(base, "review", "view", fmt.Sprintf("reviewID=%d", objectID))
}

// CaseViewURL 用例详情页链接。
func CaseViewURL(objectID uint) string {
	if objectID == 0 {
		return ""
	}
	return URL("case", "view", fmt.Sprintf("caseID=%d", objectID))
}

// CaseViewURLWithBase 使用指定站点前缀拼接用例详情页链接。
func CaseViewURLWithBase(base string, objectID uint) string {
	if objectID == 0 {
		return ""
	}
	return URLWithBase(base, "case", "view", fmt.Sprintf("caseID=%d", objectID))
}

// TesttaskViewURL 测试单详情页链接。
func TesttaskViewURL(testtaskID uint) string {
	if testtaskID == 0 {
		return ""
	}
	return URL("testtask", "view", fmt.Sprintf("taskID=%d", testtaskID))
}

// IssueViewURL 问题详情页链接。
func IssueViewURL(issueID uint) string {
	if issueID == 0 {
		return ""
	}
	return URL("issue", "view", fmt.Sprintf("issueID=%d", issueID))
}

// IssueViewURLWithBase 使用指定站点前缀拼接问题详情页链接。
func IssueViewURLWithBase(base string, issueID uint) string {
	if issueID == 0 {
		return ""
	}
	return URLWithBase(base, "issue", "view", fmt.Sprintf("issueID=%d", issueID))
}

// RiskViewURL 风险详情页链接。
func RiskViewURL(riskID uint) string {
	if riskID == 0 {
		return ""
	}
	return URL("risk", "view", fmt.Sprintf("riskID=%d", riskID))
}

// RiskViewURLWithBase 使用指定站点前缀拼接风险详情页链接。
func RiskViewURLWithBase(base string, riskID uint) string {
	if riskID == 0 {
		return ""
	}
	return URLWithBase(base, "risk", "view", fmt.Sprintf("riskID=%d", riskID))
}

// WeeklyIndexURL 项目周报主界面链接。
func WeeklyIndexURL(projectID uint, weekStart string) string {
	return WeeklyIndexURLWithBase(strings.TrimRight(zentaoCfg.URL, "/"), projectID, weekStart)
}

// WeeklyIndexURLWithBase 使用指定站点前缀拼接周报链接。
func WeeklyIndexURLWithBase(base string, projectID uint, weekStart string) string {
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = strings.TrimRight(zentaoCfg.URL, "/")
	}
	if base == "" || projectID == 0 {
		return ""
	}
	dateKey := compactDateYYYYMMDD(weekStart)
	q := url.Values{}
	q.Set("m", "weekly")
	q.Set("f", "index")
	q.Set("projectID", strconv.FormatUint(uint64(projectID), 10))
	q.Set("date", dateKey)
	q.Set("from", "projectweekly")
	return base + indexPath + "?" + q.Encode()
}

func compactDateYYYYMMDD(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if len(s) == 8 {
		if _, err := time.ParseInLocation("20060102", s, time.Local); err == nil {
			return s
		}
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006/01/02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t.Format("20060102")
		}
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t.Format("20060102")
	}
	return ""
}
