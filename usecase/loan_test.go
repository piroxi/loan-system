package usecase_test

import (
	"context"
	"fmt"
	"loan-service/entity"
	"loan-service/usecase"
	"loan-service/utils/constants"
	errs "loan-service/utils/errors"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestLoanUsecase_CreateLoan(t *testing.T) {
	const loanID = 1
	const borrowerID = 2
	const roi = 5.0
	const rate = 10.0
	const principal = 1000.0

	type args struct {
		loanRequest entity.RequestProposeLoan
		borrowerID  uint
	}
	tests := []struct {
		name      string
		args      args
		mockFunc  func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock)
		doCleanup bool
		want      *entity.Loan
		wantErr   error
	}{
		{
			name: "CreateLoan_Success",
			args: args{
				loanRequest: entity.RequestProposeLoan{
					Principal: principal,
					ROI:       roi,
					Rate:      rate,
				},
				borrowerID: borrowerID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectBegin()
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "loans"`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						borrowerID,
						principal,
						rate,
						roi,
						constants.StatusProposed,
						nil,
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(loanID))
				mockSql.ExpectExec(regexp.QuoteMeta(
					`UPDATE "loans"`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						borrowerID,
						principal,
						rate,
						roi,
						constants.StatusProposed,
						fmt.Sprintf("https://example.com/loans/%d/loan_proposal_%d.pdf", loanID, loanID),
						loanID,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mockSql.ExpectCommit()
			},
			doCleanup: true,
			want: &entity.Loan{
				DBCommon: entity.DBCommon{
					ID: loanID,
				},
				Principal:     principal,
				ROI:           roi,
				Rate:          rate,
				BorrowerID:    borrowerID,
				Status:        constants.StatusProposed,
				AgreementLink: &[]string{fmt.Sprintf("https://example.com/loans/%d/loan_proposal_%d.pdf", loanID, loanID)}[0],
			},
			wantErr: nil,
		},
		{
			name: "CreateLoan_Failure_DBError_CreateLoan",
			args: args{
				loanRequest: entity.RequestProposeLoan{
					Principal: principal,
					ROI:       roi,
					Rate:      rate,
				},
				borrowerID: borrowerID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectBegin()
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "loans"`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						borrowerID,
						principal,
						rate,
						roi,
						constants.StatusProposed,
						nil,
					).
					WillReturnError(fmt.Errorf("DB error"))
				mockSql.ExpectRollback()
			},
			wantErr: fmt.Errorf("DB error"),
		},
		{
			name: "CreateLoan_Failure_DBError_SaveLink",
			args: args{
				loanRequest: entity.RequestProposeLoan{
					Principal: principal,
					ROI:       roi,
					Rate:      rate,
				},
				borrowerID: borrowerID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectBegin()
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "loans"`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						borrowerID,
						principal,
						rate,
						roi,
						constants.StatusProposed,
						nil,
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(loanID))
				mockSql.ExpectExec(regexp.QuoteMeta(
					`UPDATE "loans"`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						borrowerID,
						principal,
						rate,
						roi,
						constants.StatusProposed,
						fmt.Sprintf("https://example.com/loans/%d/loan_proposal_%d.pdf", loanID, loanID),
						loanID,
					).WillReturnError(fmt.Errorf("DB error on save link"))
				mockSql.ExpectRollback()
			},
			doCleanup: true,
			wantErr:   fmt.Errorf("failed to save loan with PDF URL"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mockSql := setupMockDB(t)
			redis, mockRedis := redismock.NewClientMock()

			u := usecase.NewLoanUsecase(db, redis)

			if tt.mockFunc != nil {
				tt.mockFunc(mockSql, mockRedis)
			}

			got, err := u.CreateLoan(tt.args.loanRequest, tt.args.borrowerID)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.BorrowerID, got.BorrowerID)
				assert.Equal(t, tt.want.Principal, got.Principal)
				assert.Equal(t, tt.want.ROI, got.ROI)
				assert.Equal(t, tt.want.Rate, got.Rate)
				assert.Equal(t, tt.want.Status, got.Status)
				assert.Equal(t, tt.want.ID, got.ID)
				if tt.want.AgreementLink != nil {
					assert.Equal(t, *tt.want.AgreementLink, *got.AgreementLink)
				}
			}
			// Cleanup: remove the generated PDF file if it was created
			if tt.doCleanup {
				filename := fmt.Sprintf("loan_proposal_%d.pdf", loanID)

				_, statErr := os.Stat(filename)
				assert.NoError(t, statErr)

				removeErr := os.Remove(filename)
				assert.NoError(t, removeErr)

				_, statErr = os.Stat(filename)
				assert.True(t, os.IsNotExist(statErr))
			}
		})
	}
}

func TestLoanUsecase_RejectLoan(t *testing.T) {
	loanID := uint(1)
	validatorID := uint(2)
	rejectReason := "Insufficient credit score"
	type args struct {
		rejectionRequest entity.RequestRejectLoan
		validatorID      uint
	}
	tests := []struct {
		name     string
		args     args
		mockFunc func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock)
		want     *entity.LoanApproval
		wantErr  error
	}{
		{
			name: "RejectLoan_Success",
			args: args{
				rejectionRequest: entity.RequestRejectLoan{
					LoanID:       1,
					RejectReason: rejectReason,
				},
				validatorID: validatorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs(1, constants.StatusProposed, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(loanID, constants.StatusProposed))
				mockSql.ExpectBegin()
				mockSql.ExpectExec(regexp.QuoteMeta(
					`UPDATE "loans"`)).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), constants.StatusRejected, sqlmock.AnyArg(), loanID,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "loan_approvals"`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						loanID,
						validatorID,
						rejectReason,
						"",
						sqlmock.AnyArg(),
					).
					WillReturnRows(sqlmock.NewRows([]string{"loan_id", "validator_id", "reject_reason"}).AddRow(loanID, validatorID, rejectReason))
				mockSql.ExpectCommit()
			},
			want: &entity.LoanApproval{
				LoanID:       loanID,
				ValidatorID:  validatorID,
				RejectReason: &rejectReason,
			},
		},
		{
			name: "RejectLoan_Failure_LoanNotFound",
			args: args{
				rejectionRequest: entity.RequestRejectLoan{
					LoanID:       loanID,
					RejectReason: rejectReason,
				},
				validatorID: validatorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs(loanID, constants.StatusProposed, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
		},
		{
			name: "RejectLoan_Failure_DBError_UpdateLoan",
			args: args{
				rejectionRequest: entity.RequestRejectLoan{
					LoanID:       loanID,
					RejectReason: rejectReason,
				},
				validatorID: validatorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs(loanID, constants.StatusProposed, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(loanID, constants.StatusProposed))
				mockSql.ExpectBegin()
				mockSql.ExpectExec(regexp.QuoteMeta(
					`UPDATE "loans"`)).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), constants.StatusRejected, sqlmock.AnyArg(), loanID,
					).
					WillReturnError(fmt.Errorf("DB error on update loan"))
				mockSql.ExpectRollback()
			},
			wantErr: fmt.Errorf("DB error on update loan"),
		},
		{
			name: "RejectLoan_Failure_DBError_InsertApproval",
			args: args{
				rejectionRequest: entity.RequestRejectLoan{
					LoanID:       loanID,
					RejectReason: rejectReason,
				},
				validatorID: validatorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs(loanID, constants.StatusProposed, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(loanID, constants.StatusProposed))
				mockSql.ExpectBegin()
				mockSql.ExpectExec(regexp.QuoteMeta(
					`UPDATE "loans"`)).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), constants.StatusRejected, sqlmock.AnyArg(), loanID,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "loan_approvals"`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						loanID,
						validatorID,
						rejectReason,
						"",
						sqlmock.AnyArg(),
					).
					WillReturnError(fmt.Errorf("DB error on insert approval"))
				mockSql.ExpectRollback()
			},
			wantErr: fmt.Errorf("DB error on insert approval"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mockSql := setupMockDB(t)
			redis, mockRedis := redismock.NewClientMock()
			u := usecase.NewLoanUsecase(db, redis)

			if tt.mockFunc != nil {
				tt.mockFunc(mockSql, mockRedis)
			}

			got, err := u.RejectLoan(tt.args.rejectionRequest, tt.args.validatorID)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.args.rejectionRequest.LoanID, got.LoanID)
				assert.Equal(t, tt.args.validatorID, got.ValidatorID)
				assert.Equal(t, tt.args.rejectionRequest.RejectReason, *got.RejectReason)
			}
		})
	}
}

func TestLoanUsecase_ApproveLoan(t *testing.T) {
	loanID := uint(1)
	validatorID := uint(2)
	agreementLink := "https://example.com/loan_agreement.pdf"
	photoURL := "https://example.com/photo.jpg"
	approvalID := uint(3)
	type args struct {
		approvalRequest entity.RequestApproveLoan
		validatorID     uint
	}
	tests := []struct {
		name     string
		args     args
		mockFunc func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock)
		want     *entity.LoanApproval
		wantErr  error
	}{
		{
			name: "ApproveLoan_Success",
			args: args{
				approvalRequest: entity.RequestApproveLoan{
					LoanID:   1,
					PhotoURL: photoURL,
				},
				validatorID: validatorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs(1, constants.StatusProposed, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status", "agreement_link"}).AddRow(loanID, constants.StatusProposed, agreementLink))
				mockSql.ExpectBegin()
				mockSql.ExpectExec(regexp.QuoteMeta(
					`UPDATE "loans"`)).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), constants.StatusApproved, agreementLink, loanID,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "loan_approvals"`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						loanID,
						validatorID,
						nil,
						photoURL,
						sqlmock.AnyArg(),
					).
					WillReturnRows(sqlmock.NewRows([]string{"id", "loan_id", "validator_id", "photo_url"}).AddRow(approvalID, loanID, validatorID, photoURL))
				mockSql.ExpectCommit()
			},
			want: &entity.LoanApproval{
				DBCommon: entity.DBCommon{
					ID: approvalID,
				},
				LoanID:      loanID,
				ValidatorID: validatorID,
				PhotoURL:    photoURL,
			},
		},
		{
			name: "ApproveLoan_Failure_LoanNotFound",
			args: args{
				approvalRequest: entity.RequestApproveLoan{
					LoanID:   1,
					PhotoURL: photoURL,
				},
				validatorID: validatorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs(1, constants.StatusProposed, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
		},
		{
			name: "ApproveLoan_Failure_DBError_UpdateLoan",
			args: args{
				approvalRequest: entity.RequestApproveLoan{
					LoanID:   1,
					PhotoURL: photoURL,
				},
				validatorID: validatorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs(1, constants.StatusProposed, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status", "agreement_link"}).AddRow(loanID, constants.StatusProposed, agreementLink))
				mockSql.ExpectBegin()
				mockSql.ExpectExec(regexp.QuoteMeta(
					`UPDATE "loans"`)).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), constants.StatusApproved, agreementLink, loanID,
					).
					WillReturnError(fmt.Errorf("DB error on update loan"))
				mockSql.ExpectRollback()
			},
			wantErr: fmt.Errorf("DB error on update loan"),
		},
		{
			name: "ApproveLoan_Failure_DBError_InsertApproval",
			args: args{
				approvalRequest: entity.RequestApproveLoan{
					LoanID:   1,
					PhotoURL: photoURL,
				},
				validatorID: validatorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs(1, constants.StatusProposed, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status", "agreement_link"}).AddRow(loanID, constants.StatusProposed, agreementLink))
				mockSql.ExpectBegin()
				mockSql.ExpectExec(regexp.QuoteMeta(
					`UPDATE "loans"`)).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), constants.StatusApproved, agreementLink, loanID,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "loan_approvals"`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						loanID,
						validatorID,
						nil,
						photoURL,
						sqlmock.AnyArg(),
					).
					WillReturnError(fmt.Errorf("DB error on insert approval"))
				mockSql.ExpectRollback()
			},
			wantErr: fmt.Errorf("DB error on insert approval"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mockSql := setupMockDB(t)
			redis, mockRedis := redismock.NewClientMock()
			u := usecase.NewLoanUsecase(db, redis)
			if tt.mockFunc != nil {
				tt.mockFunc(mockSql, mockRedis)
			}
			got, err := u.ApproveLoan(tt.args.approvalRequest, tt.args.validatorID)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.want.LoanID, got.LoanID)
				assert.Equal(t, tt.want.ValidatorID, got.ValidatorID)
				assert.Equal(t, tt.want.PhotoURL, got.PhotoURL)
				assert.Equal(t, tt.want.ID, got.ID)
			}
		})
	}
}

func TestLoanUsecase_AddInvestment(t *testing.T) {
	loanID := uint(1)
	investorID := uint(2)
	principal := 1000.0
	amount := 500.0
	type args struct {
		ctx               context.Context
		investmentRequest entity.RequestAddInvestment
		investorID        uint
	}
	tests := []struct {
		name     string
		args     args
		mockFunc func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock)
		want     *entity.Investment
		wantErr  error
	}{
		{
			name: "successfully add investment to match principal and change loan status to invested",
			args: args{
				ctx: context.Background(),
				investmentRequest: entity.RequestAddInvestment{
					LoanID: loanID,
					Amount: amount,
				},
				investorID: investorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				lockKey := fmt.Sprintf("event_lock:%d", loanID)
				mockRedis.ExpectSetNX(lockKey, "locked", 5*time.Second).SetVal(true)
				mockRedis.ExpectDel(lockKey).SetVal(1)

				mockSql.ExpectBegin()
				mockSql.ExpectQuery(`SELECT .* FROM "loans"`).
					WithArgs(loanID, constants.StatusApproved, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "principal", "status"}).
						AddRow(loanID, principal, constants.StatusApproved))

				// Mock loan.Investments preload (empty in this test)
				mockSql.ExpectQuery(`SELECT .* FROM "investments"`).
					WithArgs(loanID).
					WillReturnRows(sqlmock.NewRows([]string{"id", "loan_id", "amount", "investor_id"}).
						AddRow(1, loanID, principal-amount, investorID))

				mockSql.ExpectQuery(`INSERT INTO "investments"`).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						loanID,
						investorID,
						amount,
					).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

				mockSql.ExpectExec(`UPDATE "loans"`).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						principal,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						constants.StatusInvested,
						sqlmock.AnyArg(),
						loanID,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mockSql.ExpectQuery(`INSERT INTO "investments"`).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						loanID,
						investorID,
						amount,
						1,
					).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

				mockSql.ExpectCommit()
			},
			want: &entity.Investment{
				LoanID:     loanID,
				InvestorID: investorID,
				Amount:     amount,
			},
			wantErr: nil,
		},
		{
			name: "add investment below principal but not enough to change loan status",
			args: args{
				ctx: context.Background(),
				investmentRequest: entity.RequestAddInvestment{
					LoanID: loanID,
					Amount: amount,
				},
				investorID: investorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				lockKey := fmt.Sprintf("event_lock:%d", loanID)
				mockRedis.ExpectSetNX(lockKey, "locked", 5*time.Second).SetVal(true)
				mockRedis.ExpectDel(lockKey).SetVal(1)

				mockSql.ExpectBegin()
				mockSql.ExpectQuery(`SELECT .* FROM "loans"`).
					WithArgs(loanID, constants.StatusApproved, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "principal", "status"}).
						AddRow(loanID, principal, constants.StatusApproved))

				// Mock loan.Investments preload (empty in this test)
				mockSql.ExpectQuery(`SELECT .* FROM "investments"`).
					WithArgs(loanID).
					WillReturnRows(sqlmock.NewRows([]string{"id", "loan_id", "amount", "investor_id"}))

				mockSql.ExpectQuery(`INSERT INTO "investments"`).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						loanID,
						investorID,
						amount,
					).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

				mockSql.ExpectCommit()
			},
			want: &entity.Investment{
				LoanID:     loanID,
				InvestorID: investorID,
				Amount:     amount,
			},
		},
		{
			name: "failure due to investment exceeding principal",
			args: args{
				ctx: context.Background(),
				investmentRequest: entity.RequestAddInvestment{
					LoanID: loanID,
					Amount: principal + 100,
				},
				investorID: investorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				lockKey := fmt.Sprintf("event_lock:%d", loanID)
				mockRedis.ExpectSetNX(lockKey, "locked", 5*time.Second).SetVal(true)
				mockRedis.ExpectDel(lockKey).SetVal(1)

				mockSql.ExpectBegin()
				mockSql.ExpectQuery(`SELECT .* FROM "loans"`).
					WithArgs(loanID, constants.StatusApproved, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "principal", "status"}).
						AddRow(loanID, principal, constants.StatusApproved))

				// Mock loan.Investments preload (empty in this test)
				mockSql.ExpectQuery(`SELECT .* FROM "investments"`).
					WithArgs(loanID).
					WillReturnRows(sqlmock.NewRows([]string{"id", "loan_id", "amount", "investor_id"}))

				mockSql.ExpectRollback()
			},
			wantErr: fmt.Errorf(errs.ErrInvestmentExceedsPrincipal),
		},
		{
			name: "failure due to lock acquisition error",
			args: args{
				ctx: context.Background(),
				investmentRequest: entity.RequestAddInvestment{
					LoanID: loanID,
					Amount: amount,
				},
				investorID: investorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				lockKey := fmt.Sprintf("event_lock:%d", loanID)
				mockRedis.ExpectSetNX(lockKey, "locked", 5*time.Second).SetErr(assert.AnError)
			},
			wantErr: fmt.Errorf(errs.ErrLockAcquisitionFailed),
		},
		{
			name: "failure due to failed to acquire lock",
			args: args{
				ctx: context.Background(),
				investmentRequest: entity.RequestAddInvestment{
					LoanID: loanID,
					Amount: amount,
				},
				investorID: investorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				lockKey := fmt.Sprintf("event_lock:%d", loanID)
				mockRedis.ExpectSetNX(lockKey, "locked", 5*time.Second).SetVal(false)
			},
			wantErr: fmt.Errorf(errs.ErrBusySystem),
		},
		{
			name: "failure due to DB error on getting loan",
			args: args{
				ctx: context.Background(),
				investmentRequest: entity.RequestAddInvestment{
					LoanID: loanID,
					Amount: amount,
				},
				investorID: investorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				lockKey := fmt.Sprintf("event_lock:%d", loanID)
				mockRedis.ExpectSetNX(lockKey, "locked", 5*time.Second).SetVal(true)
				mockRedis.ExpectDel(lockKey).SetVal(1)

				mockSql.ExpectBegin()
				mockSql.ExpectQuery(`SELECT .* FROM "loans"`).
					WithArgs(loanID, constants.StatusApproved, 1).
					WillReturnError(fmt.Errorf("DB error on getting loan"))
				mockSql.ExpectRollback()
			},
			wantErr: fmt.Errorf("DB error on getting loan"),
		},
		{
			name: "failure due to DB error on creating investment",
			args: args{
				ctx: context.Background(),
				investmentRequest: entity.RequestAddInvestment{
					LoanID: loanID,
					Amount: amount,
				},
				investorID: investorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				lockKey := fmt.Sprintf("event_lock:%d", loanID)
				mockRedis.ExpectSetNX(lockKey, "locked", 5*time.Second).SetVal(true)
				mockRedis.ExpectDel(lockKey).SetVal(1)

				mockSql.ExpectBegin()
				mockSql.ExpectQuery(`SELECT .* FROM "loans"`).
					WithArgs(loanID, constants.StatusApproved, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "principal", "status"}).
						AddRow(loanID, principal, constants.StatusApproved))

				// Mock loan.Investments preload (empty in this test)
				mockSql.ExpectQuery(`SELECT .* FROM "investments"`).
					WithArgs(loanID).
					WillReturnRows(sqlmock.NewRows([]string{"id", "loan_id", "amount", "investor_id"}))

				mockSql.ExpectQuery(`INSERT INTO "investments"`).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						loanID,
						investorID,
						amount,
					).WillReturnError(fmt.Errorf("DB error on creating investment"))

				mockSql.ExpectRollback()
			},
			wantErr: fmt.Errorf("DB error on creating investment"),
		},
		{
			name: "failure due to DB error on updating loan status",
			args: args{
				ctx: context.Background(),
				investmentRequest: entity.RequestAddInvestment{
					LoanID: loanID,
					Amount: principal,
				},
				investorID: investorID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				lockKey := fmt.Sprintf("event_lock:%d", loanID)
				mockRedis.ExpectSetNX(lockKey, "locked", 5*time.Second).SetVal(true)
				mockRedis.ExpectDel(lockKey).SetVal(1)

				mockSql.ExpectBegin()
				mockSql.ExpectQuery(`SELECT .* FROM "loans"`).
					WithArgs(loanID, constants.StatusApproved, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "principal", "status"}).
						AddRow(loanID, principal, constants.StatusApproved))

				// Mock loan.Investments preload (empty in this test)
				mockSql.ExpectQuery(`SELECT .* FROM "investments"`).
					WithArgs(loanID).
					WillReturnRows(sqlmock.NewRows([]string{"id", "loan_id", "amount", "investor_id"}))

				mockSql.ExpectQuery(`INSERT INTO "investments"`).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						loanID,
						investorID,
						principal,
					).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

				mockSql.ExpectExec(`UPDATE "loans"`).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						principal,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						constants.StatusInvested,
						sqlmock.AnyArg(),
						loanID,
					).
					WillReturnError(fmt.Errorf("DB error on updating loan status"))

				mockSql.ExpectRollback()
			},
			wantErr: fmt.Errorf("DB error on updating loan status"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mockSql := setupMockDB(t)
			redis, mockRedis := redismock.NewClientMock()
			u := usecase.NewLoanUsecase(db, redis)
			if tt.mockFunc != nil {
				tt.mockFunc(mockSql, mockRedis)
			}
			got, err := u.AddInvestment(tt.args.ctx, tt.args.investmentRequest, tt.args.investorID)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.want.LoanID, got.LoanID)
				assert.Equal(t, tt.want.InvestorID, got.InvestorID)
				assert.Equal(t, tt.want.Amount, got.Amount)
			}
			assert.NoError(t, mockSql.ExpectationsWereMet())
			assert.NoError(t, mockRedis.ExpectationsWereMet())
		})
	}
}

func TestLoanUsecase_DisburseLoan(t *testing.T) {
	loanID := uint(1)
	disburserID := uint(2)
	signedAgreementURL := "https://example.com/signed_agreement.pdf"
	disbursementID := uint(3)
	type args struct {
		disbursementRequest entity.RequestDisburseLoan
		disburserID         uint
	}
	tests := []struct {
		name     string
		args     args
		mockFunc func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock)
		want     *entity.LoanDisbursement
		wantErr  error
	}{
		{
			name: "DisburseLoan_Success",
			args: args{
				disbursementRequest: entity.RequestDisburseLoan{
					LoanID:             loanID,
					SignedAgreementURL: signedAgreementURL,
				},
				disburserID: disburserID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs(loanID, constants.StatusInvested, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(loanID, constants.StatusInvested))
				mockSql.ExpectBegin()
				mockSql.ExpectExec(regexp.QuoteMeta(
					`UPDATE "loans"`)).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), constants.StatusDisbursed, sqlmock.AnyArg(), loanID,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "loan_disbursements"`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						loanID,
						signedAgreementURL,
						disburserID,
						sqlmock.AnyArg(),
					).
					WillReturnRows(sqlmock.NewRows([]string{"id", "loan_id", "signed_agreement_url", "disburser_id"}).AddRow(disbursementID, loanID, signedAgreementURL, disburserID))
				mockSql.ExpectCommit()
			},
			want: &entity.LoanDisbursement{
				DBCommon:           entity.DBCommon{ID: disbursementID},
				LoanID:             loanID,
				SignedAgreementURL: signedAgreementURL,
				DisburserID:        disburserID,
			},
		},
		{
			name: "DisburseLoan_Failure_LoanNotFound",
			args: args{
				disbursementRequest: entity.RequestDisburseLoan{
					LoanID:             loanID,
					SignedAgreementURL: signedAgreementURL,
				},
				disburserID: disburserID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs(loanID, constants.StatusInvested, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
		},
		{
			name: "DisburseLoan_Failure_DBError_UpdateLoan",
			args: args{
				disbursementRequest: entity.RequestDisburseLoan{
					LoanID:             loanID,
					SignedAgreementURL: signedAgreementURL,
				},
				disburserID: disburserID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs(loanID, constants.StatusInvested, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(loanID, constants.StatusInvested))
				mockSql.ExpectBegin()
				mockSql.ExpectExec(regexp.QuoteMeta(
					`UPDATE "loans"`)).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), constants.StatusDisbursed, sqlmock.AnyArg(), loanID,
					).
					WillReturnError(fmt.Errorf("DB error on update loan"))
				mockSql.ExpectRollback()
			},
			wantErr: fmt.Errorf("DB error on update loan"),
		},
		{
			name: "DisburseLoan_Failure_DBError_InsertDisbursement",
			args: args{
				disbursementRequest: entity.RequestDisburseLoan{
					LoanID:             loanID,
					SignedAgreementURL: signedAgreementURL,
				},
				disburserID: disburserID,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs(loanID, constants.StatusInvested, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(loanID, constants.StatusInvested))
				mockSql.ExpectBegin()
				mockSql.ExpectExec(regexp.QuoteMeta(
					`UPDATE "loans"`)).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), constants.StatusDisbursed, sqlmock.AnyArg(), loanID,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "loan_disbursements"`)).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						loanID,
						signedAgreementURL,
						disburserID,
						sqlmock.AnyArg(),
					).
					WillReturnError(fmt.Errorf("DB error on insert disbursement"))
				mockSql.ExpectRollback()
			},
			wantErr: fmt.Errorf("DB error on insert disbursement"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mockSql := setupMockDB(t)
			redis, mockRedis := redismock.NewClientMock()
			u := usecase.NewLoanUsecase(db, redis)
			if tt.mockFunc != nil {
				tt.mockFunc(mockSql, mockRedis)
			}
			got, err := u.DisburseLoan(tt.args.disbursementRequest, tt.args.disburserID)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.want.LoanID, got.LoanID)
				assert.Equal(t, tt.want.DisburserID, got.DisburserID)
				assert.Equal(t, tt.want.ID, got.ID)
				assert.Equal(t, tt.want.SignedAgreementURL, got.SignedAgreementURL)
			}
		})
	}
}

func TestLoanUsecase_GetLoan(t *testing.T) {
	type args struct {
		loanID string
	}
	tests := []struct {
		name     string
		args     args
		mockFunc func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock)
		want     *entity.Loan
		wantErr  error
	}{
		{
			name: "GetLoan_Success",
			args: args{
				loanID: "1",
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs("1", 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status", "principal"}).
						AddRow(1, constants.StatusDisbursed, 1000))
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loan_approvals"`)).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"loan_id", "validator_id", "reject_reason"}).
						AddRow(1, 2, nil))
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loan_disbursements"`)).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"loan_id", "signed_agreement_url", "disburser_id"}).
						AddRow(1, "https://example.com/signed_agreement.pdf", 3))
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "investments"`)).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "loan_id", "investor_id", "amount"}).AddRow(1, 1, 4, 1000))
			},
			want: &entity.Loan{
				DBCommon: entity.DBCommon{
					ID: 1,
				},
				Status:    constants.StatusDisbursed,
				Principal: 1000,
				ApprovedInfo: &entity.LoanApproval{
					LoanID:       1,
					ValidatorID:  2,
					RejectReason: nil,
				},
				DisbursementInfo: &entity.LoanDisbursement{
					LoanID:             1,
					SignedAgreementURL: "https://example.com/signed_agreement.pdf",
					DisburserID:        3,
				},
				Investments: []entity.Investment{
					{
						DBCommon: entity.DBCommon{
							ID: 1,
						},
						LoanID:     1,
						InvestorID: 4,
						Amount:     1000,
					},
				},
			},
		},
		{
			name: "GetLoan_Failure_LoanNotFound",
			args: args{
				loanID: "1",
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "loans"`)).
					WithArgs("1", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: gorm.ErrRecordNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mockSql := setupMockDB(t)
			redis, mockRedis := redismock.NewClientMock()
			u := usecase.NewLoanUsecase(db, redis)
			if tt.mockFunc != nil {
				tt.mockFunc(mockSql, mockRedis)
			}
			got, err := u.GetLoan(tt.args.loanID)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.want.ID, got.ID)
				assert.Equal(t, tt.want.Status, got.Status)
				assert.Equal(t, tt.want.Principal, got.Principal)
				if tt.want.ApprovedInfo != nil {
					assert.NotNil(t, got.ApprovedInfo)
					assert.Equal(t, tt.want.ApprovedInfo.LoanID, got.ApprovedInfo.LoanID)
					assert.Equal(t, tt.want.ApprovedInfo.ValidatorID, got.ApprovedInfo.ValidatorID)
					assert.Equal(t, tt.want.ApprovedInfo.RejectReason, got.ApprovedInfo.RejectReason)
				} else {
					assert.Nil(t, got.ApprovedInfo)
				}
				if tt.want.DisbursementInfo != nil {
					assert.NotNil(t, got.DisbursementInfo)
					assert.Equal(t, tt.want.DisbursementInfo.LoanID, got.DisbursementInfo.LoanID)
					assert.Equal(t, tt.want.DisbursementInfo.SignedAgreementURL, got.DisbursementInfo.SignedAgreementURL)
					assert.Equal(t, tt.want.DisbursementInfo.DisburserID, got.DisbursementInfo.DisburserID)
				} else {
					assert.Nil(t, got.DisbursementInfo)
				}
				if len(tt.want.Investments) > 0 {
					assert.Len(t, got.Investments, len(tt.want.Investments))
					for i := range tt.want.Investments {
						assert.Equal(t, tt.want.Investments[i].ID, got.Investments[i].ID)
						assert.Equal(t, tt.want.Investments[i].LoanID, got.Investments[i].LoanID)
						assert.Equal(t, tt.want.Investments[i].InvestorID, got.Investments[i].InvestorID)
						assert.Equal(t, tt.want.Investments[i].Amount, got.Investments[i].Amount)
					}
				} else {
					assert.Empty(t, got.Investments)
				}
			}
		})
	}
}
