package auth

import (
	"fmt"
	errs "load-service/utils/errors"
	"load-service/utils/logger"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type Authorizer struct {
	secret string
}

var (
	instance *Authorizer
	once     sync.Once
)

func StartAuthorizer(secret string) *Authorizer {
	once.Do(func() {
		instance = newAuthorizer(secret)
	})
	return instance
}

func newAuthorizer(secret string) *Authorizer {
	return &Authorizer{
		secret: secret,
	}
}

type UserClaims struct {
	jwt.RegisteredClaims
	UserID uint `json:"user_id"`
}

func GenerateToken(username string, userID uint) (string, error) {
	if instance == nil {
		return "", fmt.Errorf(errs.ErrAuthUninitialized)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		UserClaims{
			UserID: userID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    "loan-service",
			},
		},
	)

	tokenString, err := token.SignedString([]byte(instance.secret))
	if err != nil {
		logger.Error("Unable to sign token", zap.Error(err))
		return "", err
	}

	return tokenString, nil
}

func ClaimToken(tokenString string) (*UserClaims, error) {
	if instance == nil {
		return nil, fmt.Errorf(errs.ErrAuthUninitialized)
	}

	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(instance.secret), nil
	})

	if err != nil {
		logger.Error("Unable to parse token", zap.Error(err))
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf(errs.ErrInvalidToken)
	}
	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		logger.Error("Invalid token claims type")
		return nil, fmt.Errorf(errs.ErrInvalidToken)
	}

	if claims.UserID == 0 {
		logger.Error("Invalid user ID in token claims")
		return nil, fmt.Errorf(errs.ErrInvalidToken)
	}

	return claims, nil
}
