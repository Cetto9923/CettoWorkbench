package integration

import (
	"errors"
	"os"
	"testing"
)

func TestValidateTestDSN(t *testing.T) {
	// 备份环境变量
	origDDL := os.Getenv("WB_TEST_ALLOW_DDL")
	origCI := os.Getenv("CI")
	origGH := os.Getenv("GITHUB_ACTIONS")
	defer func() {
		os.Setenv("WB_TEST_ALLOW_DDL", origDDL)
		os.Setenv("CI", origCI)
		os.Setenv("GITHUB_ACTIONS", origGH)
	}()

	// 模拟本地环境（非 CI）
	os.Unsetenv("CI")
	os.Unsetenv("GITHUB_ACTIONS")
	os.Unsetenv("WB_TEST_ALLOW_DDL")

	tests := []struct {
		name        string
		dsn         string
		requireDDL  bool
		setOptIn    bool
		setCI       bool
		wantErrType error
	}{
		{
			name:        "Empty DSN",
			dsn:         "",
			requireDDL:  false,
			wantErrType: ErrMissingDSN,
		},
		{
			name:        "Business database zentaopms",
			dsn:         "root:pass@tcp(127.0.0.1:3306)/zentaopms?charset=utf8mb4",
			requireDDL:  false,
			wantErrType: ErrUnsafeDatabase,
		},
		{
			name:        "Production database prod_workbench",
			dsn:         "root:pass@tcp(127.0.0.1:3306)/prod_workbench?charset=utf8mb4",
			requireDDL:  false,
			wantErrType: ErrUnsafeDatabase,
		},
		{
			name:        "Missing _test in db name",
			dsn:         "root:pass@tcp(127.0.0.1:3306)/workbench_dev?charset=utf8mb4",
			requireDDL:  false,
			wantErrType: ErrUnsafeDatabase,
		},
		{
			name:        "Remote production host rejected",
			dsn:         "root:pass@tcp(192.168.1.50:3306)/workbench_test?charset=utf8mb4",
			requireDDL:  false,
			wantErrType: ErrDisallowedHost,
		},
		{
			name:        "Remote domain host rejected",
			dsn:         "root:pass@tcp(db.corp.net:3306)/workbench_test?charset=utf8mb4",
			requireDDL:  false,
			wantErrType: ErrDisallowedHost,
		},
		{
			name:        "Valid DSN on port 3307 but missing WB_TEST_ALLOW_DDL",
			dsn:         "workbench:pass@tcp(127.0.0.1:3307)/workbench_test?charset=utf8mb4",
			requireDDL:  true,
			setOptIn:    false,
			wantErrType: ErrMissingDDLOptIn,
		},
		{
			name:        "Valid DSN on port 3307 with WB_TEST_ALLOW_DDL=1",
			dsn:         "workbench:pass@tcp(127.0.0.1:3307)/workbench_test?charset=utf8mb4",
			requireDDL:  true,
			setOptIn:    true,
			wantErrType: nil,
		},
		{
			name:        "Valid DSN on port 3306 in CI environment",
			dsn:         "workbench:workbench_ci@tcp(127.0.0.1:3306)/workbench_test?charset=utf8mb4",
			requireDDL:  true,
			setCI:       true,
			wantErrType: nil,
		},
		{
			name:        "Valid read-only check without DDL flag",
			dsn:         "workbench:pass@tcp(localhost:3307)/isolated_test?charset=utf8mb4",
			requireDDL:  false,
			wantErrType: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setOptIn {
				os.Setenv("WB_TEST_ALLOW_DDL", "1")
			} else {
				os.Unsetenv("WB_TEST_ALLOW_DDL")
			}

			if tc.setCI {
				os.Setenv("CI", "true")
			} else {
				os.Unsetenv("CI")
			}

			err := ValidateTestDSN(tc.dsn, tc.requireDDL)
			if tc.wantErrType != nil {
				if err == nil {
					t.Fatalf("expected error type %v, got nil", tc.wantErrType)
				}
				if !errors.Is(err, tc.wantErrType) {
					t.Fatalf("expected error wrapping %v, got: %v", tc.wantErrType, err)
				}
			} else if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}
