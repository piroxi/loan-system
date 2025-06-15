package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"loan-service/entity"
	"loan-service/handler"
	"loan-service/handler/mocks"
	"loan-service/utils/auth"
	"loan-service/utils/constants"
	errs "loan-service/utils/errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetLoan(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface)
		expectStatus   int
		expectResponse handler.Response
	}{
		{
			name: "Success",
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mocksLoanUsecase.On("GetLoan", "1").Return(&entity.Loan{
					DBCommon: entity.DBCommon{
						ID: 1,
					},
					Principal:  1000,
					ROI:        5,
					Rate:       10,
					Status:     constants.StatusApproved,
					BorrowerID: 1,
				}, nil)
			},
			expectStatus: http.StatusOK,
			expectResponse: handler.Response{
				Data: map[string]interface{}{
					"created_at":  "0001-01-01T00:00:00Z",
					"updated_at":  "0001-01-01T00:00:00Z",
					"id":          float64(1),
					"principal":   float64(1000),
					"roi":         float64(5),
					"rate":        float64(10),
					"status":      string(constants.StatusApproved),
					"borrower_id": float64(1),
					"investments": interface{}(nil),
				},
			},
		},
		{
			name: "Loan not found",
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mocksLoanUsecase.On("GetLoan", "1").Return(nil, fmt.Errorf("loan not found"))
			},
			expectStatus: http.StatusNotFound,
			expectResponse: handler.Response{
				Error: errs.ErrLoanNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockLoanUsecase := mocks.NewLoanUsecaseInterface(t)
			mockUserUsecase := mocks.NewUserUsecaseInterface(t)
			auth.StartAuthorizer("test-secret")

			if tt.mockFunc != nil {
				tt.mockFunc(mockLoanUsecase, mockUserUsecase)
			}

			router := gin.Default()
			handler.RegisterLoanHandler(router.Group("/api"), mockLoanUsecase, mockUserUsecase)

			req, _ := http.NewRequest(http.MethodGet, "/api/loans/1", nil)
			token, _ := auth.GenerateToken("testuser", 1)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if tt.expectResponse.Data != nil {
				assert.Equal(t, tt.expectStatus, resp.Code)
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectResponse.Data, response.Data)
			} else {
				assert.Equal(t, tt.expectStatus, resp.Code)
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectResponse.Error, response.Error)
			}
		})
	}
}

