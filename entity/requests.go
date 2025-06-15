package entity

type RequestSignin struct {
	Username string `json:"username" binding:"required"`
}

type RequestProposeLoan struct {
	Principal float64 `json:"principal" binding:"required"`
	Rate      float64 `json:"rate" binding:"required"`
	ROI       float64 `json:"roi" binding:"required"`
}

type RequestApproveLoan struct {
	LoanID   uint   `json:"loan_id" binding:"required"`
	PhotoURL string `json:"photo_url" binding:"required"`
}

type RequestRejectLoan struct {
	LoanID       uint   `json:"loan_id" binding:"required"`
	RejectReason string `json:"reject_reason" binding:"required"`
}

type RequestAddInvestment struct {
	LoanID uint    `json:"loan_id" binding:"required"`
	Amount float64 `json:"amount" binding:"required"`
}

type RequestDisburseLoan struct {
	LoanID             uint   `json:"loan_id" binding:"required"`
	SignedAgreementURL string `json:"signed_agreement_url" binding:"required"`
}
