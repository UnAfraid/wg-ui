package query

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/graph-gophers/dataloader/v7"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/handler"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/model"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
)

func idsToStringIds(idKind model.IdKind, ids []*model.ID) ([]string, error) {
	return adapt.ArrayErr(ids, func(id *model.ID) (string, error) {
		return id.String(idKind)
	})
}

func assignNodeToNodes(ids []*model.ID, nodes []model.Node, n model.Node) {
	if n == nil {
		return
	}
	for i, id := range ids {
		if n.GetID().Equal(*id) {
			nodes[i] = n
		}
	}
}

type NodeResult struct {
	idKind model.IdKind
	nodes  []model.Node
	err    error
}

func nodesWorker(ctx context.Context, idKind model.IdKind, ids []*model.ID, nodeResultChan chan *NodeResult, wg *sync.WaitGroup) {
	defer wg.Done()

	nodes, err := resolveIdKindNodes(ctx, idKind, ids)
	if err != nil {
		nodeResultChan <- &NodeResult{
			idKind: idKind,
			err:    err,
		}
		return
	}

	nodeResultChan <- &NodeResult{
		idKind: idKind,
		nodes:  nodes,
	}
}

func resolveIdKindNodes(ctx context.Context, idKind model.IdKind, ids []*model.ID) ([]model.Node, error) {
	stringIds, err := idsToStringIds(idKind, ids)
	if err != nil {
		return nil, err
	}

	switch idKind {
	case model.IdKindUser:
		return resolveNodes(ctx, stringIds, handler.UserLoaderFromContext)
	case model.IdKindServer:
		return resolveNodes(ctx, stringIds, handler.ServerLoaderFromContext)
	case model.IdKindPeer:
		return resolveNodes(ctx, stringIds, handler.PeerLoaderFromContext)
	default:
		return nil, fmt.Errorf("node type %s is %w", idKind, resolver.ErrNotImplemented)
	}
}

func resolveNodes[K comparable, V model.Node](
	ctx context.Context,
	ids []K,
	dataLoaderInitFn func(ctx context.Context) (*dataloader.Loader[K, V], error),
) ([]model.Node, error) {
	dataLoader, err := dataLoaderInitFn(ctx)
	if err != nil {
		return nil, err
	}

	peers, errs := dataLoader.LoadMany(ctx, ids)()
	if err = errors.Join(errs...); err != nil {
		return nil, err
	}

	return adapt.Array(peers, func(node V) model.Node {
		return node
	}), nil
}
