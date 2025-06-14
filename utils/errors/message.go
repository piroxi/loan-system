package errs

const (
	// Error messages
	ErrInvestmentExceedsPrincipal = "Investment exceeds principal amount"
	ErrLoanNotFound               = "Loan not found"
	ErrLoanNotFoundApprover       = "Loan not found or already approved"
	ErrLockAcquisitionFailed      = "Failed to acquire lock for investment processing"
	ErrBusySystem                 = "System is busy, please try again later"
	ErrUserNotFound               = "Failed to find user"
	ErrUnauthorizedAction         = "Unauthorized action for the user role"

	//Authentication errors
	ErrAuthUninitialized = "Authorizer is not initialized"
	ErrInvalidToken      = "Invalid token provided"
)
