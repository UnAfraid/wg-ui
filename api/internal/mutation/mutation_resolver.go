package mutation

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/internal/model"
	"github.com/UnAfraid/wg-ui/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/auth"
	"github.com/UnAfraid/wg-ui/manage"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type mutationResolver struct {
	authService   auth.Service
	manageService manage.Service
}

func NewMutationResolver(
	authService auth.Service,
	manageService manage.Service,
) resolver.MutationResolver {
	return &mutationResolver{
		authService:   authService,
		manageService: manageService,
	}
}

func (r *mutationResolver) SignIn(ctx context.Context, input model.SignInInput) (*model.SignInPayload, error) {
	u, err := r.manageService.Authenticate(ctx, input.Email, input.Password)
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
	createdUser, err := r.manageService.CreateUser(ctx, model.CreateUserInputToUserCreateUserOptions(input))
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

	updatedUser, err := r.manageService.UpdateUser(ctx, userId, updateOptions, updateFieldMask)
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

	deletedUser, err := r.manageService.DeleteUser(ctx, userId)
	if err != nil {
		return nil, err
	}

	return &model.DeleteUserPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		User:             model.ToUser(deletedUser),
	}, nil
}

func (r *mutationResolver) GenerateWireguardKey(_ context.Context, input model.GenerateWireguardKeyInput) (*model.GenerateWireguardKeyPayload, error) {
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

	createdServer, err := r.manageService.CreateServer(ctx, createOptions, userId)
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

	updatedServer, err := r.manageService.UpdateServer(ctx, serverId, updateOptions, updateFieldMask, userId)
	if err != nil {
		return nil, err
	}

	return &model.UpdateServerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Server:           model.ToServer(updatedServer),
	}, nil
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

	deletedServer, err := r.manageService.DeleteServer(ctx, serverId, userId)
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

	srv, err := r.manageService.StartServer(ctx, serverId)
	if err != nil {
		return nil, err
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

	srv, err := r.manageService.StopServer(ctx, serverId)
	if err != nil {
		return nil, err
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

	peer, err := r.manageService.CreatePeer(ctx, serverId, model.CreatePeerInputToCreateOptions(input), userId)
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
	peer, err := r.manageService.UpdatePeer(ctx, peerId, updateOptions, updateFieldMask, userId)
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

	peer, err := r.manageService.DeletePeer(ctx, peerId, userId)
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

	server, err := r.manageService.ImportForeignServer(ctx, input.Name, userId)
	if err != nil {
		return nil, err
	}

	return &model.ImportForeignServerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Server:           model.ToServer(server),
	}, nil
}
