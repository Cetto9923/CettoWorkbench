package po

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"workbench/internal/middleware"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

const issueRiskTeamAccountsContextKey = "issueRiskTeamAccounts"

func (h *Handler) issueRiskPageAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		if currentUserHasPerm(c, perm.PoBoardDemandList) {
			c.Next()
			return
		}
		if _, ok := h.resolveIssueRiskTeamScope(c, middleware.CurrentUser(c)); !ok {
			return
		}
		c.Next()
	}
}

func (h *Handler) issueRiskItemsAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		if currentUserHasPerm(c, perm.PoBoardDemandList) {
			c.Next()
			return
		}
		if _, ok := h.resolveIssueRiskTeamScope(c, middleware.CurrentUser(c)); !ok {
			return
		}
		c.Next()
	}
}

func (h *Handler) resolveIssueRiskTeamScope(c *gin.Context, actor *model.User) ([]string, bool) {
	if h.teamViewAccess == nil || h.teamScopeAccounts == nil {
		h.denyHomeAccess(c)
		return nil, false
	}
	allowed, err := h.teamViewAccess(c.Request.Context(), actor)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return nil, false
	}
	scope, rawID := c.Query("scope"), c.Query("scopeId")
	if c.Query("team") != "" {
		scope, rawID = "dept", c.Query("team")
	}
	if c.Query("teamgroupId") != "" {
		scope, rawID = "team", c.Query("teamgroupId")
	}
	if !allowed || strings.TrimSpace(scope) == "" {
		h.denyHomeAccess(c)
		return nil, false
	}
	scopeID, err := parseTeamScopeID(rawID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "团队范围参数无效"})
		return nil, false
	}
	accounts, err := h.teamScopeAccounts(c.Request.Context(), actor, scope, scopeID)
	if err != nil {
		writeTeamScopeError(c, h.logger, err)
		return nil, false
	}
	c.Set(issueRiskTeamAccountsContextKey, accounts)
	return accounts, true
}

func parseTeamScopeID(raw string) (uint, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(value), nil
}

func writeTeamScopeError(c *gin.Context, logger *zap.Logger, err error) {
	if bizErr, ok := errorx.IsBizError(err); ok {
		status := http.StatusBadRequest
		if bizErr.Code == errorx.ErrCodeForbidden {
			status = http.StatusForbidden
		}
		c.AbortWithStatusJSON(status, gin.H{"message": bizErr.Msg})
		return
	}
	if logger != nil {
		logger.Error("resolve issue risk team scope", zap.Error(err))
	}
	c.AbortWithStatus(http.StatusServiceUnavailable)
}

func (h *Handler) IssueRisk(c *gin.Context) {
	render.Page(c, http.StatusOK, "po/issue-risk", gin.H{
		"Title":           "问题风险",
		"PageTitle":       "问题风险",
		"PageDescription": "统一跟踪和治理跨需求、跨团队的问题与风险项",
	})
}
func (h *Handler) IssueRiskItems(c *gin.Context) {
	var req IssueRiskListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	if es := req.Validate(); len(es) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "参数校验失败", "errors": es})
		return
	}
	if req.Scope != "" {
		if scopedAccounts, ok := c.Get(issueRiskTeamAccountsContextKey); ok {
			req.teamAccounts, _ = scopedAccounts.([]string)
		} else {
			if h.teamScopeAccounts == nil {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			accounts, err := h.teamScopeAccounts(c.Request.Context(), middleware.CurrentUser(c), req.Scope, req.ScopeID)
			if err != nil {
				writeTeamScopeError(c, h.logger, err)
				return
			}
			req.teamAccounts = accounts
		}
	}
	resp, err := h.svc.IssueRiskList(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("po issue risk list", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取问题风险失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "items": resp.Items, "total": resp.Total, "kindCounts": resp.KindCounts, "loopCounts": resp.LoopCounts, "overdueCount": resp.OverdueCount, "projects": resp.Projects, "page": req.Page, "pageSize": req.PageSize})
}
