package personlabel

import "testing"

func TestFormatSuffixDedupe(t *testing.T) {
	cases := []struct {
		account, realname, want string
	}{
		{"", "", ""},
		{"", "程统", ""},
		{"003030", "", "003030"},
		{"003030", "程统", "程统(003030)"},
		{"003030", "程统(003030)", "程统(003030)"},
		{"003030", "程统 (003030)", "程统 (003030)"},
		{"003030", "003030", "003030"},
		{"003030", "程统(003030)备用", "程统(003030)备用"},
		{"zhangsan", "张三", "张三(zhangsan)"},
		{" 003030 ", " 程统 ", "程统(003030)"},
	}
	for _, tc := range cases {
		if got := Format(tc.account, tc.realname); got != tc.want {
			t.Errorf("Format(%q, %q) = %q, want %q", tc.account, tc.realname, got, tc.want)
		}
	}
}
