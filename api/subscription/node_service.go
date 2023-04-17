package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/sirupsen/logrus"
)

type NodeService interface {
	Subscribe(ctx context.Context) (_ <-chan model.NodeChangedEvent, err error)
}

type nodeEvent struct {
	Node struct {
		Id struct {
			Kind string `json:"Kind"`
		} `json:"id"`
	} `json:"node"`
}

type nodeService struct {
	subscription Subscription
	path         string
}

func NewNodeService(subscription Subscription) NodeService {
	return &nodeService{
		subscription: subscription,
		path:         "node",
	}
}

func (s *nodeService) Subscribe(ctx context.Context) (_ <-chan model.NodeChangedEvent, err error) {
	bytesChannel, err := s.subscription.Subscribe(ctx, path.Join(s.path, "*"))
	if err != nil {
		return nil, err
	}

	observerChan := make(chan model.NodeChangedEvent)
	go func() {
		defer close(observerChan)

		for bytes := range bytesChannel {
			nodeChangedEvent, err := unmarshalEvent(bytes)
			if err != nil {
				logrus.WithError(err).Error("failed to process node changed event")
				return
			}
			observerChan <- nodeChangedEvent
		}
	}()

	return observerChan, nil
}

func unmarshalEvent(bytes []byte) (model.NodeChangedEvent, error) {
	var ne nodeEvent
	if err := json.Unmarshal(bytes, &ne); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node event: %w", err)
	}

	switch ne.Node.Id.Kind {
	case model.IdKindUser.String():
		var userChangedEvent *model.UserChangedEvent
		err := json.Unmarshal(bytes, &userChangedEvent)
		return userChangedEvent, err
	case model.IdKindServer.String():
		var serverChangedEvent *model.ServerChangedEvent
		err := json.Unmarshal(bytes, &serverChangedEvent)
		return serverChangedEvent, err
	case model.IdKindPeer.String():
		var peerChangedEvent *model.PeerChangedEvent
		err := json.Unmarshal(bytes, &peerChangedEvent)
		return peerChangedEvent, err
	default:
		return nil, fmt.Errorf("unhandled node kind: %s", ne.Node.Id.Kind)
	}
}
