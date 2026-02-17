package backend

import (
	"context"

	"github.com/UnAfraid/wg-ui/pkg/server"
)

// serverCounterAdapter adapts server.Repository to ServerCounter interface
type serverCounterAdapter struct {
	serverRepository server.Repository
}

// NewServerCounter creates a ServerCounter from a server.Repository
func NewServerCounter(serverRepository server.Repository) ServerCounter {
	return &serverCounterAdapter{
		serverRepository: serverRepository,
	}
}

func (a *serverCounterAdapter) CountServersByBackendId(ctx context.Context, backendId string) (int, error) {
	return a.serverRepository.CountByBackendId(ctx, backendId)
}

func (a *serverCounterAdapter) CountEnabledServersByBackendId(ctx context.Context, backendId string) (int, error) {
	return a.serverRepository.CountEnabledByBackendId(ctx, backendId)
}
