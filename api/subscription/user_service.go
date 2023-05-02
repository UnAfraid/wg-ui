package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/sirupsen/logrus"
)

type userService struct {
	subscription Subscription
	path         string
}

func NewUserService(subscription Subscription) Service[*model.UserChangedEvent] {
	return &userService{
		subscription: subscription,
		path:         path.Join("node", model.IdKindUser.String()),
	}
}

func (s *userService) Notify(userChanged *model.UserChangedEvent) error {
	bytes, err := json.Marshal(userChanged)
	if err != nil {
		return err
	}

	if err := s.subscription.Notify(bytes, path.Join(s.path, userChanged.Node.ID.Base64())); err != nil {
		return fmt.Errorf("failed to notify userChanged: %w", err)
	}
	return nil
}

func (s *userService) Subscribe(ctx context.Context) (_ <-chan *model.UserChangedEvent, err error) {
	bytesChannel, err := s.subscription.Subscribe(ctx, path.Join(s.path, "*"))
	if err != nil {
		return nil, err
	}

	observerChan := make(chan *model.UserChangedEvent)
	go func() {
		defer close(observerChan)

		for bytes := range bytesChannel {
			if err = notify[*model.UserChangedEvent](bytes, observerChan); err != nil {
				logrus.WithError(err).Error("failed to notify user changed")
				return
			}
		}
	}()

	return observerChan, nil
}

func (s *userService) HasSubscribers() bool {
	return s.subscription.HasSubscribers(path.Join(s.path, "*"))
}
