package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *serverResolver) InterfaceStats(_ context.Context, svc *model.Server) (*model.ServerInterfaceStats, error) {
	interfaceStats, err := r.wgService.InterfaceStats(svc.Name)
	if err != nil {
		return nil, err
	}
	return model.ToServerInterfaceStats(interfaceStats), nil
}
