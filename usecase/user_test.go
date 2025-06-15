package usecase_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"loan-service/usecase"
	"loan-service/utils/constants"
)

const username = "testuser"
const userID = 1
const roleID = 0

func TestGetUserByUsername_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	uc := usecase.NewUserUsecase(db)

	mock.ExpectQuery(`SELECT \* FROM "users" WHERE username = \$1 ORDER BY "users"\."id" LIMIT \$2`).
		WithArgs(username, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}).
			AddRow(userID, username, roleID))

	user, err := uc.GetUserByUsername(username)
	assert.NoError(t, err)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, uint(1), user.ID)
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	uc := usecase.NewUserUsecase(db)

	mock.ExpectQuery(`SELECT \* FROM "users" WHERE username = \$1 ORDER BY "users"\."id" LIMIT \$2`).
		WithArgs("notfound", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	user, err := uc.GetUserByUsername("notfound")
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestGetUserRole_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	uc := usecase.NewUserUsecase(db)

	mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1 ORDER BY "users"\."id" LIMIT \$2`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}).
			AddRow(userID, username, roleID))

	role, err := uc.GetUserRole(userID)
	assert.NoError(t, err)
	assert.Equal(t, constants.RoleMap[roleID], role)
}

func TestGetUserRole_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	uc := usecase.NewUserUsecase(db)

	mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1 ORDER BY "users"\."id" LIMIT \$2`).
		WithArgs(99, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	role, err := uc.GetUserRole(99)
	assert.Error(t, err)
	assert.Equal(t, constants.RoleUnknown, role)
}
