// =============================================================================
// 文件: internal/middleware/permission_test.go
// 模块: 中间件
// 类型: middleware
// 职责: RequirePerm 中间件的回归测试，覆盖以下场景：
//       1. 未登录 → 401
//       2. 超级管理员 → 短路放行
//       3. 系统内置自助权限（perm.AuthLogout / perm.PoHome）对所有登录用户放行
//       4. 普通权限按 userPerms 校验
//       5. 命中系统自助权限时不再依赖 userPerms（即便角色没勾选也应放行）
// 依赖: internal/model
//       internal/pkg/perm
// =============================================================================

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"workbench/internal/model"
	"workbench/internal/pkg/perm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newCtx 构造一个最小化的 gin.Context，注入 currentUser 与可选的 userPerms。
func newCtx(u *model.User, perms map[string]bool) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if u != nil {
		c.Set("currentUser", u)
	}
	if perms != nil {
		c.Set("userPerms", perms)
	}
	return c, w
}

func runPerm(t *testing.T, u *model.User, perms map[string]bool, p perm.Permission) *httptest.ResponseRecorder {
	t.Helper()
	c, w := newCtx(u, perms)
	handler := RequirePerm(p)
	handler(c)
	if !c.IsAborted() {
		c.Status(http.StatusOK)
	}
	return w
}

func TestRequirePerm_UnauthenticatedReturns401(t *testing.T) {
	w := runPerm(t, nil, nil, perm.PoHome)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestRequirePerm_SuperAdminShortCircuit(t *testing.T) {
	u := &model.User{ID: 1, Account: "admin", IsSuperAdmin: true}
	// 即便 userPerms 为空、perm 不在 systemPerms，超管也应放行
	w := runPerm(t, u, nil, perm.UserList)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestRequirePerm_SystemPerm_PoHome_AllowsAnyLoggedUser(t *testing.T) {
	u := &model.User{ID: 2, Account: "alice", IsSuperAdmin: false}
	// 关键场景：角色没有任何 perm，老用户登录访问 /home 应放行
	w := runPerm(t, u, nil, perm.PoHome)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestRequirePerm_SystemPerm_AuthLogout_StillAllowsAnyLoggedUser(t *testing.T) {
	u := &model.User{ID: 3, Account: "bob", IsSuperAdmin: false}
	w := runPerm(t, u, nil, perm.AuthLogout)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestRequirePerm_RegularPerm_MissingDenied(t *testing.T) {
	u := &model.User{ID: 4, Account: "carol", IsSuperAdmin: false}
	// 普通用户没拿到 user:list 时应被 403
	w := runPerm(t, u, map[string]bool{}, perm.UserList)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Code)
	}
}

func TestRequirePerm_RegularPerm_Allowed(t *testing.T) {
	u := &model.User{ID: 5, Account: "dave", IsSuperAdmin: false}
	w := runPerm(t, u, map[string]bool{perm.UserList.String(): true}, perm.UserList)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}
