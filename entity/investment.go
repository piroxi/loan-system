package entity

type Investment struct {
	DBCommon
	LoanID     uint    `json:"loan_id"`
	InvestorID uint    `json:"investor_id"`
	Amount     float64 `json:"amount"`
}
