// =============================================================================
// 文件: internal/module/user/service.go
// 模块: 用户
// 类型: crud
// 职责: 用户 CRUD；以及通过禅道 GET /user 获取可见范围（view.sprints 等）供其它模块过滤。
// 依赖: internal/model/user.go
//       internal/module/user/repo.go
//       internal/pkg/encode
//       internal/pkg/zentao
// =============================================================================

package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"workbench/internal/model"
	"workbench/internal/pkg/encode"
	"workbench/internal/pkg/zentao"
)

// zentaoViewTTLDefault 可见范围短缓存时长（跨机 HTTP，避免每次业务查询都打禅道）。
const zentaoViewTTLDefault = 3 * time.Minute

// zentaoAPI 禅道 REST 调用面（*zentao.Client 满足）。
type zentaoAPI interface {
	Do(ctx context.Context, method, path string, body any, out any) error
}

// ZentaoView 对齐禅道 app->user->view（含登录时 merge 的公开执行等）。
type ZentaoView struct {
	Admin    bool
	Sprints  []uint
	Products []uint
	Projects []uint
	Programs []uint
}

type zentaoViewCacheEntry struct {
	view ZentaoView
	at   time.Time
}

type zentaoUserProfileResp struct {
	Profile struct {
		Admin bool `json:"admin"`
		View  struct {
			Sprints  string `json:"sprints"`
			Products string `json:"products"`
			Projects string `json:"projects"`
			Programs string `json:"programs"`
		} `json:"view"`
	} `json:"profile"`
}

// Service 处理用户业务逻辑。
type Service struct {
	repo  *Repo
	ztAPI zentaoAPI

	zentaoViewMu    sync.Mutex
	zentaoViewCache map[string]zentaoViewCacheEntry
}

// NewService 创建 Service。ztAPI 可为 nil（仅本地用户 CRUD 时）；调用 ZentaoView 前须配置。
func NewService(repo *Repo, ztAPI zentaoAPI) *Service {
	return &Service{repo: repo, ztAPI: ztAPI}
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

// AccountDisplayMap 返回全部用户 account →「姓名(工号)」映射，供其它模块解析责任人展示名。
func (s *Service) AccountDisplayMap(ctx context.Context, actor *model.User) (map[string]string, error) {
	_ = actor
	return s.repo.FindAccountDisplayMap(ctx)
}

// ListInsideUsers 返回人员选择控件用的内部用户列表（按当前登录用户部门亲和度排序）。
func (s *Service) ListInsideUsers(ctx context.Context, actor *model.User) ([]InsideUserOption, error) {
	users, err := s.repo.FindInsideUsers(ctx, actorDeptID(actor))
	if err != nil {
		return nil, err
	}
	if users == nil {
		return []InsideUserOption{}, nil
	}
	return users, nil
}

func actorDeptID(actor *model.User) uint64 {
	if actor == nil {
		return 0
	}
	return actor.DeptID
}

// ZentaoView 调用 GET /user，返回当前账号可见范围；按 account 短缓存。
// account 优先取 ctx（RequireLogin 注入），否则回退 actor.Account。
func (s *Service) ZentaoView(ctx context.Context, actor *model.User) (ZentaoView, error) {
	if s == nil {
		return ZentaoView{}, fmt.Errorf("user service is nil")
	}
	account := zentao.AccountFrom(ctx)
	if account == "" && actor != nil {
		account = strings.TrimSpace(actor.Account)
	}
	if account == "" {
		return ZentaoView{}, fmt.Errorf("未登录或禅道账号为空")
	}
	ctx = zentao.WithAccount(ctx, account)

	if view, ok := s.getCachedZentaoView(account); ok {
		return view, nil
	}

	if s.ztAPI == nil {
		return ZentaoView{}, fmt.Errorf("禅道 API 未配置")
	}

	var resp zentaoUserProfileResp
	if err := s.ztAPI.Do(ctx, "GET", "/user", nil, &resp); err != nil {
		return ZentaoView{}, err
	}

	view := ZentaoView{
		Admin:    resp.Profile.Admin,
		Sprints:  parseCSVUintIDs(resp.Profile.View.Sprints),
		Products: parseCSVUintIDs(resp.Profile.View.Products),
		Projects: parseCSVUintIDs(resp.Profile.View.Projects),
		Programs: parseCSVUintIDs(resp.Profile.View.Programs),
	}
	s.putCachedZentaoView(account, view)
	return view, nil
}

// VisibleSprintIDs 返回当前用户可见执行 ID（对齐 view.sprints），供其它模块 IN 过滤。
func (s *Service) VisibleSprintIDs(ctx context.Context, actor *model.User) ([]uint, error) {
	view, err := s.ZentaoView(ctx, actor)
	if err != nil {
		return nil, err
	}
	out := make([]uint, len(view.Sprints))
	copy(out, view.Sprints)
	return out, nil
}

func (s *Service) getCachedZentaoView(account string) (ZentaoView, bool) {
	ttl := zentaoViewTTLDefault

	s.zentaoViewMu.Lock()
	defer s.zentaoViewMu.Unlock()
	if s.zentaoViewCache == nil {
		return ZentaoView{}, false
	}
	e, ok := s.zentaoViewCache[account]
	if !ok || e.at.IsZero() {
		return ZentaoView{}, false
	}
	if time.Since(e.at) >= ttl {
		delete(s.zentaoViewCache, account)
		return ZentaoView{}, false
	}
	return e.view, true
}

func (s *Service) putCachedZentaoView(account string, view ZentaoView) {
	s.zentaoViewMu.Lock()
	defer s.zentaoViewMu.Unlock()
	if s.zentaoViewCache == nil {
		s.zentaoViewCache = map[string]zentaoViewCacheEntry{}
	}
	s.zentaoViewCache[account] = zentaoViewCacheEntry{view: view, at: time.Now()}
}

// InvalidateZentaoView 清除指定账号的可见范围短缓存（登录成功时调用，保证拉到最新 view）。
func (s *Service) InvalidateZentaoView(account string) {
	account = strings.TrimSpace(account)
	if s == nil || account == "" {
		return
	}
	s.zentaoViewMu.Lock()
	defer s.zentaoViewMu.Unlock()
	if s.zentaoViewCache == nil {
		return
	}
	delete(s.zentaoViewCache, account)
}

// parseCSVUintIDs 解析禅道 view 字段（逗号分隔 ID），去重且忽略非法段。
func parseCSVUintIDs(raw string) []uint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []uint{}
	}
	parts := strings.Split(raw, ",")
	out := make([]uint, 0, len(parts))
	seen := make(map[uint]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil || n == 0 {
			continue
		}
		id := uint(n)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
