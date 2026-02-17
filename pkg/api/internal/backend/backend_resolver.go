package backend

import (
	"context"
	"strings"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/handler"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/model"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/resolver"
	backendpkg "github.com/UnAfraid/wg-ui/pkg/backend"
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/manage"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/server"
	wireguardbackend "github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

type backendResolver struct {
	backendService backendpkg.Service
	serverService  server.Service
	peerService    peer.Service
	manageService  manage.Service
}

func NewBackendResolver(
	backendService backendpkg.Service,
	serverService server.Service,
	peerService peer.Service,
	manageService manage.Service,
) resolver.BackendResolver {
	return &backendResolver{
		backendService: backendService,
		serverService:  serverService,
		peerService:    peerService,
		manageService:  manageService,
	}
}

func (r *backendResolver) Supported(ctx context.Context, b *model.Backend) (bool, error) {
	backendId, err := b.ID.String(model.IdKindBackend)
	if err != nil {
		return false, err
	}

	backendLoader, err := handler.BackendLoaderFromContext(ctx)
	if err != nil {
		return false, err
	}

	backendEntity, err := backendLoader.Load(ctx, backendId)()
	if err != nil {
		return false, err
	}

	if backendEntity == nil {
		return false, nil
	}

	backendType := getBackendTypeFromURL(backendEntity.URL)
	return wireguardbackend.IsSupported(backendType), nil
}

func (r *backendResolver) Servers(ctx context.Context, b *model.Backend, query *string, enabled *bool) ([]*model.Server, error) {
	backendId, err := b.ID.String(model.IdKindBackend)
	if err != nil {
		return nil, err
	}

	servers, err := r.serverService.FindServers(ctx, &server.FindOptions{
		BackendId: &backendId,
		Query:     adapt.Dereference(query),
		Enabled:   enabled,
	})
	if err != nil {
		return nil, err
	}

	return adapt.Array(servers, model.ToServer), nil
}

func (r *backendResolver) Peers(ctx context.Context, b *model.Backend, query *string) ([]*model.Peer, error) {
	backendId, err := b.ID.String(model.IdKindBackend)
	if err != nil {
		return nil, err
	}

	// First, get servers on this backend
	servers, err := r.serverService.FindServers(ctx, &server.FindOptions{
		BackendId: &backendId,
	})
	if err != nil {
		return nil, err
	}

	if len(servers) == 0 {
		return nil, nil
	}

	serverIds := adapt.Array(servers, func(s *server.Server) string {
		return s.Id
	})

	peers, err := r.peerService.FindPeers(ctx, &peer.FindOptions{
		ServerIds: serverIds,
		Query:     adapt.Dereference(query),
	})
	if err != nil {
		return nil, err
	}

	return adapt.Array(peers, model.ToPeer), nil
}

func (r *backendResolver) ForeignServers(ctx context.Context, b *model.Backend) ([]*model.ForeignServer, error) {
	backendId, err := b.ID.String(model.IdKindBackend)
	if err != nil {
		return nil, err
	}

	foreignServers, err := r.manageService.ForeignServers(ctx, backendId)
	if err != nil {
		return nil, err
	}

	return adapt.Array(foreignServers, func(fs *wireguardbackend.ForeignServer) *model.ForeignServer {
		return model.ToForeignServer(fs, backendId)
	}), nil
}

func (r *backendResolver) CreateUser(ctx context.Context, b *model.Backend) (*model.User, error) {
	if b.CreateUser == nil {
		return nil, nil
	}

	userId, err := b.CreateUser.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	userLoader, err := handler.UserLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return userLoader.Load(ctx, userId)()
}

func (r *backendResolver) UpdateUser(ctx context.Context, b *model.Backend) (*model.User, error) {
	if b.UpdateUser == nil {
		return nil, nil
	}

	userId, err := b.UpdateUser.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	userLoader, err := handler.UserLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return userLoader.Load(ctx, userId)()
}

func (r *backendResolver) DeleteUser(ctx context.Context, b *model.Backend) (*model.User, error) {
	if b.DeleteUser == nil {
		return nil, nil
	}

	userId, err := b.DeleteUser.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	userLoader, err := handler.UserLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return userLoader.Load(ctx, userId)()
}

func getBackendTypeFromURL(url string) string {
	if idx := strings.Index(url, "://"); idx != -1 {
		return url[:idx]
	}
	return ""
}
