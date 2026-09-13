// 验证角色字典、key 顺序与默认允许角色推导逻辑。
package workbenchroles

import (
	"reflect"
	"testing"
)

// 期望的 6 个角色 key（与 Role* 常量语义保持一致，便于断言）。
var allExpectedKeys = []string{RolePO, RoleLead, RoleSM, RoleDev, RoleQA, RolePMO}

func TestRoleMap(t *testing.T) {
	m := RoleMap()
	if got, want := len(m), len(allExpectedKeys); got != want {
		t.Fatalf("RoleMap() 数量 = %d, 期望 %d", got, want)
	}
	for _, key := range allExpectedKeys {
		def, ok := m[key]
		if !ok {
			t.Errorf("RoleMap() 缺失 key %q", key)
			continue
		}
		if def.Label == "" {
			t.Errorf("RoleMap()[%q].Label 为空，必须非空", key)
		}
	}
}

func TestAllRoleKeys(t *testing.T) {
	got := AllRoleKeys()
	if len(got) != len(allExpectedKeys) {
		t.Fatalf("AllRoleKeys() 长度 = %d, 期望 %d (got=%v)", len(got), len(allExpectedKeys), got)
	}
	// 顺序稳定：第一个必须是 RolePO。
	if got[0] != RolePO {
		t.Errorf("AllRoleKeys()[0] = %q, 期望 %q (PO 必须排首位)", got[0], RolePO)
	}
	// 全集相等且无重复。
	if !reflect.DeepEqual(got, allExpectedKeys) {
		t.Errorf("AllRoleKeys() = %v, 期望 %v", got, allExpectedKeys)
	}
	seen := make(map[string]struct{}, len(got))
	for _, k := range got {
		if _, dup := seen[k]; dup {
			t.Errorf("AllRoleKeys() 含重复 key %q", k)
		}
		seen[k] = struct{}{}
	}
}

func TestDefaultAllowedFor(t *testing.T) {
	// 超管 → 全部 6 个 key，且与 AllRoleKeys() 完全一致。
	admin := DefaultAllowedFor("admin", true, 0)
	if !reflect.DeepEqual(admin, AllRoleKeys()) {
		t.Errorf("super admin 允许角色 = %v, 期望 %v", admin, AllRoleKeys())
	}

	// 普通账号 → 固定 PO/Dev/QA，且顺序稳定。
	normal := DefaultAllowedFor("alice", false, 123)
	wantNormal := []string{RolePO, RoleDev, RoleQA}
	if !reflect.DeepEqual(normal, wantNormal) {
		t.Errorf("普通账号默认允许角色 = %v, 期望 %v", normal, wantNormal)
	}

	// deptID 不参与判定，相同 isSuperAdmin=false 在不同 deptID 下结果必须一致。
	if !reflect.DeepEqual(
		DefaultAllowedFor("alice", false, 0),
		DefaultAllowedFor("bob", false, 999),
	) {
		t.Error("DefaultAllowedFor 在 isSuperAdmin=false 时必须忽略 deptID/account")
	}
}
