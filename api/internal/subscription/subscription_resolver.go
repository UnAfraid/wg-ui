package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/internal/model"
	"github.com/UnAfraid/wg-ui/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/user"
)

const totalSubscriptionNodeSources = 3

type subscriptionResolver struct {
	userService   user.Service
	serverService server.Service
	peerService   peer.Service
}

func NewSubscriptionResolver(
	userService user.Service,
	serverService server.Service,
	peerService peer.Service,
) resolver.SubscriptionResolver {
	return &subscriptionResolver{
		userService:   userService,
		serverService: serverService,
		peerService:   peerService,
	}
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
			apiEvents <- fromToAdaptFn(event)
		}
	}()

	return apiEvents, err
}