func TestCreateLoan(t *testing.T) {
	tests := []struct {
		name           string
		body           entity.RequestProposeLoan
		mockFunc       func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface)
		expectStatus   int
		expectResponse handler.Response
	}{
		{
			name: "Success",
			body: entity.RequestProposeLoan{
				Principal: 1000,
				ROI:       5,
				Rate:      10,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleBorrower, nil)
				mocksLoanUsecase.On("CreateLoan", entity.RequestProposeLoan{
					Principal: 1000,
					ROI:       5,
					Rate:      10,
				}, uint(1)).Return(&entity.Loan{
					DBCommon:   entity.DBCommon{ID: 1},
					Principal:  1000,
					ROI:        5,
					Rate:       10,
					Status:     constants.StatusProposed,
					BorrowerID: 1,
				}, nil)
			},
			expectStatus: http.StatusCreated,
			expectResponse: handler.Response{
				Data: map[string]interface{}{
					"created_at":  "0001-01-01T00:00:00Z",
					"updated_at":  "0001-01-01T00:00:00Z",
					"id":          float64(1),
					"principal":   float64(1000),
					"roi":         float64(5),
					"rate":        float64(10),
					"status":      string(constants.StatusProposed),
					"borrower_id": float64(1),
					"investments": interface{}(nil),
				},
			},
		},
		{
			name: "Wrong role",
			body: entity.RequestProposeLoan{
				Principal: 1000,
				ROI:       5,
				Rate:      10,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleValidator, nil)
			},
			expectStatus: http.StatusForbidden,
			expectResponse: handler.Response{
				Error: errs.ErrUnauthorizedAction,
			},
		},
		{
			name: "Invalid request body",
			body: entity.RequestProposeLoan{
				Principal: 0,
				ROI:       5,
				Rate:      10,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleBorrower, nil)
			},
			expectStatus: http.StatusBadRequest,
			expectResponse: handler.Response{
				Error: "Error:Field validation",
			},
		},
		{
			name: "Invalid loan parameters",
			body: entity.RequestProposeLoan{
				Principal: 1000,
				ROI:       -5,
				Rate:      10,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleBorrower, nil)
			},
			expectStatus: http.StatusBadRequest,
			expectResponse: handler.Response{
				Error: "Invalid loan parameters",
			},
		},
		{
			name: "CreateLoan error",
			body: entity.RequestProposeLoan{
				Principal: 1000,
				ROI:       5,
				Rate:      10,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleBorrower, nil)
				mocksLoanUsecase.On("CreateLoan", entity.RequestProposeLoan{
					Principal: 1000,
					ROI:       5,
					Rate:      10,
				}, uint(1)).Return(nil, fmt.Errorf("error creating loan"))
			},
			expectStatus: http.StatusInternalServerError,
			expectResponse: handler.Response{
				Error: "error creating loan",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockLoanUsecase := mocks.NewLoanUsecaseInterface(t)
			mockUserUsecase := mocks.NewUserUsecaseInterface(t)
			auth.StartAuthorizer("test-secret")

			if tt.mockFunc != nil {
				tt.mockFunc(mockLoanUsecase, mockUserUsecase)
			}

			router := gin.Default()
			handler.RegisterLoanHandler(router.Group("/api"), mockLoanUsecase, mockUserUsecase)

			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/loans/create", bytes.NewBuffer(bodyBytes))
			token, _ := auth.GenerateToken("testuser", 1)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectStatus, resp.Code)
			if tt.expectStatus == http.StatusCreated {
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectResponse.Data, response.Data)
			} else {
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response.Error, tt.expectResponse.Error)
			}
		})
	}
}

func TestRejectLoan(t *testing.T) {
	rejectReason := "Insufficient credit score"
	tests := []struct {
		name           string
		body           entity.RequestRejectLoan
		mockFunc       func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface)
		expectStatus   int
		expectResponse handler.Response
	}{
		{
			name: "Success",
			body: entity.RequestRejectLoan{
				LoanID:       1,
				RejectReason: rejectReason,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleValidator, nil)
				mocksLoanUsecase.On("RejectLoan", entity.RequestRejectLoan{
					LoanID:       1,
					RejectReason: rejectReason,
				}, uint(1)).Return(&entity.LoanApproval{
					DBCommon:     entity.DBCommon{ID: 1},
					LoanID:       1,
					RejectReason: &rejectReason,
					ValidatorID:  1,
				}, nil)
			},
			expectStatus: http.StatusOK,
			expectResponse: handler.Response{
				Data: map[string]interface{}{
					"created_at":    "0001-01-01T00:00:00Z",
					"updated_at":    "0001-01-01T00:00:00Z",
					"approved_at":   "0001-01-01T00:00:00Z",
					"id":            float64(1),
					"loan_id":       float64(1),
					"reject_reason": rejectReason,
					"photo_url":     "",
					"validator_id":  float64(1),
				},
			},
		},
		{
			name: "Wrong role",
			body: entity.RequestRejectLoan{
				LoanID:       1,
				RejectReason: rejectReason,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleBorrower, nil)
			},
			expectStatus: http.StatusForbidden,
			expectResponse: handler.Response{
				Error: errs.ErrUnauthorizedAction,
			},
		},
		{
			name: "Invalid request body",
			body: entity.RequestRejectLoan{
				LoanID:       0,
				RejectReason: rejectReason,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleValidator, nil)
			},
			expectStatus: http.StatusBadRequest,
			expectResponse: handler.Response{
				Error: "Error:Field validation",
			},
		},
		{
			name: "RejectLoan error",
			body: entity.RequestRejectLoan{
				LoanID:       1,
				RejectReason: rejectReason,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleValidator, nil)
				mocksLoanUsecase.On("RejectLoan", entity.RequestRejectLoan{
					LoanID:       1,
					RejectReason: rejectReason,
				}, uint(1)).Return(nil, fmt.Errorf("error rejecting loan"))
			},
			expectStatus: http.StatusInternalServerError,
			expectResponse: handler.Response{
				Error: "error rejecting loan",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockLoanUsecase := mocks.NewLoanUsecaseInterface(t)
			mockUserUsecase := mocks.NewUserUsecaseInterface(t)
			auth.StartAuthorizer("test-secret")

			if tt.mockFunc != nil {
				tt.mockFunc(mockLoanUsecase, mockUserUsecase)
			}

			router := gin.Default()
			handler.RegisterLoanHandler(router.Group("/api"), mockLoanUsecase, mockUserUsecase)

			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/loans/reject", bytes.NewBuffer(bodyBytes))
			token, _ := auth.GenerateToken("testuser", 1)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectStatus, resp.Code)
			if tt.expectStatus == http.StatusOK {
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectResponse.Data, response.Data)
			} else {
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response.Error, tt.expectResponse.Error)
			}
		})
	}
}

