package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/go-chi/jwtauth/v5"
)

func websocketAuthenticationInit(jwtAuth *jwtauth.JWTAuth) func(ctx context.Context, payload transport.InitPayload) (context.Context, error) {
	return func(ctx context.Context, payload transport.InitPayload) (context.Context, error) {
		authorization := payload.Authorization()
		if len(authorization) <= 7 || strings.ToUpper(authorization[0:6]) != "BEARER" {
			return ctx, nil
		}

		token, err := jwtauth.VerifyToken(jwtAuth, authorization[7:])
		return jwtauth.NewContext(ctx, token, err), nil
	}
}

func authenticationMiddleware(jwtAuth *jwtauth.JWTAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := jwtauth.VerifyRequest(jwtAuth, r, jwtauth.TokenFromHeader, jwtauth.TokenFromCookie)
			ctx := jwtauth.NewContext(r.Context(), token, err)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
