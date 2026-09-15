// =============================================================================
// File: internal/module/schedule/form_normalization.go
// Module: schedule workbench
// Purpose: Normalization helpers and parsing functions for schedule module.
// =============================================================================

package schedule

import (
	"strconv"
	"strings"
)

// ParseCommaSeparatedUints parses a comma‑separated list of unsigned integers.
func ParseCommaSeparatedUints(raw string) []uint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]uint, 0, len(parts))
	seen := make(map[uint]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.ParseUint(part, 10, 64)
		if err != nil || value == 0 {
			continue
		}
		id := uint(value)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ParseCommaSeparatedStages parses a comma‑separated list of stage filter values.
func ParseCommaSeparatedStages(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	allowed := map[string]struct{}{
		StageFilterNoWindow:       {},
		StageFilterNoStory:        {},
		StageFilterNoTask:         {},
		StageFilterTaskUnassigned: {},
		StageFilterTaskAssigned:   {},
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := allowed[part]; !ok {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NormalizeStageFilterForTab retains only the stage values allowed for the given tab.
func NormalizeStageFilterForTab(raw, tab string) string {
	stages := ParseCommaSeparatedStages(raw)
	if len(stages) == 0 {
		return ""
	}
	allowed := allowedStageFilterValues(tab)
	out := make([]string, 0, len(stages))
	for _, stage := range stages {
		if allowed[stage] {
			out = append(out, stage)
		}
	}
	return strings.Join(out, ",")
}

// Normalize normalizes pagination and filter fields for ListBizDemandsReq.
func (r *ListBizDemandsReq) Normalize() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 10
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
	r.Filter = NormalizeDemandFilter(r.Filter)
	r.Keyword = strings.TrimSpace(r.Keyword)
	r.Groups = strings.TrimSpace(r.Groups)
	r.Products = strings.TrimSpace(r.Products)
	r.Stages = strings.TrimSpace(r.Stages)
	r.Windows = strings.TrimSpace(r.Windows)
	r.Pri = NormalizePriorityFilter(r.Pri)
	r.WindowType = NormalizeWindowTypeFilter(r.WindowType)
	r.DevOwner = strings.TrimSpace(r.DevOwner)
	r.TestOwner = strings.TrimSpace(r.TestOwner)
	r.AcceptOwner = strings.TrimSpace(r.AcceptOwner)
}

// Normalize normalizes pagination and filter fields for ListIndependentReq.
func (r *ListIndependentReq) Normalize() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 10
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
	r.Filter = NormalizeDemandFilter(r.Filter)
	r.Groups = strings.TrimSpace(r.Groups)
	r.Products = strings.TrimSpace(r.Products)
	r.Stages = strings.TrimSpace(r.Stages)
	r.Windows = strings.TrimSpace(r.Windows)
	r.Keyword = strings.TrimSpace(r.Keyword)
	r.Pri = NormalizePriorityFilter(r.Pri)
	r.WindowType = NormalizeWindowTypeFilter(r.WindowType)
	r.DevOwner = strings.TrimSpace(r.DevOwner)
	r.TestOwner = strings.TrimSpace(r.TestOwner)
}