func TestApproveLoan(t *testing.T) {
	tests := []struct {
		name           string
		body           entity.RequestApproveLoan
		mockFunc       func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface)
		expectStatus   int
		expectResponse handler.Response
	}{
		{
			name: "Success",
			body: entity.RequestApproveLoan{
				LoanID:   1,
				PhotoURL: "http://example.com/photo.jpg",
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleValidator, nil)
				mocksLoanUsecase.On("ApproveLoan", entity.RequestApproveLoan{
					LoanID:   1,
					PhotoURL: "http://example.com/photo.jpg",
				}, uint(1)).Return(&entity.LoanApproval{
					DBCommon:    entity.DBCommon{ID: 1},
					LoanID:      1,
					ValidatorID: 1,
					PhotoURL:    "http://example.com/photo.jpg",
				}, nil)
			},
			expectStatus: http.StatusOK,
			expectResponse: handler.Response{
				Data: map[string]interface{}{
					"created_at":   "0001-01-01T00:00:00Z",
					"updated_at":   "0001-01-01T00:00:00Z",
					"approved_at":  "0001-01-01T00:00:00Z",
					"id":           float64(1),
					"loan_id":      float64(1),
					"photo_url":    "http://example.com/photo.jpg",
					"validator_id": float64(1),
				},
			},
		},
		{
			name: "Wrong role",
			body: entity.RequestApproveLoan{
				LoanID:   1,
				PhotoURL: "http://example.com/photo.jpg",
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleBorrower, nil)
			},
			expectStatus: http.StatusForbidden,
			expectResponse: handler.Response{
				Error: errs.ErrUnauthorizedAction,
			},
		},
		{
			name: "Invalid request body",
			body: entity.RequestApproveLoan{
				LoanID:   0,
				PhotoURL: "http://example.com/photo.jpg",
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleValidator, nil)
			},
			expectStatus: http.StatusBadRequest,
			expectResponse: handler.Response{
				Error: "Error:Field validation",
			},
		},
		{
			name: "ApproveLoan error",
			body: entity.RequestApproveLoan{
				LoanID:   1,
				PhotoURL: "http://example.com/photo.jpg",
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleValidator, nil)
				mocksLoanUsecase.On("ApproveLoan", entity.RequestApproveLoan{
					LoanID:   1,
					PhotoURL: "http://example.com/photo.jpg",
				}, uint(1)).Return(nil, fmt.Errorf("error approving loan"))
			},
			expectStatus: http.StatusInternalServerError,
			expectResponse: handler.Response{
				Error: "error approving loan",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockLoanUsecase := mocks.NewLoanUsecaseInterface(t)
			mockUserUsecase := mocks.NewUserUsecaseInterface(t)
			auth.StartAuthorizer("test-secret")

			if tt.mockFunc != nil {
				tt.mockFunc(mockLoanUsecase, mockUserUsecase)
			}

			router := gin.Default()
			handler.RegisterLoanHandler(router.Group("/api"), mockLoanUsecase, mockUserUsecase)

			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/loans/approve", bytes.NewBuffer(bodyBytes))
			token, _ := auth.GenerateToken("testuser", 1)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectStatus, resp.Code)
			if tt.expectStatus == http.StatusOK {
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectResponse.Data, response.Data)
			} else {
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response.Error, tt.expectResponse.Error)
			}
		})
	}
}

