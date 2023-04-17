package subscription

import (
	"context"
)

type Subscription interface {
	Notify(bytes []byte, channel string) error
	Subscribe(ctx context.Context, channel string) (<-chan []byte, error)
}
