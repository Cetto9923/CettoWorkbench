// =============================================================================
// 文件: internal/model/user_trimmed_test.go
// 模块: 数据模型
// 类型: test
// 职责: 锁定取账号的统一口径，含未登录与空格的边界。
// =============================================================================
package model

import "testing"

func TestUserTrimmedAccount(t *testing.T) {
	cases := []struct {
		name string
		user *User
		want string
	}{
		{"未登录", nil, ""},
		{"空账号", &User{}, ""},
		{"去除空格", &User{Account: "  alice  "}, "alice"},
		{"正常账号", &User{Account: "003030"}, "003030"},
	}
	for _, tc := range cases {
		if got := tc.user.TrimmedAccount(); got != tc.want {
			t.Errorf("%s：期望 %q 实际 %q", tc.name, tc.want, got)
		}
	}
}