func TestAddInvestment(t *testing.T) {
	tests := []struct {
		name           string
		body           entity.RequestAddInvestment
		mockFunc       func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface)
		expectStatus   int
		expectResponse handler.Response
	}{
		{
			name: "Success",
			body: entity.RequestAddInvestment{
				LoanID: 1,
				Amount: 500,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleInvestor, nil)
				mocksLoanUsecase.On("AddInvestment", mock.Anything, entity.RequestAddInvestment{
					LoanID: 1,
					Amount: 500,
				}, uint(1)).Return(&entity.Investment{
					DBCommon:   entity.DBCommon{ID: 1},
					LoanID:     1,
					Amount:     500,
					InvestorID: 1,
				}, nil)
			},
			expectStatus: http.StatusOK,
			expectResponse: handler.Response{
				Data: map[string]interface{}{
					"created_at":  "0001-01-01T00:00:00Z",
					"updated_at":  "0001-01-01T00:00:00Z",
					"id":          float64(1),
					"loan_id":     float64(1),
					"amount":      float64(500),
					"investor_id": float64(1),
				},
			},
		},
		{
			name: "Wrong role",
			body: entity.RequestAddInvestment{
				LoanID: 1,
				Amount: 500,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleBorrower, nil)
			},
			expectStatus: http.StatusForbidden,
			expectResponse: handler.Response{
				Error: errs.ErrUnauthorizedAction,
			},
		},
		{
			name: "Invalid request body",
			body: entity.RequestAddInvestment{
				LoanID: 0,
				Amount: 500,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleInvestor, nil)
			},
			expectStatus: http.StatusBadRequest,
			expectResponse: handler.Response{
				Error: "Error:Field validation",
			},
		},
		{
			name: "Wrong input",
			body: entity.RequestAddInvestment{
				LoanID: 1,
				Amount: -10,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleInvestor, nil)
			},
			expectStatus: http.StatusBadRequest,
			expectResponse: handler.Response{
				Error: "Invalid input: LoanID and Amount are required",
			},
		},
		{
			name: "AddInvestment error",
			body: entity.RequestAddInvestment{
				LoanID: 1,
				Amount: 500,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleInvestor, nil)
				mocksLoanUsecase.On("AddInvestment", mock.Anything, entity.RequestAddInvestment{
					LoanID: 1,
					Amount: 500,
				}, uint(1)).Return(nil, fmt.Errorf("error adding investment"))
			},
			expectStatus: http.StatusInternalServerError,
			expectResponse: handler.Response{
				Error: "error adding investment",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockLoanUsecase := mocks.NewLoanUsecaseInterface(t)
			mockUserUsecase := mocks.NewUserUsecaseInterface(t)
			auth.StartAuthorizer("test-secret")

			if tt.mockFunc != nil {
				tt.mockFunc(mockLoanUsecase, mockUserUsecase)
			}

			router := gin.Default()
			handler.RegisterLoanHandler(router.Group("/api"), mockLoanUsecase, mockUserUsecase)

			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/loans/invest", bytes.NewBuffer(bodyBytes))
			token, _ := auth.GenerateToken("testuser", 1)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectStatus, resp.Code)
			if tt.expectStatus == http.StatusOK {
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectResponse.Data, response.Data)
			} else {
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response.Error, tt.expectResponse.Error)
			}
		})
	}
}

