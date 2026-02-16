package foreign

import (
	"context"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/handler"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/model"
)

type foreignServerResolver struct{}

func NewForeignServerResolver() *foreignServerResolver {
	return &foreignServerResolver{}
}

func (r *foreignServerResolver) Backend(ctx context.Context, obj *model.ForeignServer) (*model.Backend, error) {
	backendId, err := obj.Backend.ID.String(model.IdKindBackend)
	if err != nil {
		return nil, err
	}

	loader, err := handler.BackendLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return loader.Load(ctx, backendId)()
}
