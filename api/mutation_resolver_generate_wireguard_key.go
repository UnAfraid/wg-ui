package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func (r *mutationResolver) GenerateWireguardKey(ctx context.Context, input model.GenerateWireguardKeyInput) (*model.GenerateWireguardKeyPayload, error) {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	return &model.GenerateWireguardKeyPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		PrivateKey:       key.String(),
		PublicKey:        key.PublicKey().String(),
	}, nil
}