func TestDisburseLoan(t *testing.T) {
	tests := []struct {
		name           string
		body           entity.RequestDisburseLoan
		mockFunc       func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface)
		expectStatus   int
		expectResponse handler.Response
	}{
		{
			name: "Success",
			body: entity.RequestDisburseLoan{
				LoanID:             1,
				SignedAgreementURL: "http://example.com/agreement.pdf",
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleDisburser, nil)
				mocksLoanUsecase.On("DisburseLoan", entity.RequestDisburseLoan{
					LoanID:             1,
					SignedAgreementURL: "http://example.com/agreement.pdf",
				}, uint(1)).Return(&entity.LoanDisbursement{
					DBCommon:           entity.DBCommon{ID: 1},
					LoanID:             1,
					SignedAgreementURL: "http://example.com/agreement.pdf",
					DisburserID:        1,
				}, nil)
			},
			expectStatus: http.StatusOK,
			expectResponse: handler.Response{
				Data: map[string]interface{}{
					"created_at":           "0001-01-01T00:00:00Z",
					"updated_at":           "0001-01-01T00:00:00Z",
					"disbursed_at":         "0001-01-01T00:00:00Z",
					"disburser_id":         float64(1),
					"id":                   float64(1),
					"loan_id":              float64(1),
					"signed_agreement_url": "http://example.com/agreement.pdf",
				},
			},
		},
		{
			name: "Wrong role",
			body: entity.RequestDisburseLoan{
				LoanID:             1,
				SignedAgreementURL: "http://example.com/agreement.pdf",
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleBorrower, nil)
			},
			expectStatus: http.StatusForbidden,
			expectResponse: handler.Response{
				Error: errs.ErrUnauthorizedAction,
			},
		},
		{
			name: "Invalid request body",
			body: entity.RequestDisburseLoan{
				LoanID:             0,
				SignedAgreementURL: "http://example.com/agreement.pdf",
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleDisburser, nil)
			},
			expectStatus: http.StatusBadRequest,
			expectResponse: handler.Response{
				Error: "Error:Field validation",
			},
		},
		{
			name: "DisburseLoan error",
			body: entity.RequestDisburseLoan{
				LoanID:             1,
				SignedAgreementURL: "http://example.com/agreement.pdf",
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleDisburser, nil)
				mocksLoanUsecase.On("DisburseLoan", entity.RequestDisburseLoan{
					LoanID:             1,
					SignedAgreementURL: "http://example.com/agreement.pdf",
				}, uint(1)).Return(nil, fmt.Errorf("error disbursing loan"))
			},
			expectStatus: http.StatusInternalServerError,
			expectResponse: handler.Response{
				Error: "error disbursing loan",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockLoanUsecase := mocks.NewLoanUsecaseInterface(t)
			mockUserUsecase := mocks.NewUserUsecaseInterface(t)
			auth.StartAuthorizer("test-secret")

			if tt.mockFunc != nil {
				tt.mockFunc(mockLoanUsecase, mockUserUsecase)
			}

			router := gin.Default()
			handler.RegisterLoanHandler(router.Group("/api"), mockLoanUsecase, mockUserUsecase)

			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/loans/disburse", bytes.NewBuffer(bodyBytes))
			token, _ := auth.GenerateToken("testuser", 1)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectStatus, resp.Code)
			if tt.expectStatus == http.StatusOK {
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectResponse.Data, response.Data)
			} else {
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response.Error, tt.expectResponse.Error)
			}
		})
	}
}

