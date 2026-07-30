// =============================================================================
// 文件: internal/module/po/service.go
// 模块: PO 工作台
// 类型: action
// 职责: 组装 PO 首页价值流统计与需求列表（受理/联调测试/评价反馈读 MySQL，其余读 Redis）。
// 依赖: internal/model
//       internal/module/schedule
//       internal/pkg/redis
//       internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/module/schedule"
	redispkg "workbench/internal/pkg/redis"
	"workbench/internal/pkg/zentao"
)

const valueStreamKeyPrefix = "valuestream"

var valueStreamStages = []struct {
	label  string
	status string
}{
	{label: "全部", status: "all"},
	{label: "受理", status: "accept"},
	{label: "澄清", status: "clarify"},
	{label: "排期", status: "schedule"},
	{label: "提测", status: "developing"},
	{label: "联调测试", status: "testing"},
	{label: "验收", status: "waitacceptance"},
	{label: "交付", status: "acceptanced"},
	{label: "评价反馈", status: "released"},
}

type workItemRef struct {
	kind        string
	id          string
	valueStream string
}

// Service PO 工作台业务逻辑。
type Service struct {
	repo     *Repo
	redis    *redispkg.Clients
	schedule *schedule.Service
	logger   *zap.Logger
}

// NewService 创建 Service。
func NewService(repo *Repo, redisClients *redispkg.Clients, scheduleSvc *schedule.Service, logger *zap.Logger) *Service {
	return &Service{repo: repo, redis: redisClients, schedule: scheduleSvc, logger: logger}
}

// Home 加载首页价值流阶段统计。
func (s *Service) Home(ctx context.Context, actor *model.User) (*HomeResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}

	counts, err := s.loadAllStageCounts(ctx, account)
	if err != nil {
		return nil, err
	}

	stages := make([]ValueStreamStage, 0, len(valueStreamStages))
	for i, def := range valueStreamStages {
		demand := counts[i*2]
		story := counts[i*2+1]
		if filter, ok := mysqlStageFilters[def.status]; ok {
			n, countErr := s.repo.CountRoleDemands(ctx, account, filter)
			if countErr != nil {
				return nil, countErr
			}
			demand = n
			story = 0
		}
		stages = append(stages, ValueStreamStage{
			Label:       def.label,
			Status:      def.status,
			Count:       demand + story,
			DemandCount: demand,
			StoryCount:  story,
		})
	}

	versionWindows := []schedule.HomeVersionWindowCard{}
	if s.schedule == nil {
		if s.logger != nil {
			s.logger.Error("po home schedule service is nil, version windows skipped")
		}
	} else {
		windows, winErr := s.schedule.ListHomeVersionWindows(ctx, actor)
		if winErr != nil {
			if s.logger != nil {
				s.logger.Warn("po home version windows", zap.Error(winErr))
			}
		} else {
			versionWindows = windows
		}
	}

	return &HomeResp{Stages: stages, VersionWindows: versionWindows}, nil
}

// loadAllStageCounts 一次 Pipeline 读取全部阶段的 demand/story 数量（1 次网络往返）。
func (s *Service) loadAllStageCounts(ctx context.Context, account string) ([]int64, error) {
	n := len(valueStreamStages) * 2
	out := make([]int64, n)
	if s.redis == nil || s.redis.Clean == nil || account == "" {
		return out, nil
	}

	pipe := s.redis.Clean.Pipeline()
	cmds := make([]*goredis.IntCmd, n)
	for i, def := range valueStreamStages {
		demandKey := fmt.Sprintf("%s:%s:%s:demand", valueStreamKeyPrefix, account, def.status)
		storyKey := fmt.Sprintf("%s:%s:%s:story", valueStreamKeyPrefix, account, def.status)

		cmds[i*2] = pipe.ZCard(ctx, demandKey)
		cmds[i*2+1] = pipe.ZCard(ctx, storyKey)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	for i, cmd := range cmds {
		out[i] = cmd.Val()
	}
	return out, nil
}

// Demands 按价值流状态返回当前用户关联的需求/故事详情。
func (s *Service) Demands(ctx context.Context, actor *model.User, req DemandsReq) (*DemandsResp, error) {
	if filter, ok := mysqlStageFilters[req.Status]; ok {
		return s.listMySQLDemands(ctx, actor, req.Status, filter)
	}

	account := ""
	if actor != nil {
		account = actor.Account
	}

	refs, err := s.listValueStreamRefs(ctx, account, req.Status)
	if err != nil {
		return nil, err
	}
	items, err := s.loadWorkItemDetails(ctx, refs)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []WorkItemDetail{}
	}
	return &DemandsResp{Items: items}, nil
}

// listMySQLDemands 从 MySQL 加载指定价值流阶段的业需列表。
func (s *Service) listMySQLDemands(ctx context.Context, actor *model.User, stageStatus string, filter mysqlStageFilter) (*DemandsResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	rows, err := s.repo.FindRoleDemands(ctx, account, filter)
	if err != nil {
		return nil, err
	}
	label := valueStreamLabelForStatus(stageStatus)
	items := make([]WorkItemDetail, 0, len(rows))
	for _, row := range rows {
		pri := ""
		if row.Pri != "" {
			pri = "P" + row.Pri
		}
		items = append(items, WorkItemDetail{
			Kind:        "demand",
			ID:          fmt.Sprintf("%d", row.ID),
			Pri:         pri,
			Title:       row.Name,
			ZentaoUrl:   zentao.URL("demand", "view", fmt.Sprintf("demandID=%d", row.ID)),
			ValueStream: label,
		})
	}
	return &DemandsResp{Items: items}, nil
}

