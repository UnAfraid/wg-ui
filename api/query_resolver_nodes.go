package api

import (
	"context"
	"errors"
	"fmt"
	"sync"

	dataloader2 "github.com/UnAfraid/dataloaden/v2/dataloader"
	"github.com/UnAfraid/wg-ui/api/dataloader"
	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/hashicorp/go-multierror"
)

func (r *queryResolver) Nodes(ctx context.Context, ids []*model.ID) ([]model.Node, error) {
	idsLen := len(ids)
	if idsLen == 0 {
		return nil, nil
	}

	idsByIDKind := make(map[model.IdKind][]*model.ID)
	for _, id := range ids {
		idsByIDKind[id.Kind] = append(idsByIDKind[id.Kind], id)
	}

	var wg sync.WaitGroup
	nodeResultChan := make(chan *NodeResult)
	for idKind, ids := range idsByIDKind {
		wg.Add(1)
		go nodesWorker(ctx, idKind, ids, nodeResultChan, &wg)
	}
	go func() {
		wg.Wait()
		close(nodeResultChan)
	}()

	var err error
	nodes := make([]model.Node, idsLen)
	for nodeResult := range nodeResultChan {
		if nodeResult.err != nil {
			err = multierror.Append(err, nodeResult.err)
			continue
		}

		for _, n := range nodeResult.nodes {
			assignNodeToNodes(ids, nodes, n)
		}
	}
	if err != nil {
		return nil, err
	}
	return nodes, nil
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
		return resolveNodes(ctx, stringIds, dataloader.UserLoaderFromContext)
	case model.IdKindServer:
		return resolveNodes(ctx, stringIds, dataloader.ServerLoaderFromContext)
	case model.IdKindPeer:
		return resolveNodes(ctx, stringIds, dataloader.PeerLoaderFromContext)
	default:
		return nil, fmt.Errorf("node type %s is %w", idKind, ErrNotImplemented)
	}
}

func resolveNodes[K comparable, V model.Node](
	ctx context.Context,
	ids []K,
	dataLoaderInitFn func(ctx context.Context) (dataloader2.DataLoader[K, V], error),
) ([]model.Node, error) {
	dataLoader, err := dataLoaderInitFn(ctx)
	if err != nil {
		return nil, err
	}

	peers, errs := dataLoader.LoadAll(ids)
	if err = errors.Join(errs...); err != nil {
		return nil, err
	}

	return adapt.Array(peers, func(node V) model.Node {
		return node
	}), nil
}
