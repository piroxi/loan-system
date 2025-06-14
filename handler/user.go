package handler

import (
	"load-service/entity"
	"load-service/utils/auth"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userUsecase UserUsecaseInterface
}

func RegisterUserHandler(r *gin.RouterGroup, userUsecase UserUsecaseInterface) {
	h := &UserHandler{userUsecase: userUsecase}
	g := r.Group("/users")

	g.POST("/signin", h.signin)
}

func (h *UserHandler) signin(c *gin.Context) {
	var body entity.RequestSignin
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	user, err := h.userUsecase.GetUserByUsername(body.Username)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch user"})
		return
	}

	token, err := auth.GenerateToken(user.Username, user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(200, gin.H{
		"data": gin.H{
			"user":  user,
			"token": token,
		},
	})
}
