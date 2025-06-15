package entity

import (
	"loan-service/utils/constants"
	"time"
)

type Loan struct {
	DBCommon
	BorrowerID    uint                 `json:"borrower_id"`
	Principal     float64              `json:"principal"`
	Rate          float64              `json:"rate"`
	ROI           float64              `json:"roi"`
	Status        constants.LoanStatus `json:"status"`
	AgreementLink *string              `json:"agreement_link,omitempty"`

	ApprovedInfo     *LoanApproval     `gorm:"foreignKey:LoanID" json:"approved_info,omitempty"`
	DisbursementInfo *LoanDisbursement `gorm:"foreignKey:LoanID" json:"disbursement_info,omitempty"`
	Investments      []Investment      `gorm:"foreignKey:LoanID" json:"investments",omitempty`
}

type LoanApproval struct {
	DBCommon
	LoanID       uint      `json:"loan_id"`
	ValidatorID  uint      `json:"validator_id"`
	RejectReason *string   `json:"reject_reason,omitempty"`
	PhotoURL     string    `json:"photo_url"`
	ApprovedAt   time.Time `json:"approved_at"`
}

type LoanDisbursement struct {
	DBCommon
	LoanID             uint      `json:"loan_id"`
	SignedAgreementURL string    `json:"signed_agreement_url"`
	DisburserID        uint      `json:"disburser_id"`
	DisbursedAt        time.Time `json:"disbursed_at"`
}
