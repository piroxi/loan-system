package usecase_test

import (
	"load-service/entity"
	"load-service/usecase"
	"load-service/utils/constants"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestLoanUsecase_CreateLoan(t *testing.T) {
	type args struct {
		loanRequest entity.RequestProposeLoan
		borrowerID  uint
	}
	tests := []struct {
		name     string
		args     args
		mockFunc func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock)
		want     *entity.Loan
		wantErr  error
	}{
		{
			name: "CreateLoan_Success",
			args: args{
				loanRequest: entity.RequestProposeLoan{
					Principal: 1000,
					ROI:       5,
					Rate:      10,
				},
				borrowerID: 1,
			},
			mockFunc: func(mockSql sqlmock.Sqlmock, mockRedis redismock.ClientMock) {
				mockSql.ExpectBegin()
				mockSql.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "loans"`)).
					WithArgs(
						1000,
						5,
						10,
						1,
						constants.StatusProposed,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mockSql.ExpectCommit()
			},
			want: &entity.Loan{
				DBCommon: entity.DBCommon{
					ID: 1,
				},
				Principal:  1000,
				ROI:        5,
				Rate:       10,
				BorrowerID: 1,
				Status:     constants.StatusProposed,
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB, mockSql, _ := sqlmock.New()
			redis, mockRedis := redismock.NewClientMock()

			dialector := postgres.New(postgres.Config{
				DSN:                  "sqlmock_db_0",
				DriverName:           "postgres",
				Conn:                 mockDB,
				PreferSimpleProtocol: true,
			})
			db, _ := gorm.Open(dialector, &gorm.Config{})

			if tt.mockFunc != nil {
				tt.mockFunc(mockSql, mockRedis)
			}

			u := usecase.NewLoanUsecase(db, redis)
			got, err := u.CreateLoan(tt.args.loanRequest, tt.args.borrowerID)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
