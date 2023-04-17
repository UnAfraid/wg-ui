package model

import (
	"context"

	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/user"
)

func ToUser(user *user.User) *User {
	if user == nil {
		return nil
	}
	return &User{
		ID:        StringID(IdKindUser, user.Id),
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func CreateUserInputToUserCreateUserOptions(createUserInput CreateUserInput) *user.CreateOptions {
	return &user.CreateOptions{
		Email:    createUserInput.Email,
		Password: createUserInput.Password,
	}
}

func UpdateUserInputToUserUpdateUserOptions(ctx context.Context, updateUserInput UpdateUserInput) (*user.UpdateOptions, *user.UpdateFieldMask, error) {
	if err := updateUserInput.ID.Validate(IdKindUser); err != nil {
		return nil, nil, err
	}

	fieldMask := &user.UpdateFieldMask{
		Email:    resolverHasArgumentField(ctx, "input", "email"),
		Password: resolverHasArgumentField(ctx, "input", "password"),
	}

	var (
		email    string
		password string
	)

	if fieldMask.Email {
		email = adapt.Dereference(updateUserInput.Email)
	}

	if fieldMask.Password {
		password = adapt.Dereference(updateUserInput.Password)
	}

	options := &user.UpdateOptions{
		Email:    email,
		Password: password,
	}

	return options, fieldMask, nil
}

func userIdToUser(userId string) *User {
	if userId == "" {
		return nil
	}
	return &User{
		ID: StringID(IdKindUser, userId),
	}
}
