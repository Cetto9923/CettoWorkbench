// =============================================================================
// 文件: internal/module/po/service.go
// 模块: PO 工作台
// 类型: action
// 职责: 从 Redis 读取价值流 zset 统计并组装首页数据。
// 依赖: internal/model
//       internal/pkg/redis
// =============================================================================

package po

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"workbench/internal/model"
	redispkg "workbench/internal/pkg/redis"
)

const valueStreamKeyPrefix = "valuestream"

var valueStreamStages = []struct {
	label  string
	status string
	isAll  bool
}{
	{label: "全部", status: "all", isAll: true},
	{label: "受理", status: "accept"},
	{label: "澄清", status: "clarify"},
	{label: "排期", status: "schedule"},
	{label: "提测", status: "developing"},
	{label: "联调测试", status: "testing"},
	{label: "验收", status: "waitacceptance"},
	{label: "交付", status: "acceptanced"},
	{label: "评价反馈", status: "released"},
}

// Service PO 工作台业务逻辑。
type Service struct {
	redis  *redispkg.Clients
	logger *zap.Logger
}

// NewService 创建 Service。
func NewService(redisClients *redispkg.Clients, logger *zap.Logger) *Service {
	return &Service{redis: redisClients, logger: logger}
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
		stages = append(stages, ValueStreamStage{
			Label:       def.label,
			Status:      def.status,
			Count:       demand + story,
			DemandCount: demand,
			StoryCount:  story,
			IsAll:       def.isAll,
		})
	}
	return &HomeResp{Stages: stages}, nil
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
