package handler

import (
	"loan-service/entity"
	"loan-service/utils/constants"
	errs "loan-service/utils/errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LoanHandler struct {
	loanUsecase LoanUsecaseInterface
	userUsecase UserUsecaseInterface
}

func RegisterLoanHandler(r *gin.RouterGroup, loanUsecase LoanUsecaseInterface, userUsecase UserUsecaseInterface) {
	h := &LoanHandler{loanUsecase: loanUsecase, userUsecase: userUsecase}
	g := r.Group("/loans", authMiddleware())

	g.POST("/create", h.createLoan)
	g.GET("/:id", h.getLoan)
	g.POST("/reject", h.rejectLoan)
	g.POST("/approve", h.approveLoan)
	g.POST("/invest", h.addInvestment)
	g.POST("/disburse", h.disburseLoan)
}

func (h *LoanHandler) getLoan(c *gin.Context) {
	id := c.Param("id")
	loan, err := h.loanUsecase.GetLoan(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errs.ErrLoanNotFound})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": loan})
}

func (h *LoanHandler) createLoan(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	if !h.verifyUserRole(userID, constants.RoleBorrower) {
		c.JSON(http.StatusForbidden, gin.H{"error": errs.ErrUnauthorizedAction})
		return
	}

	var input entity.RequestProposeLoan
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Principal <= 0 || input.ROI <= 0 || input.Rate <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid loan parameters"})
		return
	}

	loan, err := h.loanUsecase.CreateLoan(input, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": loan})
}

func (h *LoanHandler) rejectLoan(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	if !h.verifyUserRole(userID, constants.RoleValidator) {
		c.JSON(http.StatusForbidden, gin.H{"error": errs.ErrUnauthorizedAction})
		return
	}

	var input entity.RequestRejectLoan
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rejection, err := h.loanUsecase.RejectLoan(input, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rejection})
}

func (h *LoanHandler) approveLoan(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	if !h.verifyUserRole(userID, constants.RoleValidator) {
		c.JSON(http.StatusForbidden, gin.H{"error": errs.ErrUnauthorizedAction})
		return
	}

	var input entity.RequestApproveLoan
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	approval, err := h.loanUsecase.ApproveLoan(input, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": approval})
}

func (h *LoanHandler) addInvestment(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	if !h.verifyUserRole(userID, constants.RoleInvestor) {
		c.JSON(http.StatusForbidden, gin.H{"error": errs.ErrUnauthorizedAction})
		return
	}

	var input entity.RequestAddInvestment
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: LoanID and Amount are required"})
		return
	}

	investment, err := h.loanUsecase.AddInvestment(c, input, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": investment})
}

func (h *LoanHandler) disburseLoan(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	if !h.verifyUserRole(userID, constants.RoleDisburser) {
		c.JSON(http.StatusForbidden, gin.H{"error": errs.ErrUnauthorizedAction})
		return
	}

	var input entity.RequestDisburseLoan
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	disbursement, err := h.loanUsecase.DisburseLoan(input, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": disbursement})
}
