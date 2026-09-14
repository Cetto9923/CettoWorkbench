// =============================================================================
// 文件: internal/pkg/encode/encode.go
// 模块: 基础设施
// 类型: infra
// 职责: 提供与禅道 zt_user 兼容的 MD5 密码摘要。
// 依赖: 无
//
// COMPATIBILITY ONLY: ZenTao zt_user.password hex-MD5. Do not reuse for new
// Workbench-owned auth stores. See docs/plan/tech-debt-governance-20260914/
// password-hash-decision.md.
// =============================================================================

package encode

import (
	"crypto/md5"
	"encoding/hex"
)

// MD5 计算明文密码的 MD5 十六进制摘要（与禅道 zt_user.password 存储规则一致）。
// 仅用于共享禅道用户表的兼容读写；新建认证存储不得调用本函数。
func MD5(plain string) string {
	sum := md5.Sum([]byte(plain))
	return hex.EncodeToString(sum[:])
}
