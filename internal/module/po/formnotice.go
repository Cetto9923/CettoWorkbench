// =============================================================================
// 文件: internal/module/po/formnotice.go
// 模块: PO 工作台
// 类型: action
// 职责: 通知中心请求、列表项与响应类型及校验。
// 依赖: 无
// =============================================================================

package po

import "strings"

type NoticeListReq struct {
	QuickView  string `form:"quickView"`
	Category   string `form:"category"`
	ObjectType string `form:"objectType"`
	TimeRange  string `form:"timeRange"`
	ReadState  string `form:"readState"`
	NeedAction string `form:"needAction"`
	Keyword    string `form:"keyword"`
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
}

func (r *NoticeListReq) Validate() []FieldError {
	r.QuickView = noticeValue(r.QuickView, "all")
	if !isNoticeValue(r.QuickView, "all", "unread", "action", "abnormal", "today") {
		return []FieldError{{Field: "quickView", Message: "无效的快捷视图"}}
	}
	r.Category = noticeValue(r.Category, "all")
	if !isNoticeValue(r.Category, "all", "business", "approval", "reminder", "collaboration", "risk", "system") {
		return []FieldError{{Field: "category", Message: "无效的通知分类"}}
	}
	r.ObjectType = noticeValue(r.ObjectType, "all")
	r.TimeRange = noticeValue(r.TimeRange, "all")
	if !isNoticeValue(r.TimeRange, "all", "today", "3d", "7d", "30d") {
		return []FieldError{{Field: "timeRange", Message: "无效的时间范围"}}
	}
	r.ReadState = noticeValue(r.ReadState, "all")
	if !isNoticeValue(r.ReadState, "all", "unread", "read") {
		return []FieldError{{Field: "readState", Message: "无效的已读状态"}}
	}
	r.NeedAction = noticeValue(r.NeedAction, "all")
	if !isNoticeValue(r.NeedAction, "all", "required", "none") {
		return []FieldError{{Field: "needAction", Message: "无效的处理状态"}}
	}
	r.Keyword = strings.TrimSpace(r.Keyword)
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		r.PageSize = 20
	}
	return nil
}

func noticeValue(value, fallback string) string {
	if value = strings.TrimSpace(value); value == "" {
		return fallback
	}
	return value
}
func isNoticeValue(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

type NoticeItem struct {
	ID         int64  `json:"id"`
	ObjectType string `json:"objectType"`
	ObjectID   int64  `json:"objectId"`
	Title      string `json:"title"`   // 列表标题（Subject 清洗截断）
	Summary    string `json:"summary"` // 列表摘要（Data 截断摘要，与 Title 重复时为空）
	Content    string `json:"content"` // 完整内容（Data 清洗后完整文本）
	Subject    string `json:"subject"` // 兼容既有字段 (= Title)
	Data       string `json:"data"`    // 兼容既有字段 (= Summary)
	Actor      string `json:"actor"`
	Action     string `json:"action"`
	Category   string `json:"category"`
	NeedAction bool   `json:"needAction"`
	Anomaly    bool   `json:"anomaly"`
	Read       bool   `json:"read"`
	Date       string `json:"date"`
	URL        string `json:"url"`
}
type NoticeBucketResp struct {
	Items      []NoticeItem     `json:"items"`
	Total      int64            `json:"total"`
	Filtered   int64            `json:"filteredTotal"`
	Unread     int64            `json:"unread"`
	Action     int64            `json:"action"`
	Abnormal   int64            `json:"abnormal"`
	Today      int64            `json:"today"`
	Categories map[string]int64 `json:"categories"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
}
