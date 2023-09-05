package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/UnAfraid/wg-ui/api/internal/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/user"
	"github.com/graph-gophers/dataloader/v7"
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
	dataLoader, ok := ctx.Value(userLoaderCtxKey).(*dataloader.Loader[string, *model.User])
	if !ok {
		return nil, errors.New("user loader not found")
	}
	return dataLoader, nil
}

func ServerLoaderFromContext(ctx context.Context) (*dataloader.Loader[string, *model.Server], error) {
	dataLoader, ok := ctx.Value(serverLoaderCtxKey).(*dataloader.Loader[string, *model.Server])
	if !ok {
		return nil, errors.New("server loader not found")
	}
	return dataLoader, nil
}

func PeerLoaderFromContext(ctx context.Context) (*dataloader.Loader[string, *model.Peer], error) {
	dataLoader, ok := ctx.Value(peerLoaderCtxKey).(*dataloader.Loader[string, *model.Peer])
	if !ok {
		return nil, errors.New("peer loader not found")
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
