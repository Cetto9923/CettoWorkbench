package po

import "strings"

// DemandActionReq 是首页站内动作的统一请求体。
type DemandActionReq struct {
	Comment string `json:"comment"`
}

func (r *DemandActionReq) normalize() {
	r.Comment = strings.TrimSpace(r.Comment)
	if len(r.Comment) > 2000 {
		r.Comment = r.Comment[:2000]
	}
}
