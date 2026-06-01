package sqllog

import (
	"context"
	"sync"
	"time"
)

type ctxKeyRequestState struct{}

// RequestState 单次 HTTP 请求的 SQL 聚合状态。
type RequestState struct {
	RequestID string
	Method    string
	Route     string
	Start     time.Time

	mu       sync.Mutex
	seq      int
	sqlCount int
	hasError bool
	hasSlow  bool
}

// WithRequestState 将请求状态写入 context。
func WithRequestState(ctx context.Context, state *RequestState) context.Context {
	if state == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyRequestState{}, state)
}

// RequestStateFromContext 读取请求状态；不存在时返回 nil。
func RequestStateFromContext(ctx context.Context) *RequestState {
	if ctx == nil {
		return nil
	}
	v := ctx.Value(ctxKeyRequestState{})
	if v == nil {
		return nil
	}
	state, _ := v.(*RequestState)
	return state
}

func (s *RequestState) nextSeq() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	s.sqlCount++
	return s.seq
}

func (s *RequestState) markQuery(err error, slow bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.hasError = true
	}
	if slow {
		s.hasSlow = true
	}
}

func (s *RequestState) snapshot() (sqlCount int, hasError, hasSlow bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sqlCount, s.hasError, s.hasSlow
}
