// =============================================================================
// 文件: internal/module/po/service_clarify_text_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 模拟禅道响应，检查澄清说明实际提交的原文和修改文本。
// =============================================================================
package po

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"workbench/internal/config"
	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

func TestClarifyDemandPreservesOriginalDescription(t *testing.T) {
	for _, submitted := range []string{"abc\ndef", "修改后的说明", ""} {
		t.Run(submitted, func(t *testing.T) { testClarifyDemandDescription(t, submitted) })
	}
}

func testClarifyDemandDescription(t *testing.T, submitted string) {
	t.Helper()
	original := "<p>abc</p><br>def"
	var actual zentao.DemandClarifyParams
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/tokens" {
			_, _ = w.Write([]byte(`{"token":"mock-token"}`))
			return
		}
		if r.URL.Path != "/demand/1002/clarify" {
			t.Errorf("unexpected URL: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&actual); err != nil {
			t.Error(err)
		}
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer upstream.Close()
	old := zentao.DefaultClient()
	zentao.SetDefaultClient(zentao.NewClient(config.ZentaoConfig{API: upstream.URL}))
	t.Cleanup(func() { zentao.SetDefaultClient(old) })
	s, mock := newServiceForDeliverTest(t)
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_demand.*WHERE id = \?`).WithArgs(1002, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "hang", "deleted", "clarifyDesc"}).AddRow(1002, "active", "0", "0", original))
	mock.ExpectQuery("SELECT .*zt_config").WillReturnRows(sqlmock.NewRows([]string{"key", "value"}))
	req := DemandClarifySubmitReq{ID: 1002, Category: "safe", BRA: "003030", ClarifyDesc: submitted,
		Products: []string{"1"}, PM: []string{"003030"}, IsMainSystem: map[string]string{"0": "1"},
		IsNewProduct: "0", IsRelatedAccounts: "0", IsNewFunction: "0", IsOtherImportantOrder: "0", MultiLegalPersonLogo: "0"}
	if err := s.ClarifyDemand(context.Background(), &model.User{Account: "003030"}, req); err != nil {
		t.Fatal(err)
	}
	want := submitted
	if submitted == "abc\ndef" {
		want = original
	}
	if actual.ClarifyDesc != want {
		t.Fatalf("sent %q want %q", actual.ClarifyDesc, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
