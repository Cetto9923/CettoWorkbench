// =============================================================================
// 文件: internal/module/profile/service_test.go
// 模块: 个人资料
// 类型: action
// 职责: 个人资料单测：覆盖 Validate、Actor 校验、密码长度 / 一致性等不依赖 DB 的逻辑。
// 依赖: testify（如未引入使用标准 testing）
// =============================================================================

package profile

import (
	"strings"
	"testing"

	"workbench/internal/pkg/encode"
	"workbench/internal/pkg/errorx"
)

func TestUpdateReq_Validate_NameRequired(t *testing.T) {
	req := UpdateReq{DisplayName: "", Email: "a@b.com", Gender: "m"}
	if errs := req.Validate(); len(errs) == 0 {
		t.Fatalf("expected validation error for empty name")
	}
}

func TestUpdateReq_Validate_EmailFormat(t *testing.T) {
	req := UpdateReq{DisplayName: "x", Email: "not-email", Gender: "m"}
	if errs := req.Validate(); len(errs) == 0 {
		t.Fatalf("expected email format error")
	}
}

func TestUpdateReq_Validate_EmailOptional(t *testing.T) {
	req := UpdateReq{DisplayName: "x", Email: "", Gender: "m"}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("empty email should be allowed; got %v", errs)
	}
}

func TestUpdateReq_Validate_GenderEnum(t *testing.T) {
	req := UpdateReq{DisplayName: "x", Gender: "x"}
	if errs := req.Validate(); len(errs) == 0 {
		t.Fatalf("invalid gender must error")
	}
}

func TestUpdateReq_Validate_GenderEmptyOK(t *testing.T) {
	req := UpdateReq{DisplayName: "x", Gender: ""}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("empty gender must be allowed (means skip); got %v", errs)
	}
}

func TestChangePasswordReq_Validate_EmptyOld(t *testing.T) {
	req := ChangePasswordReq{OldPassword: "", NewPassword: "abc12345", ConfirmPassword: "abc12345"}
	if errs := req.Validate(); len(errs) == 0 {
		t.Fatalf("empty old password must error")
	}
}

func TestChangePasswordReq_Validate_TooShort(t *testing.T) {
	req := ChangePasswordReq{OldPassword: "old12345", NewPassword: "short", ConfirmPassword: "short"}
	if errs := req.Validate(); len(errs) == 0 {
		t.Fatalf("short password must error")
	}
}

func TestChangePasswordReq_Validate_Mismatch(t *testing.T) {
	req := ChangePasswordReq{OldPassword: "old12345", NewPassword: "new12345", ConfirmPassword: "other1234"}
	if errs := req.Validate(); len(errs) == 0 {
		t.Fatalf("mismatched confirmation must error")
	}
}

func TestChangePasswordReq_Validate_OK(t *testing.T) {
	req := ChangePasswordReq{OldPassword: "old12345", NewPassword: "new12345", ConfirmPassword: "new12345"}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("valid password change should pass; got %v", errs)
	}
}

func TestServiceGet_NoActor(t *testing.T) {
	s := &Service{repo: nil}
	_, err := s.Get(nil, nil)
	if err == nil {
		t.Fatalf("nil actor must error")
	}
	if biz, ok := errorx.IsBizError(err); !ok || biz.Code != "unauthorized" {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceChangePassword_NotAuthenticated(t *testing.T) {
	s := &Service{repo: nil}
	err := s.ChangePassword(nil, nil, ChangePasswordReq{})
	if err == nil {
		t.Fatalf("nil actor must error")
	}
}

func TestEncodeMD5_KnownVector(t *testing.T) {
	// MD5("123456") = e10adc3949ba59abbe56e057f20f883e
	got := encode.MD5("123456")
	if !strings.EqualFold(got, "e10adc3949ba59abbe56e057f20f883e") {
		t.Fatalf("MD5(123456) mismatch: %s", got)
	}
}
