package usecase

import (
	"loan-service/entity"
	"loan-service/utils/constants"
	"loan-service/utils/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserUsecase struct {
	db *gorm.DB
}

func NewUserUsecase(db *gorm.DB) *UserUsecase {
	return &UserUsecase{db: db}
}

func (u *UserUsecase) GetUserByUsername(username string) (*entity.User, error) {
	var user entity.User
	if err := u.db.Where("username = ?", username).First(&user).Error; err != nil {
		logger.Error("Failed to fetch user by username", zap.String("username", username), zap.Error(err))
		return nil, err
	}
	return &user, nil
}

func (u UserUsecase) GetUserRole(userID uint) (constants.UserRole, error) {
	var user entity.User
	if err := u.db.First(&user, userID).Error; err != nil {
		logger.Error("Failed to fetch user by ID", zap.Uint("userID", userID), zap.Error(err))
		return constants.RoleUnknown, err
	}

	return constants.RoleMap[user.Role], nil
}
