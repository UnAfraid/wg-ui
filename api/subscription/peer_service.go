package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/sirupsen/logrus"
)

type peerService struct {
	subscription Subscription
	path         string
}

func NewPeerService(subscription Subscription) Service[*model.PeerChangedEvent] {
	return &peerService{
		subscription: subscription,
		path:         path.Join("node", model.IdKindPeer.String()),
	}
}

func (s *peerService) Notify(peerChanged *model.PeerChangedEvent) error {
	bytes, err := json.Marshal(peerChanged)
	if err != nil {
		return err
	}

	if err := s.subscription.Notify(bytes, path.Join(s.path, peerChanged.Node.ID.Base64())); err != nil {
		return fmt.Errorf("failed to notify peerChanged: %w", err)
	}
	return nil
}

func (s *peerService) Subscribe(ctx context.Context) (_ <-chan *model.PeerChangedEvent, err error) {
	bytesChannel, err := s.subscription.Subscribe(ctx, path.Join(s.path, "*"))
	if err != nil {
		return nil, err
	}

	observerChan := make(chan *model.PeerChangedEvent)
	go func() {
		defer close(observerChan)

		for bytes := range bytesChannel {
			if err = notify[*model.PeerChangedEvent](bytes, observerChan); err != nil {
				logrus.WithError(err).Error("failed to notify peer changed")
				return
			}
		}
	}()

	return observerChan, nil
}
