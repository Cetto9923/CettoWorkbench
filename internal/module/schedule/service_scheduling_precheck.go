package schedule

import (
	"context"
	"sort"
	"strconv"
)

// NoticeProduct 产品访问拦截提示中的产品条目。
// JSON tag 必须小写，前端按 products[].id / products[].name 读取。
type NoticeProduct struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	ViewURL string `json:"viewUrl"` // 禅道产品概况页完整链接，由 handler 用 zentao.ProductViewURLWithBase 填充
}

// ProductAccessNoticeError 排期前置校验失败错误。
// 当存在「既不在版本窗口中、又无匹配计划」的系统时返回，
// 由上层转换为前端二次确认提示。
type ProductAccessNoticeError struct {
	Products []NoticeProduct
}

// Error 实现 error 接口。
func (e *ProductAccessNoticeError) Error() string {
	return "存在未在窗口中且无匹配计划的系统"
}

// precheckSchedulingProducts 排期保存前置只读校验：
// 收集本次提交中 new/edit 涉及的系统，判断是否在当前用户有权限操作的产品集合内，
// 不在则记入拦截列表。本函数为只读，不写库、不进事务，
// 在 SaveScheduling 事务开启之前调用。
func (s *Service) precheckSchedulingProducts(ctx context.Context, windowID uint, stories []SaveSchedulingStory, account string) (*ProductAccessNoticeError, error) {
	// 收集 new/edit 涉及的系统 ID（跳过 delete 与空 action）。
	var rawProductIDs []uint
	for _, story := range stories {
		switch story.Action {
		case "new", "edit":
			if story.ProductID != 0 {
				rawProductIDs = append(rawProductIDs, story.ProductID)
			}
		}
	}
	productIDs := uniqueUints(rawProductIDs)
	if len(productIDs) == 0 {
		return nil, nil
	}

	// 查询当前用户有权限操作的产品集合。
	products, err := s.repo.GetUserProducts(ctx, account)
	if err != nil {
		return nil, err
	}
	accessSet := make(map[uint]struct{}, len(products))
	for _, p := range products {
		accessSet[p.ID] = struct{}{}
	}

	// 遍历去重后的系统，不在有权集合内则记入拦截列表。
	var offenders []uint
	for _, productID := range productIDs {
		if _, ok := accessSet[productID]; !ok {
			offenders = append(offenders, productID)
		}
	}
	if len(offenders) == 0 {
		return nil, nil
	}

	// 批量查询系统名称。
	nameByID, err := s.repo.FindProductsByIDs(ctx, offenders)
	if err != nil {
		return nil, err
	}

	// 按 productID 升序构造提示列表。
	sort.Slice(offenders, func(i, j int) bool {
		return offenders[i] < offenders[j]
	})
	noticeProducts := make([]NoticeProduct, 0, len(offenders))
	for _, id := range offenders {
		name := nameByID[id]
		if name == "" {
			name = strconv.FormatUint(uint64(id), 10)
		}
		noticeProducts = append(noticeProducts, NoticeProduct{ID: id, Name: name})
	}
	return &ProductAccessNoticeError{Products: noticeProducts}, nil
}
