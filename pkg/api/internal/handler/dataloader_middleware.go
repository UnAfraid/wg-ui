package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/graph-gophers/dataloader/v7"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/model"
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/server"
	"github.com/UnAfraid/wg-ui/pkg/user"
)

var (
	userLoaderCtxKey   = &contextKey{"userLoader"}
	serverLoaderCtxKey = &contextKey{"serverLoader"}
	peerLoaderCtxKey   = &contextKey{"peerLoader"}
)

func NewDataLoaderMiddleware(
	wait time.Duration,
	maxBatch int,
	userService user.Service,
	serverService server.Service,
	peerService peer.Service,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ctx = context.WithValue(ctx, userLoaderCtxKey, newBatchedLoader(userBatchFn(userService), wait, maxBatch))
			ctx = context.WithValue(ctx, serverLoaderCtxKey, newBatchedLoader(serverBatchFn(serverService), wait, maxBatch))
			ctx = context.WithValue(ctx, peerLoaderCtxKey, newBatchedLoader(peerBatchFn(peerService), wait, maxBatch))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserLoaderFromContext(ctx context.Context) (*dataloader.Loader[string, *model.User], error) {
	return dataLoaderFromContext[string, *model.User](ctx, userLoaderCtxKey)
}

func ServerLoaderFromContext(ctx context.Context) (*dataloader.Loader[string, *model.Server], error) {
	return dataLoaderFromContext[string, *model.Server](ctx, serverLoaderCtxKey)
}

func PeerLoaderFromContext(ctx context.Context) (*dataloader.Loader[string, *model.Peer], error) {
	return dataLoaderFromContext[string, *model.Peer](ctx, peerLoaderCtxKey)
}

func dataLoaderFromContext[K comparable, T any](ctx context.Context, contextKey *contextKey) (*dataloader.Loader[K, T], error) {
	dataLoader, ok := ctx.Value(contextKey).(*dataloader.Loader[K, T])
	if !ok {
		var nodeType T
		return nil, fmt.Errorf("%T data loader not found", nodeType)
	}
	return dataLoader, nil
}

func userBatchFn(userService user.Service) func(context.Context, []string) []*dataloader.Result[*model.User] {
	return func(ctx context.Context, ids []string) []*dataloader.Result[*model.User] {
		users, err := userService.FindUsers(ctx, &user.FindOptions{
			Ids: ids,
		})
		return resultAndErrorToDataloaderResult(len(ids), adapt.Array(users, model.ToUser), err)
	}
}

func serverBatchFn(serverService server.Service) func(context.Context, []string) []*dataloader.Result[*model.Server] {
	return func(ctx context.Context, ids []string) []*dataloader.Result[*model.Server] {
		servers, err := serverService.FindServers(ctx, &server.FindOptions{
			Ids: ids,
		})
		return resultAndErrorToDataloaderResult(len(ids), adapt.Array(servers, model.ToServer), err)
	}
}

func peerBatchFn(peerService peer.Service) func(context.Context, []string) []*dataloader.Result[*model.Peer] {
	return func(ctx context.Context, ids []string) []*dataloader.Result[*model.Peer] {
		peers, err := peerService.FindPeers(ctx, &peer.FindOptions{
			Ids: ids,
		})
		return resultAndErrorToDataloaderResult(len(ids), adapt.Array(peers, model.ToPeer), err)
	}
}

func newBatchedLoader[K comparable, V any](batchFn func(context.Context, []K) []*dataloader.Result[V], wait time.Duration, maxBatch int) *dataloader.Loader[K, V] {
	return dataloader.NewBatchedLoader(batchFn, dataloader.WithWait[K, V](wait), dataloader.WithInputCapacity[K, V](maxBatch))
}

func resultAndErrorToDataloaderResult[T any](length int, values []T, err error) []*dataloader.Result[T] {
	if err != nil {
		result := make([]*dataloader.Result[T], length)
		for i := 0; i < length; i++ {
			result[i] = &dataloader.Result[T]{
				Error: err,
			}
		}
		return result
	}

	return adapt.Array(values, func(value T) *dataloader.Result[T] {
		return &dataloader.Result[T]{
			Data: value,
		}
	})
}
