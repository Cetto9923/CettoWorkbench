package zentao

import "testing"

func TestValidateDemandReviewResponse(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		fail       bool
	}{
		{"native", `{"success":true,"message":"saved"}`, 200, false},
		{"legacy", `"{\"message\":\"saved\"}"`, 200, false},
		{"permission", `{"error":"Access not allowed"}`, 403, true},
		{"controller fail", `{"result":"fail","message":"invalid"}`, 200, true},
		{"controller error", `{"status":"error","message":"invalid"}`, 200, true},
		{"false success", `{"success":false,"message":"invalid"}`, 200, true},
		{"empty", `{}`, 200, true},
		{"null", `null`, 200, true},
		{"html", `<html>error</html>`, 200, true},
		{"json then html result success", `{"result":"success","message":"ok"}<html>x</html>`, 200, false},
		{"json then html success true", "{\"success\":true,\"message\":\"saved\"}\n<div>trailer</div>", 200, false},
		{"json then html result fail", `{"result":"fail","message":"no"}<html>`, 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDemandReviewResponse([]byte(tc.body), tc.status)
			if (err != nil) != tc.fail {
				t.Fatalf("error=%v, want failure=%v", err, tc.fail)
			}
		})
	}
}
