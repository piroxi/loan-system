package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"loan-service/entity"
	"loan-service/utils/constants"
	errs "loan-service/utils/errors"
	"loan-service/utils/logger"
	"time"

	"codeberg.org/go-pdf/fpdf"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type LoanUsecase struct {
	db          *gorm.DB
	redisClient *redis.Client
}

func NewLoanUsecase(db *gorm.DB, redisClient *redis.Client) *LoanUsecase {
	return &LoanUsecase{
		db:          db,
		redisClient: redisClient,
	}
}

func (u *LoanUsecase) CreateLoan(loanRequest entity.RequestProposeLoan, borrowerID uint) (*entity.Loan, error) {
	tx := u.db.Begin()
	defer tx.Rollback()

	loan := entity.Loan{
		Principal:  loanRequest.Principal,
		ROI:        loanRequest.ROI,
		Rate:       loanRequest.Rate,
		BorrowerID: borrowerID,
		Status:     constants.StatusProposed,
	}
	if err := tx.Create(&loan).Error; err != nil {
		return nil, err
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, fmt.Sprintf("Loan Proposal: ID %d", loan.ID))
	pdf.Ln(10)
	pdf.Cell(0, 10, fmt.Sprintf("Principal: %.2f", loan.Principal))
	pdf.Ln(10)
	pdf.Cell(0, 10, fmt.Sprintf("ROI: %.2f%%", loan.ROI))
	pdf.Ln(10)
	pdf.Cell(0, 10, fmt.Sprintf("Rate: %.2f%%", loan.Rate))
	pdf.Ln(10)
	pdf.Cell(0, 10, fmt.Sprintf("Borrower ID: %d", loan.BorrowerID))
	pdf.Ln(10)
	pdfFileName := fmt.Sprintf("loan_proposal_%d.pdf", loan.ID)
	pdf.Ln(10)
	if err := pdf.OutputFileAndClose(pdfFileName); err != nil {
		logger.Error("Failed to create PDF", zap.Error(err))
		return nil, errors.New("failed to create PDF document")
	}
	agreementLink := fmt.Sprintf("https://example.com/loans/%d/%s", loan.ID, pdfFileName)
	loan.AgreementLink = &agreementLink
	if err := tx.Save(&loan).Error; err != nil {
		logger.Error("Failed to save loan with PDF URL", zap.Error(err))
		return nil, errors.New("failed to save loan with PDF URL")
	}

	tx.Commit()

	logger.Info("Loan created successfully", zap.Uint("loanID", loan.ID))

	return &loan, nil
}

