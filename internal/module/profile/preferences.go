package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"workbench/internal/middleware"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// Preferences are presentation settings only; they never grant roles or scope.
var preferenceKeys = map[string]bool{
	"last_view": true, "demand_scope": true, "team_scope": true,
	"demand_hidden_cards": true, "team_hidden_cards": true, "pinned_pages": true,
}

func validatePreference(key string, value json.RawMessage) error {
	if !preferenceKeys[key] || !json.Valid(value) || len(value) > 8192 {
		return errorx.New("invalid_preference", "偏好键或内容无效")
	}
	if key == "last_view" {
		var view string
		if json.Unmarshal(value, &view) != nil || (view != "team" && view != "demand") {
			return errorx.New("invalid_preference", "无效的首页视角")
		}
	}
	if key == "pinned_pages" || strings.HasSuffix(key, "hidden_cards") {
		var list []string
		if json.Unmarshal(value, &list) != nil || len(list) > 8 {
			return errorx.New("invalid_preference", "最多保存 8 项")
		}
		for _, item := range list {
			if key == "team_hidden_cards" && item == "versions" {
				return errorx.New("invalid_preference", "版本交付进度不可隐藏")
			}
		}
	}
	return nil
}

func (r *Repo) ReadPreference(ctx context.Context, account, key string) (json.RawMessage, bool, error) {
	var rows []struct {
		Value string `gorm:"column:prefValue"`
	}
	err := r.db.WithContext(ctx).Table("zt_wb_profile_prefs").Select("prefValue").Where("account = ? AND prefKey = ?", account, key).Limit(1).Scan(&rows).Error
	if err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	if !json.Valid([]byte(rows[0].Value)) {
		return nil, false, fmt.Errorf("invalid stored preference")
	}
	return json.RawMessage(rows[0].Value), true, nil
}

func (r *Repo) WritePreference(ctx context.Context, account, key string, value json.RawMessage) error {
	return r.db.WithContext(ctx).Exec(`INSERT INTO zt_wb_profile_prefs (account, prefKey, prefValue) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE prefValue = VALUES(prefValue), updatedAt = CURRENT_TIMESTAMP`, account, key, string(value)).Error
}

func (s *Service) SetPreference(ctx context.Context, actor *model.User, key string, value json.RawMessage) error {
	if actor == nil || actor.Account == "" {
		return errorx.New("unauthorized", "请先登录")
	}
	if err := validatePreference(key, value); err != nil {
		return err
	}
	return s.repo.WritePreference(ctx, actor.Account, key, value)
}

func (h *Handler) GetPreference(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	if actor == nil || actor.Account == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	key := c.Query("key")
	if !preferenceKeys[key] {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的偏好键"})
		return
	}
	value, found, err := h.svc.repo.ReadPreference(c.Request.Context(), actor.Account, key)
	if err != nil {
		writeProfileErr(c, h.logger, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "key": key, "val": value, "found": found})
}

func (h *Handler) SetPreference(c *gin.Context) {
	var req struct {
		Key   string          `json:"key"`
		Value json.RawMessage `json:"val"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10000)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "偏好格式无效"})
		return
	}
	if err := h.svc.SetPreference(c.Request.Context(), middleware.CurrentUser(c), req.Key, req.Value); err != nil {
		writeProfileErr(c, h.logger, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
