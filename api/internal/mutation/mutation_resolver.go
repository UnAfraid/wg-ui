package mutation

import (
	"context"
	"errors"

	"github.com/UnAfraid/wg-ui/api/internal/model"
	"github.com/UnAfraid/wg-ui/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/auth"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/user"
	"github.com/UnAfraid/wg-ui/wg"
	"github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type mutationResolver struct {
	authService   auth.Service
	userService   user.Service
	serverService server.Service
	peerService   peer.Service
	wgService     wg.Service
}

func NewMutationResolver(
	authService auth.Service,
	userService user.Service,
	serverService server.Service,
	peerService peer.Service,
	wgService wg.Service,
) resolver.MutationResolver {
	return &mutationResolver{
		authService:   authService,
		userService:   userService,
		serverService: serverService,
		peerService:   peerService,
		wgService:     wgService,
	}
}

func (r *mutationResolver) withServer(ctx context.Context, serverId string, callback func(svc *server.Server)) error {
	svc, err := r.serverService.FindServer(ctx, &server.FindOneOptions{
		IdOption: &server.IdOption{
			Id: serverId,
		},
		NameOption: nil,
	})
	if err != nil {
		return err
	}
	if svc != nil {
		callback(svc)
	}
	return nil
}

func (r *mutationResolver) SignIn(ctx context.Context, input model.SignInInput) (*model.SignInPayload, error) {
	u, err := r.userService.Authenticate(ctx, input.Email, input.Password)
	if err != nil {
		return nil, err
	}
	user := model.ToUser(u)

	tokenString, expiresIn, expiresAt, err := r.authService.Sign(user.ID.Base64())
	if err != nil {
		return nil, err
	}

	return &model.SignInPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Token:            tokenString,
		ExpiresAt:        expiresAt,
		ExpiresIn:        int(expiresIn.Seconds()),
	}, nil
}

func (r *mutationResolver) CreateUser(ctx context.Context, input model.CreateUserInput) (*model.CreateUserPayload, error) {
	createdUser, err := r.userService.CreateUser(ctx, model.CreateUserInputToUserCreateUserOptions(input))
	if err != nil {
		return nil, err
	}

	return &model.CreateUserPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		User:             model.ToUser(createdUser),
	}, nil
}

func (r *mutationResolver) UpdateUser(ctx context.Context, input model.UpdateUserInput) (*model.UpdateUserPayload, error) {
	updateOptions, updateFieldMask, err := model.UpdateUserInputToUserUpdateUserOptions(input)
	if err != nil {
		return nil, err
	}

	userId, err := input.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	updatedUser, err := r.userService.UpdateUser(ctx, userId, updateOptions, updateFieldMask)
	if err != nil {
		return nil, err
	}

	return &model.UpdateUserPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		User:             model.ToUser(updatedUser),
	}, nil
}

func (r *mutationResolver) DeleteUser(ctx context.Context, input model.DeleteUserInput) (*model.DeleteUserPayload, error) {
	userId, err := input.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	deletedUser, err := r.userService.DeleteUser(ctx, userId)
	if err != nil {
		return nil, err
	}

	return &model.DeleteUserPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		User:             model.ToUser(deletedUser),
	}, nil
}

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

func (r *mutationResolver) CreateServer(ctx context.Context, input model.CreateServerInput) (*model.CreateServerPayload, error) {
	user, err := model.ContextToUser(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := user.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	createOptions, err := model.CreateServerInputToCreateServerOptions(input)
	if err != nil {
		return nil, err
	}

	createdServer, err := r.serverService.CreateServer(ctx, createOptions, userId)
	if err != nil {
		return nil, err
	}

	return &model.CreateServerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Server:           model.ToServer(createdServer),
	}, nil
}

