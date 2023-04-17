package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/sirupsen/logrus"
)

type serverService struct {
	subscription Subscription
	path         string
}

func NewServerService(subscription Subscription) Service[*model.ServerChangedEvent] {
	return &serverService{
		subscription: subscription,
		path:         path.Join("node", model.IdKindServer.String()),
	}
}

func (s *serverService) Notify(serverChanged *model.ServerChangedEvent) error {
	bytes, err := json.Marshal(serverChanged)
	if err != nil {
		return err
	}

	if err := s.subscription.Notify(bytes, path.Join(s.path, serverChanged.Node.ID.Base64())); err != nil {
		return fmt.Errorf("failed to notify serverChanged: %w", err)
	}
	return nil
}

func (s *serverService) Subscribe(ctx context.Context) (_ <-chan *model.ServerChangedEvent, err error) {
	bytesChannel, err := s.subscription.Subscribe(ctx, path.Join(s.path, "*"))
	if err != nil {
		return nil, err
	}

	observerChan := make(chan *model.ServerChangedEvent)
	go func() {
		defer close(observerChan)

		for bytes := range bytesChannel {
			if err = notify[*model.ServerChangedEvent](bytes, observerChan); err != nil {
				logrus.WithError(err).Error("failed to notify")
				return
			}
		}
	}()

	return observerChan, nil
}
