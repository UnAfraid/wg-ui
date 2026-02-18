package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/graph-gophers/dataloader/v7"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/model"
	"github.com/UnAfraid/wg-ui/pkg/backend"
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/server"
	"github.com/UnAfraid/wg-ui/pkg/user"
)

var (
	userLoaderCtxKey    = &contextKey{"userLoader"}
	serverLoaderCtxKey  = &contextKey{"serverLoader"}
	peerLoaderCtxKey    = &contextKey{"peerLoader"}
	backendLoaderCtxKey = &contextKey{"backendLoader"}
)

func NewDataLoaderMiddleware(
	wait time.Duration,
	maxBatch int,
	userService user.Service,
	serverService server.Service,
	peerService peer.Service,
	backendService backend.Service,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ctx = context.WithValue(ctx, userLoaderCtxKey, newBatchedLoader(userBatchFn(userService), wait, maxBatch))
			ctx = context.WithValue(ctx, serverLoaderCtxKey, newBatchedLoader(serverBatchFn(serverService), wait, maxBatch))
			ctx = context.WithValue(ctx, peerLoaderCtxKey, newBatchedLoader(peerBatchFn(peerService), wait, maxBatch))
			ctx = context.WithValue(ctx, backendLoaderCtxKey, newBatchedLoader(backendBatchFn(backendService), wait, maxBatch))

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

func BackendLoaderFromContext(ctx context.Context) (*dataloader.Loader[string, *model.Backend], error) {
	return dataLoaderFromContext[string, *model.Backend](ctx, backendLoaderCtxKey)
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
		return resultAndErrorToDataloaderResult(ids, adapt.Array(users, model.ToUser), func(item *model.User) string {
			if item == nil {
				return ""
			}
			return item.ID.Value
		}, err)
	}
}

func serverBatchFn(serverService server.Service) func(context.Context, []string) []*dataloader.Result[*model.Server] {
	return func(ctx context.Context, ids []string) []*dataloader.Result[*model.Server] {
		servers, err := serverService.FindServers(ctx, &server.FindOptions{
			Ids: ids,
		})
		return resultAndErrorToDataloaderResult(ids, adapt.Array(servers, model.ToServer), func(item *model.Server) string {
			if item == nil {
				return ""
			}
			return item.ID.Value
		}, err)
	}
}

func peerBatchFn(peerService peer.Service) func(context.Context, []string) []*dataloader.Result[*model.Peer] {
	return func(ctx context.Context, ids []string) []*dataloader.Result[*model.Peer] {
		peers, err := peerService.FindPeers(ctx, &peer.FindOptions{
			Ids: ids,
		})
		return resultAndErrorToDataloaderResult(ids, adapt.Array(peers, model.ToPeer), func(item *model.Peer) string {
			if item == nil {
				return ""
			}
			return item.ID.Value
		}, err)
	}
}

func backendBatchFn(backendService backend.Service) func(context.Context, []string) []*dataloader.Result[*model.Backend] {
	return func(ctx context.Context, ids []string) []*dataloader.Result[*model.Backend] {
		backends, err := backendService.FindBackends(ctx, &backend.FindOptions{
			Ids: ids,
		})
		return resultAndErrorToDataloaderResult(ids, adapt.Array(backends, model.ToBackend), func(item *model.Backend) string {
			if item == nil {
				return ""
			}
			return item.ID.Value
		}, err)
	}
}

func newBatchedLoader[K comparable, V any](batchFn func(context.Context, []K) []*dataloader.Result[V], wait time.Duration, maxBatch int) *dataloader.Loader[K, V] {
	return dataloader.NewBatchedLoader(batchFn, dataloader.WithWait[K, V](wait), dataloader.WithInputCapacity[K, V](maxBatch))
}

func resultAndErrorToDataloaderResult[K comparable, T any](
	keys []K,
	values []T,
	keyFn func(T) K,
	err error,
) []*dataloader.Result[T] {
	if err != nil {
		result := make([]*dataloader.Result[T], len(keys))
		for i := 0; i < len(keys); i++ {
			result[i] = &dataloader.Result[T]{
				Error: err,
			}
		}
		return result
	}

	valuesByKey := make(map[K]T, len(values))
	for _, value := range values {
		valuesByKey[keyFn(value)] = value
	}

	result := make([]*dataloader.Result[T], len(keys))
	for i, key := range keys {
		value, ok := valuesByKey[key]
		if ok {
			result[i] = &dataloader.Result[T]{Data: value}
			continue
		}
		var zero T
		result[i] = &dataloader.Result[T]{Data: zero}
	}

	return result
}
