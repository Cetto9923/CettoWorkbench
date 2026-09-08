// =============================================================================
// 文件: internal/module/profile/service.go
// 模块: 个人资料
// 类型: action
// 职责: 当前登录用户自助查看/更新资料、修改密码的业务逻辑。
// 依赖: internal/model
//       internal/pkg/encode
//       internal/pkg/errorx
// =============================================================================

package profile

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/encode"
	"workbench/internal/pkg/errorx"
)

// Service 个人资料业务逻辑。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// Get 返回当前登录用户资料。
func (s *Service) Get(ctx context.Context, actor *model.User) (GetResp, error) {
	if actor == nil || actor.ID <= 0 {
		return GetResp{}, errorx.New("unauthorized", "请先登录")
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
	return GetResp{
		Account:     row.Account,
		DisplayName: row.Realname,
		Email:       row.Email,
		Mobile:      row.Mobile,
		Gender:      row.Gender,
		DeptID:      row.DeptID,
		DeptName:    row.DeptName,
		AgileGroups: groups,
		MainTeamID:  row.MainTeam,
	}, nil
}

// Update 更新姓名 / 邮箱 / 手机 / 性别 / 默认小组。
//
// 性别为 "" 时（前端 radio 选了"未设置"），跳过 gender 字段写入；
// MainTeamID 不在 AgileGroups 中时拒绝（来自本人已加入的小组白名单）。
func (s *Service) Update(ctx context.Context, actor *model.User, req UpdateReq) error {
	if actor == nil || actor.ID <= 0 {
		return errorx.New("unauthorized", "请先登录")
	}
	// 先确认用户存在
	if _, err := s.repo.FindByID(ctx, actor.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorx.New("not_found", "用户不存在")
		}
		return err
	}
	// mainTeam 必须在本人已加入的小组内
	if req.MainTeamID != 0 {
		groups, gErr := s.repo.FindUserAgileGroups(ctx, actor.Account)
		if gErr != nil {
			return gErr
		}
		if !containsAgileGroupID(groups, req.MainTeamID) {
			return errorx.New("invalid_team", "默认小组必须来自本人已加入的敏捷小组")
		}
	}
	// 字段独立更新，便于在错误时不影响其它字段。
	if err := s.repo.UpdateDisplayName(ctx, actor.ID, req.DisplayName); err != nil {
		return err
	}
	if err := s.repo.UpdateSelfContact(ctx, actor.ID, req.Email, req.Gender, req.ApplyGenderSkip()); err != nil {
		return err
	}
	if req.Mobile != "" {
		if err := s.repo.UpdateMobile(ctx, actor.ID, req.Mobile); err != nil {
			return err
		}
	}
	if err := s.repo.UpdateMainTeam(ctx, actor.ID, req.MainTeamID); err != nil {
		return err
	}
	return nil
}

// ChangePassword 校验旧密码后更新本人密码。
//
// 与禅道 max5 兼容：旧密码以 MD5 hex 形式存储在 zt_user.password；
// 验证 + 重写都走同一种哈希，确保改密后仍能用旧密码登录。
func (s *Service) ChangePassword(ctx context.Context, actor *model.User, req ChangePasswordReq) error {
	if actor == nil || actor.ID <= 0 {
		return errorx.New("unauthorized", "请先登录")
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

func containsAgileGroupID(groups []AgileGroupOption, id uint64) bool {
	for _, g := range groups {
		if g.ID == id {
			return true
		}
	}
	return false
}
