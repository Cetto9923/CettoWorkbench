package po

import "strings"

// DemandActionReq 是首页站内动作的统一请求体。
type DemandActionReq struct {
	Comment  string   `json:"comment"`
	UrgeType string   `json:"urgeType"`
	Channels []string `json:"channels"`
}

// UrgeHomeDemandReq is intentionally limited to the homepage acceptance flow.
// Recipients are always resolved by the service; callers cannot target accounts.
type UrgeHomeDemandReq struct {
	UrgeType string   `json:"urgeType"`
	Comment  string   `json:"comment"`
	Channels []string `json:"channels"`
}

func (r *UrgeHomeDemandReq) normalize() {
	r.UrgeType = strings.TrimSpace(r.UrgeType)
	if r.UrgeType == "" || r.UrgeType == "remind_accept" {
		r.UrgeType = "accept"
	}
	r.Comment = strings.TrimSpace(r.Comment)
	if len(r.Comment) > 2000 {
		r.Comment = r.Comment[:2000]
	}
	if len(r.Channels) == 0 {
		r.Channels = []string{"inapp"}
	}
}

func (r *DemandActionReq) normalize() {
	r.Comment = strings.TrimSpace(r.Comment)
	if len(r.Comment) > 2000 {
		r.Comment = r.Comment[:2000]
	}
}
