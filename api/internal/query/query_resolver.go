package query

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/UnAfraid/wg-ui/api/internal/handler"
	"github.com/UnAfraid/wg-ui/api/internal/model"
	"github.com/UnAfraid/wg-ui/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/user"
	"github.com/UnAfraid/wg-ui/wg"
)

type queryResolver struct {
	wgService     wg.Service
	peerService   peer.Service
	serverService server.Service
	userService   user.Service
}

func NewQueryResolver(
	wgService wg.Service,
	peerService peer.Service,
	serverService server.Service,
	userService user.Service,
) resolver.QueryResolver {
	return &queryResolver{
		wgService:     wgService,
		peerService:   peerService,
		serverService: serverService,
		userService:   userService,
	}
}

func (r *queryResolver) Viewer(ctx context.Context) (*model.User, error) {
	return model.ContextToUser(ctx)
}

func (r *queryResolver) Node(ctx context.Context, id model.ID) (model.Node, error) {
	switch id.Kind {
	case model.IdKindUser:
		userLoader, err := handler.UserLoaderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return userLoader.Load(ctx, id.Value)()
	case model.IdKindServer:
		serverLoader, err := handler.ServerLoaderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return serverLoader.Load(ctx, id.Value)()
	case model.IdKindPeer:
		peerLoader, err := handler.PeerLoaderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return peerLoader.Load(ctx, id.Value)()
	default:
		return nil, fmt.Errorf("node type %s is %w", id.Kind, resolver.ErrNotImplemented)
	}
}

func (r *queryResolver) Nodes(ctx context.Context, ids []*model.ID) ([]model.Node, error) {
	idsLen := len(ids)
	if idsLen == 0 {
		return nil, nil
	}

	idsByIDKind := make(map[model.IdKind][]*model.ID)
	for _, id := range ids {
		idsByIDKind[id.Kind] = append(idsByIDKind[id.Kind], id)
	}

	var waitGroup sync.WaitGroup
	nodeResultChan := make(chan *NodeResult)
	for idKind, ids := range idsByIDKind {
		waitGroup.Add(1)
		go nodesWorker(ctx, idKind, ids, nodeResultChan, &waitGroup)
	}
	go func() {
		waitGroup.Wait()
		close(nodeResultChan)
	}()

	var errs []error
	nodes := make([]model.Node, idsLen)
	for nodeResult := range nodeResultChan {
		if nodeResult.err != nil {
			errs = append(errs, nodeResult.err)
			continue
		}

		for _, n := range nodeResult.nodes {
			assignNodeToNodes(ids, nodes, n)
		}
	}
	if errs != nil {
		return nil, errors.Join(errs...)
	}
	return nodes, nil
}

func (r *queryResolver) Users(ctx context.Context, query *string) ([]*model.User, error) {
	users, err := r.userService.FindUsers(ctx, &user.FindOptions{
		Query: adapt.Dereference(query),
	})
	if err != nil {
		return nil, err
	}
	return adapt.Array(users, model.ToUser), nil
}

func (r *queryResolver) Servers(ctx context.Context, query *string, enabled *bool) ([]*model.Server, error) {
	servers, err := r.serverService.FindServers(ctx, &server.FindOptions{
		Query:   adapt.Dereference(query),
		Enabled: enabled,
	})
	if err != nil {
		return nil, err
	}
	return adapt.Array(servers, model.ToServer), nil
}

func (r *queryResolver) Peers(ctx context.Context, query *string) ([]*model.Peer, error) {
	servers, err := r.peerService.FindPeers(ctx, &peer.FindOptions{
		Query: adapt.Dereference(query),
	})
	if err != nil {
		return nil, err
	}
	return adapt.Array(servers, model.ToPeer), nil
}

func (r *queryResolver) ForeignServers(ctx context.Context) ([]*model.ForeignServer, error) {
	foreignServers, err := r.wgService.ForeignServers(ctx)
	if err != nil {
		return nil, err
	}
	return adapt.Array(foreignServers, model.ToForeignServer), nil
}
