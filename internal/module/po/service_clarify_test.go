package po

import (
	"testing"

	"workbench/internal/pkg/personlabel"
)

func TestDemandClarifySubmitReq_Validate(t *testing.T) {
	noAiCategories := []string{"management", "operation"}
	aiCategories := []string{"feature", "tech"}

	t.Run("empty required fields", func(t *testing.T) {
		req := DemandClarifySubmitReq{}
		errs := req.Validate(noAiCategories, aiCategories)

		if errs["category"] == "" {
			t.Errorf("expected error for category, got none")
		}
		if errs["bra"] == "" {
			t.Errorf("expected error for bra, got none")
		}
		if errs["isMainSystem"] == "" {
			t.Errorf("expected error for isMainSystem, got none")
		}
		if errs["products"] == "" {
			t.Errorf("expected error for products, got none")
		}
		if errs["importantOrder"] == "" {
			t.Errorf("expected error for importantOrder, got none")
		}
	})

	t.Run("duplicate products rejected", func(t *testing.T) {
		req := DemandClarifySubmitReq{
			Category: "feature",
			BRA:      "user01",
			Products: []string{"101", "101"},
			PM:       []string{"user01", "user02"},
			IsMainSystem: map[string]string{
				"0": "1",
			},
			IsNewProduct:          "0",
			IsRelatedAccounts:     "0",
			IsNewFunction:         "0",
			IsOtherImportantOrder: "0",
			MultiLegalPersonLogo:  "0",
		}
		errs := req.Validate(noAiCategories, aiCategories)
		if errs["products"] != "涉及产品不能重复" {
			t.Errorf("expected '涉及产品不能重复', got %q", errs["products"])
		}
	})

	t.Run("additional info requires link", func(t *testing.T) {
		req := DemandClarifySubmitReq{
			Category: "management",
			BRA:      "user01",
			Products: []string{"101"},
			PM:       []string{"user01"},
			IsMainSystem: map[string]string{
				"0": "1",
			},
			IsAdditionalInfo:      []string{"1"},
			AdditionalInfo:        []string{""},
			IsNewProduct:          "0",
			IsRelatedAccounts:     "0",
			IsNewFunction:         "0",
			IsOtherImportantOrder: "0",
			MultiLegalPersonLogo:  "0",
		}
		errs := req.Validate(noAiCategories, aiCategories)
		if errs["additionalInfo"] != "总领文档链接不能为空" {
			t.Errorf("expected '总领文档链接不能为空', got %q", errs["additionalInfo"])
		}
	})

	t.Run("ai category requires at least one checked story and valid scale estimation", func(t *testing.T) {
		req := DemandClarifySubmitReq{
			Category: "feature",
			BRA:      "user01",
			Products: []string{"101"},
			PM:       []string{"user01"},
			IsMainSystem: map[string]string{
				"0": "1",
			},
			ScaleEstimation:       0,
			UserStoryChecked:      map[string]string{},
			IsNewProduct:          "0",
			IsRelatedAccounts:     "0",
			IsNewFunction:         "0",
			IsOtherImportantOrder: "0",
			MultiLegalPersonLogo:  "0",
		}
		errs := req.Validate(noAiCategories, aiCategories)
		if errs["scaleEstimation"] == "" {
			t.Errorf("expected error for scaleEstimation <= 0, got none")
		}
		if errs["userStoryChecked"] != "当前需求类别必须至少采纳一条用户故事条目" {
			t.Errorf("expected story checked error, got %q", errs["userStoryChecked"])
		}
	})

	t.Run("checked story requires role, gv, product and revpoint", func(t *testing.T) {
		req := DemandClarifySubmitReq{
			Category: "feature",
			BRA:      "user01",
			Products: []string{"101"},
			PM:       []string{"user01"},
			IsMainSystem: map[string]string{
				"0": "1",
			},
			ScaleEstimation: 3,
			UserStoryChecked: map[string]string{
				"0": "1",
			},
			Role:                  []string{""},
			GV:                    []string{""},
			EntryProductID:        []string{""},
			Revpoint:              []string{""},
			IsNewProduct:          "0",
			IsRelatedAccounts:     "0",
			IsNewFunction:         "0",
			IsOtherImportantOrder: "0",
			MultiLegalPersonLogo:  "0",
		}
		errs := req.Validate(noAiCategories, aiCategories)
		if errs["userStoryRole"] == "" {
			t.Errorf("expected error for empty userStoryRole, got none")
		}
		if errs["userStoryGV"] == "" {
			t.Errorf("expected error for empty userStoryGV, got none")
		}
		if errs["userStoryProduct"] == "" {
			t.Errorf("expected error for empty userStoryProduct, got none")
		}
		if errs["userStoryRevpoint"] == "" {
			t.Errorf("expected error for empty userStoryRevpoint, got none")
		}
	})

	t.Run("noAiCategory skips story requirement and scale estimation", func(t *testing.T) {
		req := DemandClarifySubmitReq{
			Category: "management",
			BRA:      "user01",
			Products: []string{"101"},
			PM:       []string{"user01"},
			IsMainSystem: map[string]string{
				"0": "1",
			},
			IsNewProduct:          "0",
			IsRelatedAccounts:     "0",
			IsNewFunction:         "0",
			IsOtherImportantOrder: "0",
			MultiLegalPersonLogo:  "0",
		}
		errs := req.Validate(noAiCategories, aiCategories)
		if len(errs) != 0 {
			t.Errorf("expected no validation errors for valid noAiCategory, got: %v", errs)
		}
	})

	t.Run("fully valid form passes", func(t *testing.T) {
		req := DemandClarifySubmitReq{
			Category: "feature",
			BRA:      "user01",
			Products: []string{"101"},
			PM:       []string{"user01"},
			IsMainSystem: map[string]string{
				"0": "1",
			},
			ScaleEstimation: 5,
			UserStoryChecked: map[string]string{
				"0": "1",
			},
			Role:                  []string{"终端用户"},
			GV:                    []string{"能够查看订单列表以便掌握进度"},
			EntryProductID:        []string{"101"},
			Revpoint:              []string{"5"},
			IsNewProduct:          "0",
			IsRelatedAccounts:     "1",
			IsNewFunction:         "0",
			IsOtherImportantOrder: "0",
			MultiLegalPersonLogo:  "0",
		}
		errs := req.Validate(noAiCategories, aiCategories)
		if len(errs) != 0 {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})

	t.Run("multiLegalPersonLogo valid and invalid values", func(t *testing.T) {
		validLogos := []string{"0", "1", "2", "changshu", "village", "changshu_village"}
		for _, logo := range validLogos {
			req := DemandClarifySubmitReq{
				Category:              "management",
				BRA:                   "user01",
				Products:              []string{"101"},
				PM:                    []string{"user01"},
				IsMainSystem:          map[string]string{"0": "1"},
				IsNewProduct:          "0",
				IsRelatedAccounts:     "0",
				IsNewFunction:         "0",
				IsOtherImportantOrder: "0",
				MultiLegalPersonLogo:  logo,
			}
			errs := req.Validate(noAiCategories, aiCategories)
			if errs["importantOrder"] != "" {
				t.Errorf("expected logo %q to pass validation, got %v", logo, errs["importantOrder"])
			}
		}

		invalidLogos := []string{"", "-1"}
		for _, logo := range invalidLogos {
			req := DemandClarifySubmitReq{
				Category:              "management",
				BRA:                   "user01",
				Products:              []string{"101"},
				PM:                    []string{"user01"},
				IsMainSystem:          map[string]string{"0": "1"},
				IsNewProduct:          "0",
				IsRelatedAccounts:     "0",
				IsNewFunction:         "0",
				IsOtherImportantOrder: "0",
				MultiLegalPersonLogo:  logo,
			}
			errs := req.Validate(noAiCategories, aiCategories)
			if errs["importantOrder"] == "" {
				t.Errorf("expected logo %q to fail validation, got none", logo)
			}
		}
	})

	t.Run("user options formatting does not duplicate account suffix", func(t *testing.T) {
		tests := []struct {
			account  string
			realname string
			want     string
		}{
			{"003030", "程统 (003030)", "程统 (003030)"},
			{"003030", "程统(003030)", "程统(003030)"},
			{"zhangsan", "张三", "张三(zhangsan)"},
			{"001234", "李四 (001234)", "李四 (001234)"},
			{"002345", "", "002345"},
		}
		for _, tt := range tests {
			got := personlabel.Format(tt.account, tt.realname)
			if got != tt.want {
				t.Errorf("personlabel.Format(%q, %q) = %q, want %q", tt.account, tt.realname, got, tt.want)
			}
		}
	})
}
