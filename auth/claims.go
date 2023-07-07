package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	*jwt.RegisteredClaims

	UserId string `json:"userId,omitempty"`
}
