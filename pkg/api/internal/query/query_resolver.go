package query

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/handler"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/model"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/pkg/backend"
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/manage"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/server"
	"github.com/UnAfraid/wg-ui/pkg/user"
	wgbackend "github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

type queryResolver struct {
	peerService    peer.Service
	serverService  server.Service
	userService    user.Service
	backendService backend.Service
	manageService  manage.Service
}

func NewQueryResolver(
	peerService peer.Service,
	serverService server.Service,
	userService user.Service,
	backendService backend.Service,
	manageService manage.Service,
) resolver.QueryResolver {
	return &queryResolver{
		peerService:    peerService,
		serverService:  serverService,
		userService:    userService,
		backendService: backendService,
		manageService:  manageService,
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
	case model.IdKindBackend:
		backendLoader, err := handler.BackendLoaderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return backendLoader.Load(ctx, id.Value)()
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

func (r *queryResolver) AvailableBackends(ctx context.Context) ([]*model.AvailableBackend, error) {
	registeredTypes, err := r.backendService.RegisteredTypes(ctx)
	if err != nil {
		return nil, err
	}

	allTypes := wgbackend.ListTypes()
	return adapt.Array(allTypes, func(t string) *model.AvailableBackend {
		return &model.AvailableBackend{
			Type:       t,
			Supported:  wgbackend.IsSupported(t),
			Registered: slices.Contains(registeredTypes, t),
		}
	}), nil
}

func (r *queryResolver) Backends(ctx context.Context, typeArg *string) ([]*model.Backend, error) {
	backends, err := r.backendService.FindBackends(ctx, &backend.FindOptions{
		Type: typeArg,
	})
	if err != nil {
		return nil, err
	}
	return adapt.Array(backends, model.ToBackend), nil
}

func (r *queryResolver) ForeignServers(ctx context.Context) ([]*model.ForeignServer, error) {
	// Get all backends and collect foreign servers from each
	backends, err := r.backendService.FindBackends(ctx, &backend.FindOptions{
		Enabled: adapt.ToPointer(true),
	})
	if err != nil {
		return nil, err
	}

	var allForeignServers []*model.ForeignServer
	var errs []error
	for _, b := range backends {
		foreignServers, err := r.manageService.ForeignServers(ctx, b.Id)
		if err != nil {
			errs = append(errs, fmt.Errorf("backend %s: %w", b.Name, err))
			continue
		}
		for _, fs := range foreignServers {
			allForeignServers = append(allForeignServers, model.ToForeignServer(fs))
		}
	}

	if len(errs) > 0 && len(allForeignServers) == 0 {
		return nil, errors.Join(errs...)
	}

	return allForeignServers, nil
}
