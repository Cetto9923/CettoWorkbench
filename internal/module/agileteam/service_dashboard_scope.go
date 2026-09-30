package agileteam

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"workbench/internal/middleware"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/workbenchroles"
)

type DashboardScope struct {
	ID     uint   `json:"id"`
	Parent uint   `json:"parent"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	DeptID uint   `json:"deptId"`
}

func (s *Service) dashboardGlobal(ctx context.Context, actor *model.User) (bool, error) {
	if _, err := requireActorAccount(actor); err != nil {
		return false, err
	}
	if actor.IsSuperAdmin {
		return true, nil
	}
	var count int64
	err := s.repo.read().WithContext(ctx).Raw(`SELECT COUNT(*) FROM zt_gf_user_roles ur JOIN zt_roles r ON r.id=ur.roleId
WHERE ur.userId=? AND ur.deleted='0' AND r.deleted='0' AND r.isActive=1 AND r.code=?`, actor.ID, workbenchroles.RolePMO).Scan(&count).Error
	return count > 0, err
}

// DashboardScopes exposes department IDs separately from agile group IDs.
func (s *Service) DashboardScopes(ctx context.Context, actor *model.User) ([]DashboardScope, error) {
	global, err := s.dashboardGlobal(ctx, actor)
	if err != nil {
		return nil, err
	}
	var deptIDs []uint
	if !global {
		deptIDs, err = s.repo.ListDeptTreeIDs(ctx, actor.Account)
		if err != nil {
			return nil, err
		}
	}
	depts := []DashboardScope{}
	q := s.repo.read().WithContext(ctx).Table("zt_dept").Select("id,parent,name,'dept' AS kind")
	if !global {
		if len(deptIDs) == 0 {
			q = q.Where("1=0")
		} else {
			q = q.Where("id IN ?", deptIDs)
		}
	}
	if err = q.Order("grade,id").Scan(&depts).Error; err != nil {
		return nil, err
	}
	deptIDs = nil
	for _, d := range depts {
		deptIDs = append(deptIDs, d.ID)
	}
	managed, err := s.repo.ListManagedTeamgroupIDs(ctx, actor.Account)
	if err != nil {
		return nil, err
	}
	mapped, err := s.repo.ListMappedTeamgroupIDsByDepts(ctx, deptIDs)
	if err != nil {
		return nil, err
	}
	allowed := map[uint]bool{}
	for _, id := range append(managed, mapped...) {
		allowed[id] = true
	}
	var groups []DashboardScope
	q = s.repo.read().WithContext(ctx).Table("zt_teamgroup tg").Select(`tg.id,tg.parent,tg.name,'team' AS kind,COALESCE(m.deptId,pm.deptId,0) AS dept_id`).
		Joins(`LEFT JOIN zt_wb_agileteam_orgmap m ON m.teamgroupId=tg.id AND m.status='active'`).
		Joins(`LEFT JOIN zt_wb_agileteam_orgmap pm ON pm.teamgroupId=tg.parent AND pm.status='active' AND m.id IS NULL`).Where("tg.deleted='0'")
	if !global {
		ids := []uint{}
		for id := range allowed {
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			q = q.Where("1=0")
		} else {
			q = q.Where("tg.id IN ?", ids)
		}
	}
	if err = q.Order("tg.id").Scan(&groups).Error; err != nil {
		return nil, err
	}
	return append(depts, groups...), nil
}

func (s *Service) CanEnterDashboard(ctx context.Context, actor *model.User) (bool, error) {
	global, err := s.dashboardGlobal(ctx, actor)
	if err != nil || global {
		return global, err
	}
	return s.CanEnterLeadView(ctx, actor)
}

func (s *Service) DashboardGroupIDs(ctx context.Context, actor *model.User, scope string, id uint) ([]uint, error) {
	if scope != "dept" && scope != "team" {
		return nil, errorx.New("invalid_param", "无效的范围类型")
	}
	options, err := s.DashboardScopes(ctx, actor)
	if err != nil {
		return nil, err
	}
	selected := id == 0
	for _, o := range options {
		if o.Kind == scope && o.ID == id {
			selected = true
		}
	}
	if !selected {
		return nil, errorx.New("forbidden", "所选范围未授权")
	}
	depts := map[uint]bool{}
	if scope == "dept" {
		if id > 0 {
			depts[id] = true
		} else {
			for _, o := range options {
				if o.Kind == "dept" {
					depts[o.ID] = true
				}
			}
		}
		for changed := true; changed; {
			changed = false
			for _, o := range options {
				if o.Kind == "dept" && depts[o.Parent] && !depts[o.ID] {
					depts[o.ID] = true
					changed = true
				}
			}
		}
	}
	ids := []uint{}
	seen := map[uint]bool{}
	for _, o := range options {
		if o.Kind != "team" {
			continue
		}
		include := scope == "dept" && depts[o.DeptID] || scope == "team" && (id == 0 || o.ID == id || o.Parent == id)
		if include && !seen[o.ID] {
			ids = append(ids, o.ID)
			seen[o.ID] = true
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func (s *Service) DashboardAccounts(ctx context.Context, actor *model.User, scope string, id uint) ([]string, error) {
	ids, err := s.DashboardGroupIDs(ctx, actor, scope, id)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []string{}, nil
	}
	members, err := s.repo.ListMembersByGroupIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	accounts := []string{}
	for _, rows := range members {
		for _, m := range rows {
			a := strings.TrimSpace(m.Account)
			if a != "" && !seen[a] {
				seen[a] = true
				accounts = append(accounts, a)
			}
		}
	}
	sort.Strings(accounts)
	return accounts, nil
}

func (h *Handler) DashboardScopes(c *gin.Context) {
	items, err := h.svc.DashboardScopes(c.Request.Context(), middleware.CurrentUser(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}
