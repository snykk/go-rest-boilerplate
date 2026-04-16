package config

import (
	"fmt"

	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/spf13/viper"
)

var AppConfig Config

type Config struct {
	Port        int    `mapstructure:"PORT"`
	Environment string `mapstructure:"ENVIRONMENT"`
	Debug       bool   `mapstructure:"DEBUG"`

	DBPostgreDriver string `mapstructure:"DB_POSTGRE_DRIVER"`
	DBPostgreDsn    string `mapstructure:"DB_POSTGRE_DSN"`
	DBPostgreURL    string `mapstructure:"DB_POSTGRE_URL"`

	JWTSecret  string `mapstructure:"JWT_SECRET"`
	JWTExpired int    `mapstructure:"JWT_EXPIRED"`
	JWTIssuer  string `mapstructure:"JWT_ISSUER"`

	OTPEmail    string `mapstructure:"OTP_EMAIL"`
	OTPPassword string `mapstructure:"OTP_PASSWORD"`

	REDISHost     string `mapstructure:"REDIS_HOST"`
	REDISPassword string `mapstructure:"REDIS_PASS"`
	REDISExpired  int    `mapstructure:"REDIS_EXPIRED"`
}

func InitializeAppConfig() error {
	viper.SetConfigName(".env") // allow directly reading from .env file
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("internal/config")
	viper.AddConfigPath("/")
	viper.AllowEmptyEnv(true)
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		return constants.ErrLoadConfig
	}

	err = viper.Unmarshal(&AppConfig)
	if err != nil {
		return constants.ErrParseConfig
	}

	// check
	if AppConfig.Port == 0 || AppConfig.Environment == "" || AppConfig.JWTSecret == "" || AppConfig.JWTExpired == 0 || AppConfig.JWTIssuer == "" || AppConfig.OTPEmail == "" || AppConfig.OTPPassword == "" || AppConfig.REDISHost == "" || AppConfig.REDISPassword == "" || AppConfig.REDISExpired == 0 || AppConfig.DBPostgreDriver == "" {
		return constants.ErrEmptyVar
	}

	if AppConfig.Port < 1 || AppConfig.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535, got %d", AppConfig.Port)
	}
	if AppConfig.JWTExpired < 1 || AppConfig.JWTExpired > 720 {
		return fmt.Errorf("JWT_EXPIRED must be between 1 and 720 hours, got %d", AppConfig.JWTExpired)
	}
	if len(AppConfig.JWTSecret) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	if AppConfig.REDISExpired < 1 {
		return fmt.Errorf("REDIS_EXPIRED must be at least 1 minute, got %d", AppConfig.REDISExpired)
	}

	switch AppConfig.Environment {
	case constants.EnvironmentDevelopment:
		if AppConfig.DBPostgreDsn == "" {
			return constants.ErrEmptyVar
		}
	case constants.EnvironmentProduction:
		if AppConfig.DBPostgreURL == "" {
			return constants.ErrEmptyVar
		}
	}

	return nil
}
