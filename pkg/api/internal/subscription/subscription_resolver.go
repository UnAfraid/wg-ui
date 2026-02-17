package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/model"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/pkg/backend"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/server"
	"github.com/UnAfraid/wg-ui/pkg/user"
)

const totalSubscriptionNodeSources = 3

type subscriptionResolver struct {
	userService    user.Service
	serverService  server.Service
	peerService    peer.Service
	backendService backend.Service
}

func NewSubscriptionResolver(
	userService user.Service,
	serverService server.Service,
	peerService peer.Service,
	backendService backend.Service,
) resolver.SubscriptionResolver {
	return &subscriptionResolver{
		userService:    userService,
		serverService:  serverService,
		peerService:    peerService,
		backendService: backendService,
	}
}

func (r *subscriptionResolver) BackendChanged(ctx context.Context, id *model.ID) (<-chan *model.BackendChangedEvent, error) {
	var backendId string
	if id != nil {
		var err error
		backendId, err = id.String(model.IdKindBackend)
		if err != nil {
			return nil, err
		}
	}

	return domainEventToApiEvent[*backend.ChangedEvent, *model.BackendChangedEvent](ctx, r.backendService, func(event *backend.ChangedEvent) *model.BackendChangedEvent {
		if backendId != "" && event.Backend.Id != backendId {
			return nil // Filter out events for other backends
		}
		return &model.BackendChangedEvent{
			Backend: model.ToBackend(event.Backend),
			Action:  event.Action,
		}
	})
}

func (r *subscriptionResolver) UserChanged(ctx context.Context) (<-chan *model.UserChangedEvent, error) {
	return domainEventToApiEvent[*user.ChangedEvent, *model.UserChangedEvent](ctx, r.userService, func(event *user.ChangedEvent) *model.UserChangedEvent {
		return &model.UserChangedEvent{
			Node:   model.ToUser(event.User),
			Action: event.Action,
		}
	})
}

func (r *subscriptionResolver) ServerChanged(ctx context.Context) (<-chan *model.ServerChangedEvent, error) {
	return domainEventToApiEvent[*server.ChangedEvent, *model.ServerChangedEvent](ctx, r.serverService, func(event *server.ChangedEvent) *model.ServerChangedEvent {
		return &model.ServerChangedEvent{
			Node:   model.ToServer(event.Server),
			Action: event.Action,
		}
	})
}

func (r *subscriptionResolver) PeerChanged(ctx context.Context) (<-chan *model.PeerChangedEvent, error) {
	return domainEventToApiEvent[*peer.ChangedEvent, *model.PeerChangedEvent](ctx, r.peerService, func(event *peer.ChangedEvent) *model.PeerChangedEvent {
		return &model.PeerChangedEvent{
			Node:   model.ToPeer(event.Peer),
			Action: event.Action,
		}
	})
}

func (r *subscriptionResolver) NodeChanged(ctx context.Context) (<-chan model.NodeChangedEvent, error) {
	userEvents, err := r.userService.Subscribe(ctx)
	if err != nil {
		return nil, err
	}

	serverEvents, err := r.serverService.Subscribe(ctx)
	if err != nil {
		return nil, err
	}

	peerEvents, err := r.peerService.Subscribe(ctx)
	if err != nil {
		return nil, err
	}

	nodeChangedEvents := make(chan model.NodeChangedEvent)
	go func() {
		defer close(nodeChangedEvents)

		sourcesAvailable := totalSubscriptionNodeSources
		for {
			select {
			case userEvent := <-userEvents:
				if userEvent == nil {
					sourcesAvailable--
				} else {
					nodeChangedEvents <- model.UserChangedEvent{
						Node:   model.ToUser(userEvent.User),
						Action: userEvent.Action,
					}
				}
			case serverEvent := <-serverEvents:
				if serverEvent == nil {
					sourcesAvailable--
				} else {
					nodeChangedEvents <- model.ServerChangedEvent{
						Node:   model.ToServer(serverEvent.Server),
						Action: serverEvent.Action,
					}
				}
			case peerEvent := <-peerEvents:
				if peerEvent == nil {
					sourcesAvailable--
				} else {
					nodeChangedEvents <- model.PeerChangedEvent{
						Node:   model.ToPeer(peerEvent.Peer),
						Action: peerEvent.Action,
					}
				}
			}

			if sourcesAvailable <= 0 {
				return
			}
		}
	}()

	return nodeChangedEvents, nil
}

type Subscribe[T any] interface {
	Subscribe(ctx context.Context) (<-chan T, error)
}

func domainEventToApiEvent[FromType, ToType any](
	ctx context.Context,
	subscribe Subscribe[FromType],
	fromToAdaptFn func(FromType) ToType,
) (<-chan ToType, error) {
	domainEvents, err := subscribe.Subscribe(ctx)
	if err != nil {
		return nil, err
	}

	apiEvents := make(chan ToType)
	go func() {
		defer close(apiEvents)

		for event := range domainEvents {
			apiEvent := fromToAdaptFn(event)
			// Skip nil events (filtered out by the adapter function)
			if any(apiEvent) == nil {
				continue
			}
			apiEvents <- apiEvent
		}
	}()

	return apiEvents, err
}
