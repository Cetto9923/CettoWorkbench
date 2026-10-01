package po

import (
	"context"
	"sort"
	"time"
)

type DashboardDueItem struct {
	ID       uint   `json:"id"`
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Owner    string `json:"owner"`
	Deadline string `json:"deadline"`
	Stage    string `json:"stage"`
	Overdue  bool   `json:"overdue"`
}
type TeamDashboard struct {
	Overdue     int                `json:"overdue"`
	Soon        int                `json:"soon"`
	Blocked     int                `json:"blocked"`
	Suspended   int                `json:"suspended"`
	Due         []DashboardDueItem `json:"due"`
	UpdatedAt   string             `json:"updatedAt"`
	ThroughDate string             `json:"throughDate"`
}

func (s *Service) TeamDashboard(ctx context.Context, ids []uint, accounts []string, _ uint) (*TeamDashboard, error) {
	out := &TeamDashboard{Due: []DashboardDueItem{}, UpdatedAt: time.Now().Format("2006-01-02 15:04")}
	if len(ids) == 0 {
		return out, nil
	}
	todayDate := dashboardDate(todayStr())
	soon, err := s.schedule.WorkdayOffset(ctx, todayDate, 3)
	if err != nil {
		return nil, err
	}
	out.ThroughDate = soon.Format("2006-01-02")
	var demands []struct {
		ID       uint
		Name     string
		Status   string
		Hang     string
		Deadline string
		BRA      string
		Blocked  bool
	}
	blockedSQL := `(status='refuse' OR (` + dateSetBeforeTodaySQL("developFinish") + ` AND (COALESCE(managerReviewers,'')<>'' OR EXISTS(SELECT 1 FROM zt_demandmanagerreview mr WHERE mr.demand=zt_demand.id)) AND COALESCE(isManagerReview,'') NOT IN ('pass','passed')))`
	query := s.repo.teamDemandScope(ctx, ids).Select(`id,name,status,hang,COALESCE(deadline,'') AS deadline,BRA,` + blockedSQL + ` AS blocked`)
	if err = query.Scan(&demands).Error; err != nil {
		return nil, err
	}
	for _, d := range demands {
		if d.Blocked {
			out.Blocked++
		}
		if d.Hang == "1" {
			out.Suspended++
		}
		date := dashboardDate(d.Deadline)
		if date.IsZero() || isTerminal(d.Status) {
			continue
		}
		late := date.Before(todayDate)
		if late {
			out.Overdue++
		} else if !date.After(soon) {
			out.Soon++
		} else {
			continue
		}
		out.Due = append(out.Due, DashboardDueItem{ID: d.ID, Kind: "demand", Title: d.Name, Owner: d.BRA, Deadline: date.Format("2006-01-02"), Stage: d.Status, Overdue: late})
	}
	if len(accounts) > 0 {
		var tasks []struct {
			ID         uint
			Name       string
			AssignedTo string
			Deadline   string
			Status     string
		}
		if err = s.repo.db.WithContext(ctx).Table("zt_task").Select("id,name,assignedTo,deadline,status").Where("deleted='0' AND assignedTo IN ? AND status NOT IN ('done','closed','cancel') AND deadline >= '2000-01-01' AND deadline <= ?", accounts, soon.Format("2006-01-02")).Scan(&tasks).Error; err != nil {
			return nil, err
		}
		for _, t := range tasks {
			date := dashboardDate(t.Deadline)
			late := date.Before(todayDate)
			if late {
				out.Overdue++
			} else {
				out.Soon++
			}
			out.Due = append(out.Due, DashboardDueItem{ID: t.ID, Kind: "task", Title: t.Name, Owner: t.AssignedTo, Deadline: date.Format("2006-01-02"), Stage: t.Status, Overdue: late})
		}
	}
	sort.Slice(out.Due, func(i, j int) bool {
		if out.Due[i].Deadline == out.Due[j].Deadline {
			return out.Due[i].ID < out.Due[j].ID
		}
		return out.Due[i].Deadline < out.Due[j].Deadline
	})
	if len(out.Due) > 10 {
		out.Due = out.Due[:10]
	}
	return out, nil
}

func dashboardDate(raw string) time.Time {
	if len(raw) < 10 || raw[:4] == "0000" {
		return time.Time{}
	}
	t, _ := time.ParseInLocation("2006-01-02", raw[:10], time.Local)
	if t.Year() < 2000 {
		return time.Time{}
	}
	return t
}
