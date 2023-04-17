package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Service interface {
	Authenticate(ctx context.Context, username string, password string) (*User, error)
	FindUser(ctx context.Context, options *FindOneOptions) (*User, error)
	FindUsers(ctx context.Context, options *FindOptions) ([]*User, error)
	CreateUser(ctx context.Context, options *CreateOptions) (*User, error)
	UpdateUser(ctx context.Context, userId string, options *UpdateOptions, fieldMask *UpdateFieldMask) (*User, error)
	DeleteUser(ctx context.Context, userId string) (*User, error)
}

type service struct {
	userRepository Repository
}

func NewService(userRepository Repository, initialEmail string, initialPassword string) (Service, error) {
	s := &service{
		userRepository: userRepository,
	}
	if err := s.ensureUserCreated(context.Background(), initialEmail, initialPassword); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *service) Authenticate(ctx context.Context, email string, password string) (*User, error) {
	user, err := s.userRepository.FindOne(ctx, &FindOneOptions{
		IdOption: nil,
		EmailOption: &EmailOption{
			Email: email,
		},
	})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := checkPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) FindUser(ctx context.Context, options *FindOneOptions) (*User, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}
	return s.userRepository.FindOne(ctx, options)
}

func (s *service) FindUsers(ctx context.Context, options *FindOptions) ([]*User, error) {
	return s.userRepository.FindAll(ctx, options)
}

func (s *service) CreateUser(ctx context.Context, options *CreateOptions) (*User, error) {
	user, err := processCreateUser(options)
	if err != nil {
		return nil, err
	}
	return s.userRepository.Create(ctx, user)
}

func (s *service) findUserById(ctx context.Context, userId string) (*User, error) {
	user, err := s.userRepository.FindOne(ctx, &FindOneOptions{
		IdOption: &IdOption{
			Id: userId,
		},
	})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *service) UpdateUser(ctx context.Context, userId string, options *UpdateOptions, fieldMask *UpdateFieldMask) (*User, error) {
	user, err := s.findUserById(ctx, userId)
	if err != nil {
		return nil, err
	}

	if err := processUpdateUser(user, options, fieldMask); err != nil {
		return nil, err
	}

	return s.userRepository.Update(ctx, user, fieldMask)
}

func (s *service) DeleteUser(ctx context.Context, userId string) (*User, error) {
	user, err := s.findUserById(ctx, userId)
	if err != nil {
		return nil, err
	}
	return s.userRepository.Delete(ctx, user.Id)
}

func (s *service) ensureUserCreated(ctx context.Context, email string, password string) error {
	users, err := s.userRepository.FindAll(ctx, &FindOptions{})
	if err != nil {
		return err
	}
	if len(users) == 0 {
		var generatedRandomPassword bool
		if email == "" {
			email = "admin@example.com"
		}
		if password == "" || password == "random" {
			password = generateRandomPassword(16, 4, 4, 4)
			generatedRandomPassword = true
		}
		createdUser, err := s.CreateUser(ctx, &CreateOptions{
			Email:    email,
			Password: password,
		})
		if err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}

		if generatedRandomPassword {
			logrus.
				WithField("email", createdUser.Email).
				WithField("password", password).
				Info("admin user created")
		} else {
			logrus.
				WithField("email", createdUser.Email).
				Info("admin user created")
		}
	}
	return nil
}

func newId() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func processCreateUser(options *CreateOptions) (*User, error) {
	if options == nil {
		return nil, ErrCreateOptionsRequired
	}
	if len(options.Email) == 0 {
		return nil, ErrEmailRequired
	}
	if !govalidator.IsEmail(options.Email) {
		return nil, ErrEmailInvalid
	}

	id, err := newId()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new id: %w", err)
	}

	password, err := generatePassword([]byte(options.Password))
	if err != nil {
		return nil, err
	}

	now := time.Now()

	return &User{
		Id:        id,
		Email:     strings.ToLower(options.Email),
		Password:  string(password),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func processUpdateUser(user *User, options *UpdateOptions, fieldMask *UpdateFieldMask) error {
	if options == nil {
		return ErrUpdateOptionsRequired
	}

	if fieldMask == nil {
		return ErrUpdateFieldMaskRequired
	}

	if fieldMask.Email && !strings.EqualFold(user.Email, options.Email) {
		if !govalidator.IsEmail(options.Email) {
			return ErrEmailInvalid
		}
	}

	if fieldMask.Password && len(options.Password) != 0 {
		password, err := generatePassword([]byte(options.Password))
		if err != nil {
			return err
		}
		options.Password = string(password)
	}

	user.Update(options, fieldMask)
	user.UpdatedAt = time.Now()
	return nil
}