func isValidValueStreamStatus(status string) bool {
	for _, def := range valueStreamStages {
		if def.status == status {
			return true
		}
	}
	return false
}

func valueStreamLabelForStatus(status string) string {
	for _, def := range valueStreamStages {
		if def.status == status {
			return def.label
		}
	}
	return ""
}

func workItemStageKey(kind, id string) string {
	return kind + ":" + id
}

// loadWorkItemStageMap 构建 kind:id → 价值流阶段名称（不含「全部」）。
func (s *Service) loadWorkItemStageMap(ctx context.Context, account string) (map[string]string, error) {
	out := make(map[string]string)
	if s.redis == nil || s.redis.Clean == nil || account == "" {
		return out, nil
	}

	kinds := []struct {
		kind   string
		suffix string
	}{
		{kind: "demand", suffix: "demand"},
		{kind: "story", suffix: "story"},
	}

	type zrangeEntry struct {
		label string
		kind  string
		cmd   *goredis.StringSliceCmd
	}

	pipe := s.redis.Clean.Pipeline()
	entries := make([]zrangeEntry, 0)
	for _, def := range valueStreamStages {
		if def.status == "all" {
			continue
		}
		for _, k := range kinds {
			key := fmt.Sprintf("%s:%s:%s:%s", valueStreamKeyPrefix, account, def.status, k.suffix)
			entries = append(entries, zrangeEntry{
				label: def.label,
				kind:  k.kind,
				cmd:   pipe.ZRange(ctx, key, 0, -1),
			})
		}
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	for _, e := range entries {
		ids, err := e.cmd.Result()
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if id == "" {
				continue
			}
			mapKey := workItemStageKey(e.kind, id)
			if _, exists := out[mapKey]; exists {
				continue
			}
			out[mapKey] = e.label
		}
	}
	return out, nil
}

func (s *Service) listValueStreamRefs(ctx context.Context, account, status string) ([]workItemRef, error) {
	if s.redis == nil || s.redis.Clean == nil || account == "" {
		return nil, nil
	}

	kinds := []struct {
		kind   string
		suffix string
	}{
		{kind: "demand", suffix: "demand"},
		{kind: "story", suffix: "story"},
	}

	var stageMap map[string]string
	var fixedLabel string
	if status == "all" {
		var err error
		stageMap, err = s.loadWorkItemStageMap(ctx, account)
		if err != nil {
			return nil, err
		}
	} else {
		fixedLabel = valueStreamLabelForStatus(status)
	}

	refs := make([]workItemRef, 0)
	for _, k := range kinds {
		key := fmt.Sprintf("%s:%s:%s:%s", valueStreamKeyPrefix, account, status, k.suffix)
		ids, err := s.redis.Clean.ZRangeArgs(ctx, goredis.ZRangeArgs{
			Key:   key,
			Start: 0,
			Stop:  -1,
			Rev:   true,
		}).Result()
		if err != nil {
			return nil, fmt.Errorf("zrange %s: %w", key, err)
		}
		for _, id := range ids {
			if id == "" {
				continue
			}
			vs := fixedLabel
			if status == "all" {
				vs = stageMap[workItemStageKey(k.kind, id)]
			}
			refs = append(refs, workItemRef{kind: k.kind, id: id, valueStream: vs})
		}
	}
	return refs, nil
}

func (s *Service) loadWorkItemDetails(ctx context.Context, refs []workItemRef) ([]WorkItemDetail, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	if s.redis == nil || s.redis.Source == nil {
		return nil, nil
	}

	pipe := s.redis.Source.Pipeline()
	cmds := make([]*goredis.MapStringStringCmd, len(refs))
	for i, ref := range refs {
		key := workItemRedisKey(ref.kind, ref.id)
		cmds[i] = pipe.HGetAll(ctx, key)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	items := make([]WorkItemDetail, 0, len(refs))
	for i, ref := range refs {
		var title, pri, zentaoUrl string

		detail, err := cmds[i].Result()
		if err != nil {
			return nil, err
		}
		if len(detail) == 0 {
			continue
		}

		if ref.kind == "demand" {
			title = detail["name"]
			zentaoUrl = zentao.URL("demand", "view", fmt.Sprintf("demandID=%s", detail["id"]))
		} else {
			title = detail["title"]
			zentaoUrl = zentao.URL("story", "view", fmt.Sprintf("storyID=%s", detail["id"]))
		}

		if detail["pri"] != "" {
			pri = "P" + detail["pri"]
		}

		items = append(items, WorkItemDetail{
			Kind:        ref.kind,
			ID:          detail["id"],
			Pri:         pri,
			Title:       title,
			ZentaoUrl:   zentaoUrl,
			ValueStream: ref.valueStream,
		})
	}
	return items, nil
}

func workItemRedisKey(kind, id string) string {
	switch kind {
	case "story":
		return fmt.Sprintf("zt_story:%s", id)
	default:
		return fmt.Sprintf("zt_demand:%s", id)
	}
}
