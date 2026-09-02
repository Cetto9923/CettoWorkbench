package user

import (
	"strings"
	"time"
)

func genderLabel(gender string) string {
	switch strings.ToLower(strings.TrimSpace(gender)) {
	case "f":
		return "女"
	case "m":
		return "男"
	default:
		return "-"
	}
}

func activeLabel(active bool) string {
	if active {
		return "启用"
	}
	return "禁用"
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}
