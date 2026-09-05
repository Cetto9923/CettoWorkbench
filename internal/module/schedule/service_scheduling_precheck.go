package schedule

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
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

// loadDemandStoryScopeAndWindow 批量加载当前业务需求及子需求的已有研发需求和生效窗口。
// 严禁信任客户端提交的 fromDemand / product 关系，以服务端真实关系为准。
func (s *Service) loadDemandStoryScopeAndWindow(ctx context.Context, demandID uint) (uint, map[uint]ZtStory, error) {
	demandIDs := []uint{demandID}
	children, err := s.repo.FindChildDemandsByParents(ctx, []uint{demandID})
	if err != nil {
		return 0, nil, err
	}
	demandIDs = mergeDemandIDs(demandIDs, pluckDemandIDs(children))
	stories, err := s.repo.FindStoriesByDemands(ctx, demandIDs)
	if err != nil {
		return 0, nil, err
	}
	windowByDemand, err := s.repo.FindDemandWindowMappings(ctx, demandIDs)
	if err != nil {
		return 0, nil, err
	}
	windowByStory, err := s.repo.FindStoryWindowMappings(ctx, pluckStoryIDs(stories))
	if err != nil {
		return 0, nil, err
	}
	windowID := pickDemandWindowID(demandIDs, stories, windowByDemand, windowByStory)
	allowedByID := make(map[uint]ZtStory, len(stories))
	for _, st := range stories {
		allowedByID[st.ID] = st
	}
	return windowID, allowedByID, nil
}

// validateSaveSchedulingStoryScope 校验 edit 与 delete 操作的研发需求必须属于当前需求（或其子需求）已有对象集合。
// 客户端伪造不存在或归属其他需求的 ID 均在事务前被拦截并返回 403。
func validateSaveSchedulingStoryScope(stories []SaveSchedulingStory, allowedByID map[uint]ZtStory) error {
	for _, storyReq := range stories {
		action := strings.TrimSpace(storyReq.Action)
		if action == "edit" || action == "delete" {
			if storyReq.ID == 0 {
				return &TaskMutationError{Code: taskMutationNotFound, Message: "研发需求不存在"}
			}
			if _, ok := allowedByID[storyReq.ID]; !ok {
				return &TaskMutationError{
					Code:    taskMutationForbidden,
					Message: fmt.Sprintf("研发需求 %d 不属于当前业务需求", storyReq.ID),
				}
			}
		}
	}
	return nil
}

// precheckDemandSchedulingProducts 业需排期保存前置只读产品权限校验：
// 1. new: 校验目标系统 ProductID;
// 2. edit: 校验目标系统 ProductID 以及已有需求当前真实 ProductID（从数据库已有记录取）;
// 3. delete: 校验已有需求当前真实 ProductID（严禁因无客户端 ProductID 而跳过校验）。
func (s *Service) precheckDemandSchedulingProducts(
	ctx context.Context,
	windowID uint,
	stories []SaveSchedulingStory,
	allowedByID map[uint]ZtStory,
	account string,
) (*ProductAccessNoticeError, error) {
	_ = windowID
	var rawProductIDs []uint
	for _, story := range stories {
		switch strings.TrimSpace(story.Action) {
		case "new":
			if story.ProductID != 0 {
				rawProductIDs = append(rawProductIDs, story.ProductID)
			}
		case "edit":
			if story.ProductID != 0 {
				rawProductIDs = append(rawProductIDs, story.ProductID)
			}
			if existing, ok := allowedByID[story.ID]; ok && existing.Product != 0 {
				rawProductIDs = append(rawProductIDs, existing.Product)
			}
		case "delete":
			if existing, ok := allowedByID[story.ID]; ok && existing.Product != 0 {
				rawProductIDs = append(rawProductIDs, existing.Product)
			}
		}
	}
	productIDs := uniqueUints(rawProductIDs)
	return s.validateProductsAccess(ctx, productIDs, account)
}

// precheckSchedulingProducts 排期保存前置只读校验（保留独立研发需求与任务弹窗现有调用契约）。
func (s *Service) precheckSchedulingProducts(ctx context.Context, windowID uint, stories []SaveSchedulingStory, account string) (*ProductAccessNoticeError, error) {
	_ = windowID
	var rawProductIDs []uint
	for _, story := range stories {
		switch strings.TrimSpace(story.Action) {
		case "new", "edit":
			if story.ProductID != 0 {
				rawProductIDs = append(rawProductIDs, story.ProductID)
			}
		}
	}
	productIDs := uniqueUints(rawProductIDs)
	return s.validateProductsAccess(ctx, productIDs, account)
}

func (s *Service) validateProductsAccess(ctx context.Context, productIDs []uint, account string) (*ProductAccessNoticeError, error) {
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