func TestAuthAndRole(t *testing.T) {
	tests := []struct {
		name           string
		body           entity.RequestProposeLoan
		authHeader     func() string
		mockFunc       func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface)
		expectStatus   int
		expectResponse handler.Response
	}{
		{
			name: "Create Loan as Admin",
			authHeader: func() string {
				token, _ := auth.GenerateToken("testuser", 1)
				return fmt.Sprintf("Bearer %s", token)
			},
			body: entity.RequestProposeLoan{
				Principal: 1000,
				ROI:       5,
				Rate:      10,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleAdmin, nil)
				mocksLoanUsecase.On("CreateLoan", entity.RequestProposeLoan{
					Principal: 1000,
					ROI:       5,
					Rate:      10,
				}, uint(1)).Return(&entity.Loan{
					DBCommon:   entity.DBCommon{ID: 1},
					Principal:  1000,
					ROI:        5,
					Rate:       10,
					Status:     constants.StatusProposed,
					BorrowerID: 1,
				}, nil)
			},
			expectStatus: http.StatusCreated,
			expectResponse: handler.Response{
				Data: map[string]interface{}{
					"created_at":  "0001-01-01T00:00:00Z",
					"updated_at":  "0001-01-01T00:00:00Z",
					"id":          float64(1),
					"principal":   float64(1000),
					"roi":         float64(5),
					"rate":        float64(10),
					"status":      string(constants.StatusProposed),
					"borrower_id": float64(1),
					"investments": interface{}(nil),
				},
			},
		},
		{
			name: "No Authorization Header",
			authHeader: func() string {
				return ""
			},
			body: entity.RequestProposeLoan{
				Principal: 1000,
				ROI:       5,
				Rate:      10,
			},
			expectStatus: http.StatusUnauthorized,
			expectResponse: handler.Response{
				Error: "Unauthorized",
			},
		},
		{
			name: "Invalid Authorization Header",
			authHeader: func() string {
				return "InvalidHeader"
			},
			body: entity.RequestProposeLoan{
				Principal: 1000,
				ROI:       5,
				Rate:      10,
			},
			expectStatus: http.StatusUnauthorized,
			expectResponse: handler.Response{
				Error: "token is malformed",
			},
		},
		{
			name: "Wrong Authorization Header",
			authHeader: func() string {
				return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.KMUFsIDTnFmyG3nMiGM6H9FNFUROf3wh7SmqJp-QV30"
			},
			body: entity.RequestProposeLoan{
				Principal: 1000,
				ROI:       5,
				Rate:      10,
			},
			expectStatus: http.StatusUnauthorized,
			expectResponse: handler.Response{
				Error: "token is malformed",
			},
		},
		{
			name: "Error GetUserRole",
			authHeader: func() string {
				token, _ := auth.GenerateToken("testuser", 1)
				return fmt.Sprintf("Bearer %s", token)
			},
			body: entity.RequestProposeLoan{
				Principal: 1000,
				ROI:       5,
				Rate:      10,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				fmt.Println("Mocking GetUserRole to return an error")
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleUnknown, fmt.Errorf("error fetching user role"))
			},
			expectStatus: http.StatusForbidden,
			expectResponse: handler.Response{
				Error: errs.ErrUnauthorizedAction,
			},
		},
		{
			name: "Create Loan as Validator",
			authHeader: func() string {
				token, _ := auth.GenerateToken("testuser", 1)
				return fmt.Sprintf("Bearer %s", token)
			},
			body: entity.RequestProposeLoan{
				Principal: 1000,
				ROI:       5,
				Rate:      10,
			},
			mockFunc: func(mocksLoanUsecase *mocks.LoanUsecaseInterface, mockUserUsecase *mocks.UserUsecaseInterface) {
				mockUserUsecase.On("GetUserRole", uint(1)).Return(constants.RoleValidator, nil)
			},
			expectStatus: http.StatusForbidden,
			expectResponse: handler.Response{
				Error: errs.ErrUnauthorizedAction,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockLoanUsecase := mocks.NewLoanUsecaseInterface(t)
			mockUserUsecase := mocks.NewUserUsecaseInterface(t)
			auth.StartAuthorizer("test-secret")

			if tt.mockFunc != nil {
				tt.mockFunc(mockLoanUsecase, mockUserUsecase)
			}

			router := gin.Default()
			handler.RegisterLoanHandler(router.Group("/api"), mockLoanUsecase, mockUserUsecase)

			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/loans/create", bytes.NewBuffer(bodyBytes))
			authHeader := tt.authHeader()
			req.Header.Set("Authorization", authHeader)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectStatus, resp.Code)
			if tt.expectStatus == http.StatusCreated {
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectResponse.Data, response.Data)
			} else {
				var response handler.Response
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response.Error, tt.expectResponse.Error)
			}
		})
	}
}
