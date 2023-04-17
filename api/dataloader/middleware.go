package dataloader

import (
	"context"
	"net/http"
	"time"

	"github.com/UnAfraid/dataloaden/v2/dataloader"
	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/user"
)

func NewMiddleware(
	wait time.Duration,
	maxBatch int,
	userService user.Service,
	serverService server.Service,
	peerService peer.Service,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ctx = context.WithValue(ctx, userLoaderCtxKey, NewUserLoader(dataloader.Config[string, *model.User]{
				Fetch:    userFetcher(ctx, userService),
				Wait:     wait,
				MaxBatch: maxBatch,
			}))

			ctx = context.WithValue(ctx, serverLoaderCtxKey, NewServerLoader(dataloader.Config[string, *model.Server]{
				Fetch:    serverFetcher(ctx, serverService),
				Wait:     wait,
				MaxBatch: maxBatch,
			}))

			ctx = context.WithValue(ctx, peerLoaderCtxKey, NewPeerLoader(dataloader.Config[string, *model.Peer]{
				Fetch:    peerFetcher(ctx, peerService),
				Wait:     wait,
				MaxBatch: maxBatch,
			}))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
