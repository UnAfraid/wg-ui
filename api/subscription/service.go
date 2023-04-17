package subscription

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
)

type Service[T model.NodeChangedEvent] interface {
	Notify(category T) error
	Subscribe(ctx context.Context) (<-chan T, error)
}
