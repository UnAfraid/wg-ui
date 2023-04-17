package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *mutationResolver) ImportForeignServer(ctx context.Context, input model.ImportForeignServerInput) (*model.ImportForeignServerPayload, error) {
	user, err := model.ContextToUser(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := user.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	server, err := r.wgService.ImportForeignServer(ctx, input.Name, userId)
	if err != nil {
		return nil, err
	}

	return &model.ImportForeignServerPayload{
		ClientMutationID: input.ClientMutationID,
		Server:           model.ToServer(server),
	}, nil
}
