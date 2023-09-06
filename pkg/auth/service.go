package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Service interface {
	Sign(userId string) (tokenString string, expiresIn time.Duration, expiresAt time.Time, err error)
	Parse(tokenString string) (string, error)
}

type service struct {
	signingMethod   jwt.SigningMethod
	privateKey      interface{}
	publicKey       interface{}
	sessionDuration time.Duration
}

func NewService(method jwt.SigningMethod, privateKey interface{}, publicKey interface{}, sessionDuration time.Duration) Service {
	return &service{
		signingMethod:   method,
		privateKey:      privateKey,
		publicKey:       publicKey,
		sessionDuration: sessionDuration,
	}
}

func (s *service) Sign(userId string) (tokenString string, expiresIn time.Duration, expiresAt time.Time, err error) {
	now := time.Now()
	expiresIn = s.sessionDuration
	expiresAt = now.Add(expiresIn)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		UserId: userId,
		RegisteredClaims: &jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})
	tokenString, err = token.SignedString(s.privateKey)
	if err != nil {
		return "", expiresIn, expiresAt, err
	}
	return tokenString, expiresIn, expiresAt, nil
}

func (s *service) Parse(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != s.signingMethod.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.publicKey, nil
		},
		jwt.WithIssuedAt(),
		jwt.WithValidMethods([]string{s.signingMethod.Alg()}),
		jwt.WithStrictDecoding(),
	)
	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", errors.New("token is invalid")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", fmt.Errorf("unexpected claims type: %T", token.Claims)
	}

	return claims.UserId, nil
}
