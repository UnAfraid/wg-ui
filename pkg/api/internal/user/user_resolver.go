package user

import (
	"context"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/model"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/server"
)

type userResolver struct {
	serverService server.Service
	peerService   peer.Service
}

func NewUserResolver(
	serverService server.Service,
	peerService peer.Service,
) resolver.UserResolver {
	return &userResolver{
		serverService: serverService,
		peerService:   peerService,
	}
}

func (r *userResolver) Servers(ctx context.Context, u *model.User) ([]*model.Server, error) {
	userId, err := u.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	servers, err := r.serverService.FindServers(ctx, &server.FindOptions{
		CreateUserId: &userId,
	})
	if err != nil {
		return nil, err
	}

	return adapt.Array(servers, model.ToServer), nil
}

func (r *userResolver) Peers(ctx context.Context, u *model.User) ([]*model.Peer, error) {
	userId, err := u.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	peers, err := r.peerService.FindPeers(ctx, &peer.FindOptions{
		CreateUserId: &userId,
	})
	if err != nil {
		return nil, err
	}

	return adapt.Array(peers, model.ToPeer), nil
}
