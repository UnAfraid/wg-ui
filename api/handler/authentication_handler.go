package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/UnAfraid/wg-ui/api/internal/model"
	"github.com/UnAfraid/wg-ui/auth"
	"github.com/UnAfraid/wg-ui/user"
)

type AuthenticationHandler interface {
	WebsocketMiddleware() func(ctx context.Context, payload transport.InitPayload) (context.Context, error)
	AuthenticationMiddleware() func(http.Handler) http.Handler
}

type authenticationHandler struct {
	authService auth.Service
	userService user.Service
}

func NewAuthenticationMiddleware(authService auth.Service, userService user.Service) AuthenticationHandler {
	return &authenticationHandler{
		authService: authService,
		userService: userService,
	}
}

func (ah *authenticationHandler) WebsocketMiddleware() func(ctx context.Context, payload transport.InitPayload) (context.Context, error) {
	return func(ctx context.Context, payload transport.InitPayload) (context.Context, error) {
		authorizationHeader := payload.Authorization()
		if len(authorizationHeader) <= 7 || strings.ToUpper(authorizationHeader[0:6]) != "BEARER" {
			return ctx, nil
		}

		tokenString := authorizationHeader[7:]
		return ah.processToken(ctx, tokenString), nil
	}
}

func (ah *authenticationHandler) AuthenticationMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			authorizationHeader := r.Header.Get("Authorization")

			if len(authorizationHeader) <= 7 || strings.ToUpper(authorizationHeader[0:6]) != "BEARER" {
				next.ServeHTTP(w, r)
				return
			}

			tokenString := authorizationHeader[7:]
			next.ServeHTTP(w, r.WithContext(ah.processToken(ctx, tokenString)))
		})
	}
}

func (ah *authenticationHandler) processToken(ctx context.Context, tokenString string) context.Context {
	userID, err := ah.authService.Parse(tokenString)
	if err != nil {
		return model.UserToContext(ctx, nil, err)
	}

	if len(userID) == 0 {
		return model.UserToContext(ctx, nil, ErrAuthenticationRequired)
	}

	userId, err := parseUserId(userID)
	if err != nil {
		return model.UserToContext(ctx, nil, ErrClaimsInvalid)
	}

	u, err := ah.findUser(ctx, userId)
	if err != nil {
		return model.UserToContext(ctx, nil, err)
	}

	return model.UserToContext(ctx, u, nil)
}

func parseUserId(userID string) (string, error) {
	var id model.ID
	if err := id.UnmarshalGQL(userID); err != nil {
		return "", err
	}
	return id.String(model.IdKindUser)
}

func (ah *authenticationHandler) findUser(ctx context.Context, userId string) (*model.User, error) {
	u, err := ah.userService.FindUser(ctx, &user.FindOneOptions{
		IdOption: &user.IdOption{
			Id: userId,
		},
	})
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	return model.ToUser(u), nil
}
