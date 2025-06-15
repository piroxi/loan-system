package constants

type LoanStatus string

const (
	StatusProposed  LoanStatus = "proposed"
	StatusApproved  LoanStatus = "approved"
	StatusRejected  LoanStatus = "rejected"
	StatusInvested  LoanStatus = "invested"
	StatusDisbursed LoanStatus = "disbursed"
)

type UserRole string

const (
	RoleAdmin     UserRole = "admin"
	RoleBorrower  UserRole = "borrower"
	RoleValidator UserRole = "validator"
	RoleInvestor  UserRole = "investor"
	RoleDisburser UserRole = "disburser"
	RoleUnknown   UserRole = "unknown"
)

var RoleMap = []UserRole{
	RoleAdmin,
	RoleBorrower,
	RoleValidator,
	RoleInvestor,
	RoleDisburser,
}
