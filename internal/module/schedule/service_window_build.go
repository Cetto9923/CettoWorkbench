// =============================================================================
// 文件: internal/module/schedule/service_window_build.go
// 模块: 排期工作台
// 类型: action
// 职责: 版本窗口 Create/Update 请求装配与字段规范化。
// 依赖: internal/model
// =============================================================================

package schedule

import (
	"fmt"
	"strings"
	"time"

	"workbench/internal/model"
)

func buildVersionWindowFromCreateReq(req CreateReq) (*model.VersionWindow, error) {
	releaseDate, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.ReleaseDate), time.Local)
	if err != nil {
		return nil, fmt.Errorf("invalid release date")
	}

	planTestDone, err := parseOptionalWindowDate(req.PlanTestDone)
	if err != nil {
		return nil, fmt.Errorf("invalid plan test done date")
	}
	testDone, err := parseOptionalWindowDate(req.TestDone)
	if err != nil {
		return nil, fmt.Errorf("invalid test done date")
	}
	acceptDone, err := parseOptionalWindowDate(req.AcceptDone)
	if err != nil {
		return nil, fmt.Errorf("invalid accept done date")
	}

	window := &model.VersionWindow{
		Name:         strings.TrimSpace(req.Name),
		ReleaseDate:  releaseDate,
		WindowType:   normalizeWindowReleaseType(req.WindowType),
		PlanTestDone: planTestDone,
		TestDone:     testDone,
		AcceptDone:   acceptDone,
		TeamgroupID:  req.TeamgroupID,
		Status:       "planning",
	}
	if req.GroupSize > 0 {
		window.GroupSize = uint(req.GroupSize)
	} else {
		window.GroupSize = 1
	}

	startDate := strings.TrimSpace(req.StartDate)
	if startDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", startDate, time.Local)
		if err != nil {
			return nil, fmt.Errorf("invalid start date")
		}
		window.StartDate = &parsed
	}
	return window, nil
}

func applyUpdateReqToVersionWindow(window *model.VersionWindow, req UpdateReq) error {
	if window == nil {
		return fmt.Errorf("version window is nil")
	}
	releaseDate, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.ReleaseDate), time.Local)
	if err != nil {
		return fmt.Errorf("invalid release date")
	}
	planTestDone, err := parseOptionalWindowDate(req.PlanTestDone)
	if err != nil {
		return fmt.Errorf("invalid plan test done date")
	}
	testDone, err := parseOptionalWindowDate(req.TestDone)
	if err != nil {
		return fmt.Errorf("invalid test done date")
	}
	acceptDone, err := parseOptionalWindowDate(req.AcceptDone)
	if err != nil {
		return fmt.Errorf("invalid accept done date")
	}
	window.Name = strings.TrimSpace(req.Name)
	window.ReleaseDate = releaseDate
	window.WindowType = normalizeWindowReleaseType(req.WindowType)
	window.PlanTestDone = planTestDone
	window.TestDone = testDone
	window.AcceptDone = acceptDone
	window.TeamgroupID = req.TeamgroupID
	if req.GroupSize > 0 {
		window.GroupSize = uint(req.GroupSize)
	} else {
		window.GroupSize = 1
	}
	startDate := strings.TrimSpace(req.StartDate)
	if startDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", startDate, time.Local)
		if err != nil {
			return fmt.Errorf("invalid start date")
		}
		window.StartDate = &parsed
	} else {
		window.StartDate = nil
	}
	return nil
}

func parseOptionalWindowDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil, err
	}
	d := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, parsed.Location())
	return &d, nil
}

func normalizeWindowReleaseType(value string) string {
	switch strings.TrimSpace(value) {
	case "fast", "urgent":
		return strings.TrimSpace(value)
	default:
		return "regular"
	}
}

func formatOptionalWindowDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02")
}
