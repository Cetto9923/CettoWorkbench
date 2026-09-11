// =============================================================================
// 文件: internal/module/profile/service.go
// 模块: 个人资料
// 类型: action
// 职责: 当前登录用户自助查看/更新资料、修改密码的业务逻辑。
// 依赖: internal/model
//       internal/pkg/encode
//       internal/pkg/errorx
//       internal/pkg/workbenchroles
// =============================================================================

package profile

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/encode"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/workbenchroles"
)

// Service 个人资料业务逻辑。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// Get 返回当前登录用户资料、自选视图与敏捷小组。
func (s *Service) Get(ctx context.Context, actor *model.User) (GetResp, error) {
	if actor == nil || actor.ID <= 0 {
		return GetResp{}, errorx.New("unauthorized", "请先登录")
	}
	if s.repo == nil {
		return GetResp{}, errors.New("repo is nil")
	}
	row, err := s.repo.FindByID(ctx, actor.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResp{}, errorx.New("not_found", "用户不存在")
		}
		return GetResp{}, err
	}
	groups, gErr := s.repo.FindUserAgileGroups(ctx, actor.Account)
	if gErr != nil {
		return GetResp{}, gErr
	}

	allowed := s.selfSelectableRoles(ctx, actor, row.DeptID)
	preferred, err := s.repo.FindPreferredRoles(ctx, actor.Account)
	if err != nil {
		preferred = []string{}
	}
	preferred = filterPreferred(preferred, allowed)

	return GetResp{
		Account:        row.Account,
		DisplayName:    row.Realname,
		Email:          row.Email,
		Mobile:         row.Mobile,
		Gender:         row.Gender,
		DeptID:         row.DeptID,
		DeptName:       row.DeptName,
		AllowedRoles:   allowed,
		PreferredRoles: preferred,
		AgileGroups:    groups,
		MainTeamID:     row.MainTeam,
	}, nil
}

// Update 更新邮箱 / 性别 / 自选视图 / 默认小组。
func (s *Service) Update(ctx context.Context, actor *model.User, req UpdateReq) (UpdateResp, error) {
	if actor == nil || actor.ID <= 0 {
		return UpdateResp{}, errorx.New("unauthorized", "请先登录")
	}
	if s.repo == nil {
		return UpdateResp{}, errors.New("repo is nil")
	}
	// 先确认用户存在
	row, err := s.repo.FindByID(ctx, actor.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UpdateResp{}, errorx.New("not_found", "用户不存在")
		}
		return UpdateResp{}, err
	}

	// 敏捷小组白名单校验
	if req.MainTeamID != 0 {
		groups, gErr := s.repo.FindUserAgileGroups(ctx, actor.Account)
		if gErr != nil {
			return UpdateResp{}, gErr
		}
		if !containsAgileGroupID(groups, req.MainTeamID) {
			return UpdateResp{}, errorx.New("invalid_team", "默认小组必须来自本人已加入的敏捷小组")
		}
	}

	// 工作台角色白名单校验
	allowed := s.selfSelectableRoles(ctx, actor, row.DeptID)
	for _, key := range req.PreferredRoles {
		if !containsRole(allowed, key) {
			return UpdateResp{}, errorx.New("forbidden_role", "无权勾选工作台角色："+key)
		}
	}
	preferred := filterPreferred(req.PreferredRoles, allowed)

	// 更新姓名（如果传了且变更）
	if req.DisplayName != "" && req.DisplayName != row.Realname {
		if err := s.repo.UpdateDisplayName(ctx, actor.ID, req.DisplayName); err != nil {
			return UpdateResp{}, err
		}
	}
	// 更新联系方式（邮箱/性别）
	if err := s.repo.UpdateSelfContact(ctx, actor.ID, req.Email, req.Gender, req.ApplyGenderSkip()); err != nil {
		return UpdateResp{}, err
	}
	// 更新手机（如果传了且变更）
	if req.Mobile != "" && req.Mobile != row.Mobile {
		if err := s.repo.UpdateMobile(ctx, actor.ID, req.Mobile); err != nil {
			return UpdateResp{}, err
		}
	}
	// 更新默认小组
	if err := s.repo.UpdateMainTeam(ctx, actor.ID, req.MainTeamID); err != nil {
		return UpdateResp{}, err
	}
	// 更新自选角色偏好
	if err := s.repo.UpsertPreferredRoles(ctx, actor.Account, preferred); err != nil {
		return UpdateResp{}, err
	}

	return UpdateResp{
		ID:             actor.ID,
		PreferredRoles: preferred,
		MainTeamID:     req.MainTeamID,
	}, nil
}

// ChangePassword 校验旧密码后更新本人密码。
func (s *Service) ChangePassword(ctx context.Context, actor *model.User, req ChangePasswordReq) error {
	if actor == nil || actor.ID <= 0 {
		return errorx.New("unauthorized", "请先登录")
	}
	if s.repo == nil {
		return errors.New("repo is nil")
	}
	hash, err := s.repo.FindPasswordHash(ctx, actor.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorx.New("not_found", "用户不存在")
		}
		return err
	}
	if hash != encode.MD5(req.OldPassword) {
		return errorx.New("bad_password", "当前密码不正确")
	}
	return s.repo.UpdatePassword(ctx, actor.ID, encode.MD5(req.NewPassword))
}

func (s *Service) selfSelectableRoles(ctx context.Context, actor *model.User, deptID uint64) []RoleOption {
	keys := workbenchroles.DefaultAllowedFor(actor.Account, actor.IsSuperAdmin, deptID)
	labels := workbenchroles.RoleMap()
	out := make([]RoleOption, 0, len(keys))
	for _, key := range keys {
		if key == workbenchroles.RoleLead || key == workbenchroles.RolePMO {
			continue
		}
		label := key
		if def, ok := labels[key]; ok && def.Label != "" {
			label = def.Label
		}
		out = append(out, RoleOption{Key: key, Label: label})
	}
	if out == nil {
		out = []RoleOption{}
	}
	return out
}

func filterPreferred(preferred []string, allowed []RoleOption) []string {
	out := make([]string, 0, len(preferred))
	seen := map[string]bool{}
	for _, key := range preferred {
		if seen[key] || !containsRole(allowed, key) {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}

func containsRole(allowed []RoleOption, key string) bool {
	for _, opt := range allowed {
		if opt.Key == key {
			return true
		}
	}
	return false
}

func containsAgileGroupID(groups []AgileGroupOption, id uint64) bool {
	for _, g := range groups {
		if g.ID == id {
			return true
		}
	}
	return false
}
