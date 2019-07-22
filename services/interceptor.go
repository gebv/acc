package services

import (
	"context"
)

type ctxKey int

const (
	loggerKey ctxKey = iota
)

// InterceptSession обработчик контекста (запроса) на предмет сессии
// На основе метода (справочник) определяем требования к сессии
// В результате работы обработчика всегда в контекст кладется сессия
func (s *Service) InterceptSession(ctx context.Context, methodName string, ip string) (context.Context, error) {

	return ctx, nil
}
