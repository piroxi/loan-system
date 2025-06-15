package auth_test

import (
	"errors"
	"loan-service/utils/auth"
	errs "loan-service/utils/errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

// Initialize authorizer
const secret = "test-secret"

func TestAuthorizerNotInitialized(t *testing.T) {
	t.Run("uninitialized authorizer", func(t *testing.T) {
		token, err := auth.GenerateToken("test", 1)
		assert.Error(t, err)
		assert.Equal(t, errs.ErrAuthUninitialized, err.Error())
		assert.Empty(t, token)

		// Restore instance for other tests
		auth.StartAuthorizer(secret)
	})
}

func TestStartAuthorizer(t *testing.T) {
	t.Run("singleton initialization", func(t *testing.T) {
		// First call should initialize
		auth1 := auth.StartAuthorizer(secret)
		assert.NotNil(t, auth1)

		// Subsequent calls should return same instance
		auth2 := auth.StartAuthorizer("secret2")
		assert.Equal(t, auth1, auth2)
		assert.Equal(t, secret, auth2.GetSecret()) // Should keep original secret
	})
}

func TestGenerateToken(t *testing.T) {
	auth.StartAuthorizer(secret)

	tests := []struct {
		name     string
		username string
		userID   uint
		wantErr  bool
	}{
		{
			name:     "successful token generation",
			username: "testuser",
			userID:   123,
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			userID:   123,
			wantErr:  false, // Empty username is allowed as it's not in claims
		},
		{
			name:     "zero user ID",
			username: "testuser",
			userID:   0,
			wantErr:  false, // Generation succeeds, validation will fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := auth.GenerateToken(tt.username, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				// Verify token can be parsed and contains correct claims
				parsedToken, err := jwt.ParseWithClaims(token, &auth.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
					return []byte(secret), nil
				})

				assert.NoError(t, err)
				assert.True(t, parsedToken.Valid)

				if claims, ok := parsedToken.Claims.(*auth.UserClaims); ok {
					assert.Equal(t, tt.userID, claims.UserID)
					assert.Equal(t, "loan-service", claims.Issuer)
					assert.WithinDuration(t, time.Now().Add(24*time.Hour), claims.ExpiresAt.Time, time.Second)
				} else {
					t.Fatal("failed to parse claims")
				}
			}
		})
	}
}

func TestClaimToken(t *testing.T) {
	// Initialize authorizer with test secret
	auth.StartAuthorizer(secret)

	// Generate a valid token for testing
	validToken, err := auth.GenerateToken("testuser", 123)
	assert.NoError(t, err)

	// Generate an expired token
	expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256,
		auth.UserClaims{
			UserID: 456,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				Issuer:    "loan-service",
			},
		},
	).SignedString([]byte(secret))
	assert.NoError(t, err)

	tests := []struct {
		name        string
		tokenString string
		wantClaims  *auth.UserClaims
		wantErr     error
	}{
		{
			name:        "valid token",
			tokenString: validToken,
			wantClaims: &auth.UserClaims{
				UserID: 123,
			},
			wantErr: nil,
		},
		{
			name:        "expired token",
			tokenString: expiredToken,
			wantClaims:  nil,
			wantErr:     errors.New("token is expired"),
		},
		{
			name:        "malformed token",
			tokenString: "malformed.token.string",
			wantClaims:  nil,
			wantErr:     errors.New("could not base64 decode header"),
		},
		{
			name:        "empty token",
			tokenString: "",
			wantClaims:  nil,
			wantErr:     errors.New("token contains an invalid number of segments"),
		},
		{
			name:        "invalid signature",
			tokenString: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMjN9.invalid-signature",
			wantClaims:  nil,
			wantErr:     errors.New("could not base64 decode signature"),
		},
		{
			name: "zero user ID",
			tokenString: func() string {
				token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256,
					auth.UserClaims{
						UserID: 0,
						RegisteredClaims: jwt.RegisteredClaims{
							ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
							IssuedAt:  jwt.NewNumericDate(time.Now()),
							Issuer:    "loan-service",
						},
					},
				).SignedString([]byte(secret))
				return token
			}(),
			wantClaims: nil,
			wantErr:    errors.New(errs.ErrInvalidToken),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := auth.ClaimToken(tt.tokenString)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr.Error())
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, tt.wantClaims.UserID, claims.UserID)
			}
		})
	}
}

func TestUserClaims(t *testing.T) {
	t.Run("claims validation", func(t *testing.T) {
		claims := auth.UserClaims{
			UserID: 123,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				Issuer:    "loan-service",
			},
		}

		assert.Equal(t, uint(123), claims.UserID)
		assert.Equal(t, "loan-service", claims.Issuer)
	})
}
