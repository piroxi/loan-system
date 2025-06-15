package handler

import (
	"context"
	"loan-service/entity"
	"loan-service/utils/constants"
)

type LoanUsecaseInterface interface {
	CreateLoan(loanRequest entity.RequestProposeLoan, borrowerID uint) (*entity.Loan, error)
	RejectLoan(rejectionRequest entity.RequestRejectLoan, validatorID uint) (*entity.LoanApproval, error)
	ApproveLoan(approvalRequest entity.RequestApproveLoan, validatorID uint) (*entity.LoanApproval, error)
	AddInvestment(ctx context.Context, investmentRequest entity.RequestAddInvestment, investorID uint) (*entity.Investment, error)
	DisburseLoan(disbursementRequest entity.RequestDisburseLoan, disburserID uint) (*entity.LoanDisbursement, error)
	GetLoan(loanID string) (*entity.Loan, error)
}

type UserUsecaseInterface interface {
	GetUserByUsername(username string) (*entity.User, error)
	GetUserRole(userID uint) (constants.UserRole, error)
}
