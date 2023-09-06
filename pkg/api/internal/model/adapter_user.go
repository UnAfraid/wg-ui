package model

import (
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/user"
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

func UpdateUserInputToUserUpdateUserOptions(input UpdateUserInput) (*user.UpdateOptions, *user.UpdateFieldMask, error) {
	if err := input.ID.Validate(IdKindUser); err != nil {
		return nil, nil, err
	}

	fieldMask := &user.UpdateFieldMask{
		Email:    input.Email.IsSet(),
		Password: input.Password.IsSet(),
	}

	var (
		email    string
		password string
	)

	if fieldMask.Email {
		email = adapt.Dereference(input.Email.Value())
	}

	if fieldMask.Password {
		password = adapt.Dereference(input.Password.Value())
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
