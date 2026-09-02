// =============================================================================
// 文件: internal/module/user/service.go
// 模块: 用户
// 类型: crud
// 职责: 实现用户列表查询和筛选业务逻辑。
// 依赖: internal/model/user.go
//       internal/module/user/repo.go
// =============================================================================

package user

import (
	"context"
	"errors"
	"strings"

	"workbench/internal/model"
	"workbench/internal/pkg/encode"
)

// Service 处理用户业务逻辑。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// List 获取用户列表并处理筛选逻辑。
func (s *Service) List(ctx context.Context, actor *model.User, req ListReq) (ListResp, error) {
	_ = actor
	req.Normalize()

	users, total, err := s.repo.FindAll(ctx, RepoFindAllReq{
		Account:     req.Account,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		IsActive:    req.IsActiveFilter(),
		Page:        req.Page,
		PageSize:    req.PageSize,
	})
	if err != nil {
		return ListResp{}, err
	}
	return ListResp{
		Items: users,
		Total: total,
	}, nil
}

// GetByID 获取用户详情。
func (s *Service) GetByID(ctx context.Context, actor *model.User, id int64) (*model.User, error) {
	_ = actor
	return s.repo.FindByID(ctx, id)
}

// Create 创建用户并分配角色。
func (s *Service) Create(ctx context.Context, actor *model.User, req CreateReq) (CreateResp, error) {
	exists, err := s.repo.ExistsByAccount(ctx, req.Account, 0)
	if err != nil {
		return CreateResp{}, err
	}
	if exists {
		return CreateResp{}, errors.New("用户名已存在")
	}

	m := &model.User{
		Account:      req.Account,
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		Gender:       req.Gender,
		PasswordHash: encode.MD5(req.Password),
		DeptID:       req.DeptID,
	}
	m.SetActive(req.IsActive)
	if err := s.repo.Create(ctx, m); err != nil {
		return CreateResp{}, err
	}
	if err := s.repo.ReplaceUserRoles(ctx, m.ID, req.RoleIDs); err != nil {
		return CreateResp{}, err
	}
	return CreateResp{ID: m.ID}, nil
}

// Update 更新用户并分配角色。
func (s *Service) Update(ctx context.Context, actor *model.User, req UpdateReq) (UpdateResp, error) {
	user, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return UpdateResp{}, err
	}

	user.Email = req.Email
	user.DisplayName = req.DisplayName
	user.Gender = req.Gender
	user.SetActive(req.IsActive)
	user.DeptID = req.DeptID
	if err := s.repo.Update(ctx, user); err != nil {
		return UpdateResp{}, err
	}
	if err := s.repo.ReplaceUserRoles(ctx, user.ID, req.RoleIDs); err != nil {
		return UpdateResp{}, err
	}
	return UpdateResp{ID: user.ID}, nil
}

// Delete 删除用户。
func (s *Service) Delete(ctx context.Context, actor *model.User, req DeleteReq) error {
	_ = actor
	return s.repo.Delete(ctx, req.ID)
}

// BatchCreate 批量创建用户并分配角色。
func (s *Service) BatchCreate(ctx context.Context, actor *model.User, req BatchCreateReq) (BatchCreateResp, error) {
	if len(req.Users) == 0 {
		return BatchCreateResp{}, errors.New("请至少填写一条用户数据")
	}

	users := make([]*model.User, 0, len(req.Users))
	roleIDsList := make([][]int64, 0, len(req.Users))
	seen := make(map[string]struct{}, len(req.Users))

	for _, item := range req.Users {
		accountKey := strings.ToLower(strings.TrimSpace(item.Account))
		if _, ok := seen[accountKey]; ok {
			return BatchCreateResp{}, errors.New("批量数据中存在重复用户名: " + item.Account)
		}
		seen[accountKey] = struct{}{}

		exists, err := s.repo.ExistsByAccount(ctx, item.Account, 0)
		if err != nil {
			return BatchCreateResp{}, err
		}
		if exists {
			return BatchCreateResp{}, errors.New("用户名已存在: " + item.Account)
		}

		user := &model.User{
			Account:      item.Account,
			Email:        item.Email,
			DisplayName:  item.DisplayName,
			Gender:       item.Gender,
			PasswordHash: encode.MD5(item.Password),
			DeptID:       item.DeptID,
		}
		user.SetActive(item.IsActive)
		users = append(users, user)
		roleIDsList = append(roleIDsList, item.RoleIDs)
	}

	if err := s.repo.BatchCreate(ctx, users, roleIDsList); err != nil {
		return BatchCreateResp{}, err
	}
	return BatchCreateResp{Count: len(users)}, nil
}

// ToggleStatus 切换用户启用状态。
func (s *Service) ToggleStatus(ctx context.Context, actor *model.User, req ToggleStatusReq) error {
	target, err := s.GetByID(ctx, actor, req.ID)
	if err != nil {
		return err
	}

	if actor != nil && actor.ID == req.ID {
		return errors.New("不能对自己执行启用/禁用操作")
	}
	if target.IsSuperAdmin {
		return errors.New("不能对超级管理员执行启用/禁用操作")
	}

	return s.repo.UpdateStatus(ctx, uint64(req.ID), !target.IsActive)
}

// ResetPassword 仅校验业务规则（目标用户是否存在）。
// 字段格式校验（密码强度、两次一致）由 Handler 层负责，此处不重复执行。
func (s *Service) ResetPassword(ctx context.Context, actor *model.User, req ResetPasswordReq) error {
	_ = actor
	if req.NewPassword != req.ConfirmPassword {
		return errors.New("两次输入的密码不一致")
	}

	_, err := s.GetByID(ctx, actor, int64(req.ID))
	if err != nil {
		return err
	}

	return s.repo.UpdatePassword(ctx, req.ID, encode.MD5(req.NewPassword))
}

// GetRoles 查询可分配角色列表。
func (s *Service) GetRoles(ctx context.Context, actor *model.User) ([]model.Role, error) {
	_ = actor
	return s.repo.ListRoles(ctx)
}

// GetUserRoleIDs 查询用户已分配角色 ID。
func (s *Service) GetUserRoleIDs(ctx context.Context, actor *model.User, userID int64) ([]int64, error) {
	_ = actor
	return s.repo.ListUserRoleIDs(ctx, userID)
}

// GetDepts 查询可选部门列表。
func (s *Service) GetDepts(ctx context.Context, actor *model.User) ([]model.Dept, error) {
	_ = actor
	return s.repo.ListDepts(ctx)
}

// Export 查询用户导出数据。
func (s *Service) Export(ctx context.Context, actor *model.User, req ExportReq) ([]model.User, error) {
	_ = actor
	_ = req
	return s.repo.FindAllForExport(ctx)
}