func (u *LoanUsecase) RejectLoan(rejectionRequest entity.RequestRejectLoan, validatorID uint) (*entity.LoanApproval, error) {
	var loan entity.Loan
	if err := u.db.First(&loan, "id = ? AND status = ?", rejectionRequest.LoanID, constants.StatusProposed).Error; err != nil {
		return nil, err
	}
	tx := u.db.Begin(&sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	defer tx.Rollback()

	loan.Status = constants.StatusRejected
	if err := tx.Save(&loan).Error; err != nil {
		return nil, err
	}

	rejection := entity.LoanApproval{
		LoanID:       loan.ID,
		RejectReason: &rejectionRequest.RejectReason,
		ValidatorID:  validatorID,
	}
	if err := tx.Create(&rejection).Error; err != nil {
		return nil, err
	}

	tx.Commit()

	return &rejection, nil
}

func (u *LoanUsecase) ApproveLoan(approvalRequest entity.RequestApproveLoan, validatorID uint) (*entity.LoanApproval, error) {
	var loan entity.Loan
	if err := u.db.First(&loan, "id = ? AND status = ?", approvalRequest.LoanID, constants.StatusProposed).Error; err != nil {
		return nil, err
	}
	tx := u.db.Begin(&sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	defer tx.Rollback()

	loan.Status = constants.StatusApproved
	if err := tx.Save(&loan).Error; err != nil {
		return nil, err
	}

	approval := entity.LoanApproval{
		LoanID:      loan.ID,
		ValidatorID: validatorID,
		PhotoURL:    approvalRequest.PhotoURL,
		ApprovedAt:  time.Now(),
	}
	if err := tx.Create(&approval).Error; err != nil {
		return nil, err
	}

	tx.Commit()

	return &approval, nil
}

func (u *LoanUsecase) AddInvestment(
	ctx context.Context,
	investmentRequest entity.RequestAddInvestment,
	investorID uint,
) (*entity.Investment, error) {
	var loan entity.Loan
	lockKey := fmt.Sprintf("event_lock:%d", investmentRequest.LoanID)
	locked, err := u.redisClient.SetNX(ctx, lockKey, "locked", 5*time.Second).Result()
	if err != nil {
		return nil, errors.New(errs.ErrLockAcquisitionFailed)
	}
	if !locked {
		return nil, errors.New(errs.ErrBusySystem)
	}
	defer u.redisClient.Del(ctx, lockKey)

	tx := u.db.Begin(&sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	defer tx.Rollback()

	if err := tx.Preload("Investments").First(&loan, "id = ? AND status = ?", investmentRequest.LoanID, constants.StatusApproved).Error; err != nil {
		return nil, err
	}

	total := 0.0
	for _, inv := range loan.Investments {
		total += inv.Amount
	}
	if total+investmentRequest.Amount > loan.Principal {
		return nil, errors.New(errs.ErrInvestmentExceedsPrincipal)
	}

	investment := entity.Investment{
		LoanID:     investmentRequest.LoanID,
		InvestorID: investorID,
		Amount:     investmentRequest.Amount,
	}
	if err := tx.Create(&investment).Error; err != nil {
		return nil, err
	}
	if total+investment.Amount == loan.Principal {
		loan.Status = constants.StatusInvested
		if err := tx.Save(&loan).Error; err != nil {
			return nil, err
		}
	}

	tx.Commit()

	return &investment, nil
}

func (u *LoanUsecase) DisburseLoan(disbursementRequest entity.RequestDisburseLoan, disburserID uint) (*entity.LoanDisbursement, error) {
	var loan entity.Loan
	if err := u.db.First(&loan, "id = ? AND status = ?", disbursementRequest.LoanID, constants.StatusInvested).Error; err != nil {
		logger.Error("Failed to find loan for disbursement", zap.Uint("loanID", disbursementRequest.LoanID), zap.Error(err))
		return nil, err
	}
	loan.Status = constants.StatusDisbursed
	disbursementRequest.LoanID = loan.ID

	tx := u.db.Begin()
	defer tx.Rollback()

	if err := tx.Save(&loan).Error; err != nil {
		logger.Error("Failed to update loan status to disbursed", zap.Uint("loanID", disbursementRequest.LoanID), zap.Error(err))
		return nil, err
	}

	disbursement := entity.LoanDisbursement{
		LoanID:             disbursementRequest.LoanID,
		SignedAgreementURL: disbursementRequest.SignedAgreementURL,
		DisburserID:        disburserID,
		DisbursedAt:        time.Now(),
	}

	if err := tx.Create(&disbursement).Error; err != nil {
		logger.Error("Failed to create loan disbursement record", zap.Uint("loanID", disbursement.LoanID), zap.Error(err))
		return nil, err
	}

	tx.Commit()
	return &disbursement, nil
}

func (u *LoanUsecase) GetLoan(loanID string) (*entity.Loan, error) {
	var loan entity.Loan
	if err := u.db.Preload("ApprovedInfo").Preload("DisbursementInfo").Preload("Investments").First(&loan, "id = ?", loanID).Error; err != nil {
		logger.Error("Failed to fetch loan by ID", zap.String("loanID", loanID), zap.Error(err))
		return nil, err
	}
	return &loan, nil
}
