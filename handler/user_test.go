package handler_test

import (
	"bytes"
	"encoding/json"
	"loan-service/entity"
	"loan-service/handler"
	"loan-service/handler/mocks"
	"loan-service/utils/auth"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSignin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name               string
		body               gin.H
		mockFunc           func(mockUsecase *mocks.UserUsecaseInterface)
		expectStatus       int
		expectTokenCreated bool
	}{
		{
			name: "Success",
			body: gin.H{"username": "testuser"},
			mockFunc: func(mockUsecase *mocks.UserUsecaseInterface) {
				mockUsecase.
					On("GetUserByUsername", mock.Anything).
					Return(&entity.User{
						DBCommon: entity.DBCommon{
							ID: 1,
						},
						Username: "testuser",
					}, nil)
			},
			expectStatus:       http.StatusOK,
			expectTokenCreated: true,
		},
		{
			name:               "Invalid request body",
			body:               gin.H{}, // Missing username
			expectStatus:       http.StatusBadRequest,
			expectTokenCreated: false,
		},
		{
			name:               "Missing username",
			body:               gin.H{"username": ""},
			expectStatus:       http.StatusBadRequest,
			expectTokenCreated: false,
		},
		{
			name: "User fetch error",
			body: gin.H{"username": "failuser"},
			mockFunc: func(mockUsecase *mocks.UserUsecaseInterface) {
				mockUsecase.
					On("GetUserByUsername", mock.Anything).
					Return(nil, assert.AnError)
			},
			expectStatus:       http.StatusInternalServerError,
			expectTokenCreated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsecase := new(mocks.UserUsecaseInterface)

			if tt.mockFunc != nil {
				tt.mockFunc(mockUsecase)
			}

			auth.StartAuthorizer("test-secret")

			router := gin.Default()
			handler.RegisterUserHandler(router.Group("/api"), mockUsecase)

			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/users/signin", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectStatus, resp.Code)

			if tt.expectTokenCreated && resp.Code == http.StatusOK {
				var jsonResp map[string]map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &jsonResp)
				assert.NoError(t, err)
				assert.Contains(t, jsonResp["data"], "token")
				assert.Contains(t, jsonResp["data"], "user")
			}
		})
	}
}
