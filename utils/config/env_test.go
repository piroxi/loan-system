package config_test

import (
	"fmt"
	"loan-service/utils/config"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		want        config.Config
		wantErr     bool
		cleanupFunc func()
	}{
		{
			name: "successful load with all environment variables",
			envVars: map[string]string{
				"DB_HOST":     "localhost",
				"DB_PORT":     "5432",
				"DB_USER":     "user",
				"DB_PASS":     "password",
				"DB_NAME":     "dbname",
				"REDIS_HOST":  "redis",
				"REDIS_PORT":  "6379",
				"AUTH_SECRET": "secret",
			},
			want: config.Config{
				DBHost:     "localhost",
				DBPort:     "5432",
				DBUser:     "user",
				DBPass:     "password",
				DBName:     "dbname",
				RedisHost:  "redis",
				RedisPort:  "6379",
				AuthSecret: "secret",
			},
			wantErr: false,
			cleanupFunc: func() {
				os.Unsetenv("DB_HOST")
				os.Unsetenv("DB_PORT")
				os.Unsetenv("DB_USER")
				os.Unsetenv("DB_PASS")
				os.Unsetenv("DB_NAME")
				os.Unsetenv("REDIS_HOST")
				os.Unsetenv("REDIS_PORT")
				os.Unsetenv("AUTH_SECRET")
				fmt.Printf("db_host1: %s\n", os.Getenv("DB_HOST"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variables for the test
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Make sure we clean up after the test
			defer tt.cleanupFunc()

			// Execute the function
			err := config.LoadConfig()

			// Assert results
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, config.Conf)
			}
		})
	}
}
