package model

import (
	"github.com/UnAfraid/wg-ui/pkg/backend"
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
)

func CreateBackendInputToCreateOptions(input CreateBackendInput) *backend.CreateOptions {
	return &backend.CreateOptions{
		Name:        input.Name,
		Description: adapt.Dereference(input.Description.Value()),
		Url:         input.URL,
		Enabled:     adapt.Dereference(input.Enabled.Value()),
	}
}

func UpdateBackendInputToUpdateOptionsAndFieldMask(input UpdateBackendInput) (*backend.UpdateOptions, *backend.UpdateFieldMask) {
	fieldMask := &backend.UpdateFieldMask{
		Name:        input.Name.IsSet(),
		Description: input.Description.IsSet(),
		Url:         input.URL.IsSet(),
		Enabled:     input.Enabled.IsSet(),
	}

	var (
		name        string
		description string
		url         string
		enabled     bool
	)

	if fieldMask.Name {
		name = adapt.Dereference(input.Name.Value())
	}

	if fieldMask.Description {
		description = adapt.Dereference(input.Description.Value())
	}

	if fieldMask.Url {
		url = adapt.Dereference(input.URL.Value())
	}

	if fieldMask.Enabled {
		enabled = adapt.Dereference(input.Enabled.Value())
	}

	options := &backend.UpdateOptions{
		Name:        name,
		Description: description,
		Url:         url,
		Enabled:     enabled,
	}

	return options, fieldMask
}

func ToBackend(b *backend.Backend) *Backend {
	if b == nil {
		return nil
	}

	return &Backend{
		ID:          StringID(IdKindBackend, b.Id),
		Name:        b.Name,
		Description: b.Description,
		URL:         b.Url,
		Enabled:     b.Enabled,
		CreateUser:  userIdToUser(b.CreateUserId),
		UpdateUser:  userIdToUser(b.UpdateUserId),
		DeleteUser:  userIdToUser(b.DeleteUserId),
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
		DeletedAt:   b.DeletedAt,
	}
}
