package schedule

import "testing"

func TestExtractScheduleSearchID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		keyword string
		want    string
	}{
		{name: "plain id", keyword: "37436", want: "37436"},
		{name: "req display id", keyword: "REQ-37436", want: "37436"},
		{name: "req display id no hyphen", keyword: "REQ37436", want: "37436"},
		{name: "rd display id", keyword: "RD-37448", want: "37448"},
		{name: "sub display id", keyword: "SUB-123", want: "123"},
		{name: "title should not match", keyword: "US37436-0000-反洗钱系统", want: ""},
		{name: "invalid suffix", keyword: "REQ-ABC", want: ""},
		{name: "plain text", keyword: "反洗钱系统", want: ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := extractScheduleSearchID(tc.keyword); got != tc.want {
				t.Fatalf("extractScheduleSearchID(%q) = %q, want %q", tc.keyword, got, tc.want)
			}
		})
	}
}
