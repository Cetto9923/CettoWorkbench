// =============================================================================
// 文件: internal/module/po/form_weekly.go
// 模块: PO 工作台
// 类型: action
// 职责: 项目周报聚合列表 / 详情 / 历史 的 Req/Resp（关注 + 全量）
// 依赖: internal/pkg/errorx
// =============================================================================

package po

import (
	"strconv"
	"strings"
)

// ProjectWeeklyListReq 项目周报列表查询。
// Scope=watched：我的关注；Scope=all：全量项目周报。
// TeamID / TeamIDs：承建团队 = 项目经理所在部门（zt_user.dept → zt_dept），
// 选中节点含自身 path 及子孙（多选 OR 并集）；空=不限。
type ProjectWeeklyListReq struct {
	Filter     string `form:"filter"`  // all|waiting|attention|risk|deviation|submitted|abnormal
	Keyword    string `form:"keyword"` // 项目编号/名称/项目经理
	Limit      int    `form:"limit"`
	Scope      string `form:"scope"`   // watched|all
	TeamID     uint   `form:"teamId"`  // 兼容单选
	TeamIDs    string `form:"teamIds"` // 多选：逗号分隔 "12,34"
	teamIDList []uint
}

// Validate 规范化筛选与分页上限，并解析 teamIds。
func (r *ProjectWeeklyListReq) Validate() []FieldError {
	r.Filter = strings.TrimSpace(strings.ToLower(r.Filter))
	if r.Filter == "" {
		r.Filter = "all"
	}
	switch r.Filter {
	case "all", "waiting", "attention", "risk", "deviation", "submitted", "abnormal":
	default:
		return []FieldError{{Field: "filter", Message: "不支持的筛选条件"}}
	}
	r.Scope = strings.TrimSpace(strings.ToLower(r.Scope))
	if r.Scope == "" {
		r.Scope = "mine"
	}
	switch r.Scope {
	case "mine", "participated", "watched", "all":
	default:
		return []FieldError{{Field: "scope", Message: "不支持的范围"}}
	}
	r.Keyword = strings.TrimSpace(r.Keyword)
	if r.Limit <= 0 {
		r.Limit = 200
	}
	if r.Limit > 500 {
		r.Limit = 500
	}
	r.teamIDList = parseProjectWeeklyTeamIDs(r.TeamIDs, r.TeamID)
	return nil
}

// TeamIDList 返回解析后的承建团队 ID（去重、去 0）。
func (r ProjectWeeklyListReq) TeamIDList() []uint {
	return r.teamIDList
}

// ProjectWeeklyStats 顶部统计（与筛选同源）。
type ProjectWeeklyStats struct {
	Watched   int `json:"watched"`
	Submitted int `json:"submitted"`
	Waiting   int `json:"waiting"`
	Abnormal  int `json:"abnormal"`
	Attention int `json:"attention"`
	Risk      int `json:"risk"`
	Deviation int `json:"deviation"`
}

// ProjectWeeklyListItem 一项目一行（最新一期周报视角）。
type ProjectWeeklyListItem struct {
	ProjectID             uint    `json:"projectId"`
	ProjectCode           string  `json:"projectCode"`
	ProjectName           string  `json:"projectName"`
	PMAccount             string  `json:"pmAccount"`
	PMName                string  `json:"pmName"`
	PMDeptID              uint    `json:"pmDeptId"`
	PMDeptName            string  `json:"pmDeptName"`
	ReportID              uint    `json:"reportId"`
	WeekStart             string  `json:"weekStart"`
	WeekEnd               string  `json:"weekEnd"`
	WeekSN                int     `json:"weekSN"`
	SubmitStatus          string  `json:"submitStatus"` // submitted|waiting|overdue
	OverallSituation      int     `json:"overallSituation"`
	OverallSituationLabel string  `json:"overallSituationLabel"`
	OverallSituationDesc  string  `json:"overallSituationDesc"`
	FinishedCount         int     `json:"finishedCount"`
	UnfinishedCount       int     `json:"unfinishedCount"`
	NextWeekCount         int     `json:"nextWeekCount"`
	OpenIssueCount        int     `json:"openIssueCount"`
	OpenRiskCount         int     `json:"openRiskCount"`
	PlanDeviationDays     int     `json:"planDeviationDays"`
	ReleaseRisk           string  `json:"releaseRisk"` // normal|delayRelease|hasRisk|earlyRelease|abnormalRelease|""
	ReleaseRiskLabel      string  `json:"releaseRiskLabel"`
	Staff                 int     `json:"staff"`
	WorkloadHours         float64 `json:"workloadHours"`
	HasAbnormal           bool    `json:"hasAbnormal"`
	Source                string  `json:"source"` // participated|watched|both
	IsParticipated        bool    `json:"isParticipated"`
	IsWatched             bool    `json:"isWatched"`
	URL                   string  `json:"url"`
}

// ProjectWeeklyListResp 列表 + 同源统计卡。
type ProjectWeeklyListResp struct {
	WeekStart  string                  `json:"weekStart"`
	WeekEnd    string                  `json:"weekEnd"`
	DeadlineAt string                  `json:"deadlineAt"`
	Scope      string                  `json:"scope"`
	TeamID     uint                    `json:"teamId"`
	Stats      ProjectWeeklyStats      `json:"stats"`
	Items      []ProjectWeeklyListItem `json:"items"`
	Total      int                     `json:"total"`
}

// ProjectWeeklyDetailResp 侧滑详情。
type ProjectWeeklyDetailResp struct {
	ProjectWeeklyListItem
	Warning string `json:"warning"`
}

// ProjectWeeklyHistoryItem 历史周报一行。
type ProjectWeeklyHistoryItem struct {
	ReportID              uint    `json:"reportId"`
	WeekStart             string  `json:"weekStart"`
	WeekEnd               string  `json:"weekEnd"`
	OverallSituation      int     `json:"overallSituation"`
	OverallSituationLabel string  `json:"overallSituationLabel"`
	OpenIssueCount        int     `json:"openIssueCount"`
	OpenRiskCount         int     `json:"openRiskCount"`
	PlanDeviationDays     int     `json:"planDeviationDays"`
	ReleaseRisk           string  `json:"releaseRisk"`
	ReleaseRiskLabel      string  `json:"releaseRiskLabel"`
	SubmitStatus          string  `json:"submitStatus"`
	Staff                 int     `json:"staff"`
	WorkloadHours         float64 `json:"workloadHours"`
}

// ProjectWeeklyItemReq 详情/历史查询（scope 与列表一致）。
type ProjectWeeklyItemReq struct {
	Scope string `form:"scope"` // mine|participated|watched|all
}

// Validate 规范化 scope。
func (r *ProjectWeeklyItemReq) Validate() []FieldError {
	r.Scope = strings.TrimSpace(strings.ToLower(r.Scope))
	if r.Scope == "" {
		r.Scope = "mine"
	}
	switch r.Scope {
	case "mine", "participated", "watched", "all":
		return nil
	default:
		return []FieldError{{Field: "scope", Message: "不支持的范围"}}
	}
}

func parseProjectWeeklyTeamIDs(teamIDsCSV string, legacyTeamID uint) []uint {
	seen := map[uint]struct{}{}
	out := make([]uint, 0, 4)
	add := func(id uint) {
		if id == 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, part := range strings.Split(teamIDsCSV, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			continue
		}
		add(uint(n))
	}
	add(legacyTeamID)
	return out
}