func (r *mutationResolver) UpdateServer(ctx context.Context, input model.UpdateServerInput) (*model.UpdateServerPayload, error) {
	user, err := model.ContextToUser(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := user.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	updateOptions, updateFieldMask, err := model.UpdateServerInputToUpdateOptionsAndUpdateFieldMask(input)
	if err != nil {
		return nil, err
	}

	serverId, err := input.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	updatedServer, err := r.serverService.UpdateServer(ctx, serverId, updateOptions, updateFieldMask, userId)
	if err != nil {
		return nil, err
	}

	var errs []error
	if updatedServer.Enabled && updateOptions.Running {
		peers, err := r.peerService.FindPeers(ctx, &peer.FindOptions{
			ServerId: &serverId,
		})
		if err != nil {
			errs = append(errs, err)
		} else {
			if err := r.wgService.ConfigureWireGuard(updatedServer.Name, updatedServer.PrivateKey, updatedServer.ListenPort, updatedServer.FirewallMark, peers); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return &model.UpdateServerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Server:           model.ToServer(updatedServer),
	}, errors.Join(errs...)
}

func (r *mutationResolver) DeleteServer(ctx context.Context, input model.DeleteServerInput) (*model.DeleteServerPayload, error) {
	user, err := model.ContextToUser(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := user.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	serverId, err := input.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	stoppedServer, err := r.wgService.StopServer(ctx, serverId)
	if err != nil {
		return nil, err
	}

	deletedServer, err := r.serverService.DeleteServer(ctx, stoppedServer.Id, userId)
	if err != nil {
		return nil, err
	}

	return &model.DeleteServerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Server:           model.ToServer(deletedServer),
	}, nil
}

func (r *mutationResolver) StartServer(ctx context.Context, input model.StartServerInput) (*model.StartServerPayload, error) {
	serverId, err := input.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	srv, err := r.wgService.StartServer(ctx, serverId)
	if err != nil {
		return nil, err
	}

	if err := srv.RunHooks(server.HookActionStart); err != nil {
		logrus.WithError(err).
			WithField("server", srv.Name).
			Error("failed to run hooks on server start")
	}

	return &model.StartServerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Server:           model.ToServer(srv),
	}, nil
}

func (r *mutationResolver) StopServer(ctx context.Context, input model.StopServerInput) (*model.StopServerPayload, error) {
	serverId, err := input.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	srv, err := r.wgService.StopServer(ctx, serverId)
	if err != nil {
		return nil, err
	}

	if err := srv.RunHooks(server.HookActionStop); err != nil {
		logrus.WithError(err).
			WithField("server", srv.Name).
			Error("failed to run hooks on server stop")
	}

	return &model.StopServerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Server:           model.ToServer(srv),
	}, nil
}

func (r *mutationResolver) CreatePeer(ctx context.Context, input model.CreatePeerInput) (*model.CreatePeerPayload, error) {
	user, err := model.ContextToUser(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := user.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	serverId, err := input.ServerID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	peer, err := r.peerService.CreatePeer(ctx, serverId, model.CreatePeerInputToCreateOptions(input), userId)
	if err != nil {
		return nil, err
	}

	err = r.withServer(ctx, peer.ServerId, func(svc *server.Server) {
		if svc.Enabled && svc.Running {
			err = r.wgService.AddPeer(svc.Name, svc.PrivateKey, svc.ListenPort, svc.FirewallMark, peer)
		}
	})
	if err != nil {
		return nil, err
	}

	return &model.CreatePeerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Peer:             model.ToPeer(peer),
	}, nil
}

func (r *mutationResolver) UpdatePeer(ctx context.Context, input model.UpdatePeerInput) (*model.UpdatePeerPayload, error) {
	user, err := model.ContextToUser(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := user.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	peerId, err := input.ID.String(model.IdKindPeer)
	if err != nil {
		return nil, err
	}

	updateOptions, updateFieldMask := model.UpdatePeerInputToUpdatePeerOptionsAndUpdatePeerFieldMask(input)
	peer, err := r.peerService.UpdatePeer(ctx, peerId, updateOptions, updateFieldMask, userId)
	if err != nil {
		return nil, err
	}

	err = r.withServer(ctx, peer.ServerId, func(svc *server.Server) {
		if svc.Enabled && svc.Running {
			err = r.wgService.UpdatePeer(svc.Name, svc.PrivateKey, svc.ListenPort, svc.FirewallMark, peer)
		}
	})
	if err != nil {
		return nil, err
	}

	return &model.UpdatePeerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Peer:             model.ToPeer(peer),
	}, nil
}

func (r *mutationResolver) DeletePeer(ctx context.Context, input model.DeletePeerInput) (*model.DeletePeerPayload, error) {
	user, err := model.ContextToUser(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := user.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	peerId, err := input.ID.String(model.IdKindPeer)
	if err != nil {
		return nil, err
	}

	peer, err := r.peerService.DeletePeer(ctx, peerId, userId)
	if err != nil {
		return nil, err
	}

	err = r.withServer(ctx, peer.ServerId, func(svc *server.Server) {
		if svc.Enabled && svc.Running {
			err = r.wgService.RemovePeer(svc.Name, svc.PrivateKey, svc.ListenPort, svc.FirewallMark, peer)
		}
	})
	if err != nil {
		return nil, err
	}

	return &model.DeletePeerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Peer:             model.ToPeer(peer),
	}, nil
}

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
		ClientMutationID: input.ClientMutationID.Value(),
		Server:           model.ToServer(server),
	}, nil
}
