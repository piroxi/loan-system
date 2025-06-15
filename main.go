package main

import (
	"fmt"
	"loan-service/entity"
	"loan-service/handler"
	"loan-service/usecase"
	"loan-service/utils/auth"
	"loan-service/utils/config"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	config.LoadConfig()

	Conf := config.Conf
	if Conf == (config.Config{}) {
		panic("Configuration is not loaded properly")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		Conf.DBHost, Conf.DBUser, Conf.DBPass, Conf.DBName, Conf.DBPort)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&entity.Loan{}, &entity.LoanApproval{}, &entity.Investment{}, &entity.LoanDisbursement{})

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", Conf.RedisHost, Conf.RedisPort),
		Password: "",
		DB:       0,
	})

	if Conf.AuthSecret == "" {
		panic("AUTH_SECRET is not set")
	}

	auth.StartAuthorizer(Conf.AuthSecret)
	g := gin.Default()
	r := g.Group("/api")

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	userUsecase := usecase.NewUserUsecase(db)
	loanUsecase := usecase.NewLoanUsecase(db, rdb)

	handler.RegisterLoanHandler(r, loanUsecase, userUsecase)
	handler.RegisterUserHandler(r, userUsecase)

	g.Run(":8080")
}
