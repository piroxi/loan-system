package handler

import (
	"loan-service/utils/auth"
	"loan-service/utils/constants"
	"loan-service/utils/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Response struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func (h *LoanHandler) verifyUserRole(userID uint, expectedRole constants.UserRole) bool {
	role, err := h.userUsecase.GetUserRole(userID)
	if err != nil {
		logger.Error("Failed to get user role", zap.Uint("userID", userID), zap.Error(err))
		return false
	}

	if role == constants.RoleAdmin {
		return true
	}

	if role != expectedRole {
		logger.Error("Unauthorized action for user role", zap.Uint("userID", userID), zap.String("role", string(role)))
		return false
	}
	return true
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		tokenString := token[len("Bearer "):]

		claims, err := auth.ClaimToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)

		c.Next()
	}
}
