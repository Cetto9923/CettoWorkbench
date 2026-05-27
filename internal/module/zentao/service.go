// =============================================================================
// 文件: internal/module/zentao/service.go
// 模块: 禅道集成
// 类型: action
// 职责: 实现禅道 token 获取、会话缓存与产品接口调用逻辑。
// 依赖: internal/config
//
//	internal/model
//
// =============================================================================
package zentao

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"go.uber.org/zap"

	"goframework/internal/config"
)

const (
	zentaoTokenSessionKey       = "zentaoToken"
	zentaoTokenExpireSessionKey = "zentaoTokenExpireAt"
	zentaoTokenTTL              = time.Hour

	// 路由
	zentaoProductsRoute = "/products"
	zentaoTokensRoute   = "/tokens"
)

// Service 处理禅道集成业务逻辑。
type Service struct {
	repo       *Repo
	sessionMgr *scs.SessionManager
	cfg        config.ZentaoConfig
	client     *http.Client
	logger     *zap.Logger
}

// NewService 创建 Service。
func NewService(
	repo *Repo,
	sessionMgr *scs.SessionManager,
	cfg config.ZentaoConfig,
	logger *zap.Logger,
) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repo:       repo,
		sessionMgr: sessionMgr,
		cfg:        cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		logger: logger,
	}
}

// Products 获取禅道产品列表。
func (s *Service) Products(ctx context.Context, req ProductsReq) (ProductsResp, error) {
	_ = req
	token, err := s.getToken(ctx)
	if err != nil {
		return ProductsResp{}, err
	}
	raw, err := s.fetchProducts(ctx, token)
	if err != nil {
		return ProductsResp{}, err
	}

	return ProductsResp{Raw: raw}, nil
}

// Users 获取禅道用户列表（account、realname）。
func (s *Service) Users(ctx context.Context) ([]UserItem, error) {
	if s.repo == nil {
		return nil, errors.New("zentao repo is nil")
	}
	rows, err := s.repo.FindAllActiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]UserItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, UserItem{
			Account:  row.Account,
			Realname: row.Realname,
		})
	}
	return items, nil
}

func (s *Service) getToken(ctx context.Context) (string, error) {
	if s.sessionMgr == nil {
		return "", errors.New("session manager is nil")
	}
	existing := s.sessionMgr.GetString(ctx, zentaoTokenSessionKey)
	expireAt := s.sessionMgr.GetTime(ctx, zentaoTokenExpireSessionKey)
	if existing != "" && !expireAt.IsZero() && time.Now().Before(expireAt) {
		return existing, nil
	}

	token, err := s.requestToken(ctx)
	if err != nil {
		return "", err
	}
	s.sessionMgr.Put(ctx, zentaoTokenSessionKey, token)
	s.sessionMgr.Put(ctx, zentaoTokenExpireSessionKey, time.Now().Add(zentaoTokenTTL))
	return token, nil
}

func (s *Service) requestToken(ctx context.Context) (string, error) {
	var parsed tokenResponse

	baseURL := strings.TrimRight(strings.TrimSpace(s.cfg.URL), "/")
	if baseURL == "" {
		return "", errors.New("zentao.url is empty")
	}
	account := strings.TrimSpace(s.cfg.Account)
	password := strings.TrimSpace(s.cfg.Password)
	if account == "" || password == "" {
		return "", errors.New("zentao.account or zentao.password is empty")
	}

	body, err := json.Marshal(tokenRequest{
		Account:  account,
		Password: password,
	})
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+zentaoTokensRoute,
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("token api status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.Token) == "" {
		return "", errors.New("token api returns empty token")
	}
	return parsed.Token, nil
}

func (s *Service) fetchProducts(ctx context.Context, token string) ([]byte, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(s.cfg.URL), "/")
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		baseURL+zentaoProductsRoute,
		nil,
	)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Token", token)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("products api status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}
