package config

import (
	"errors"

	"github.com/spf13/viper"
)

var AppConfig Config

type Config struct {
	Port        int    `mapstructure:"PORT"`
	Environment string `mapstructure:"ENVIRONMENT"`
	Debug       bool   `mapstructure:"DEBUG"`

	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     int    `mapstructure:"DB_PORT"`
	DBDatabase string `mapstructure:"DB_DATABASE"`
	DBUsername string `mapstructure:"DB_USERNAME"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBDsn      string `mapstructure:"DB_DSN"`

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
		return errors.New("failed to load config file")
	}

	err = viper.Unmarshal(&AppConfig)
	if err != nil {
		return errors.New("failed to parse env to config struct")
	}

	// check
	if AppConfig.Port == 0 || AppConfig.Environment == "" || AppConfig.JWTSecret == "" || AppConfig.JWTExpired == 0 || AppConfig.JWTIssuer == "" || AppConfig.OTPEmail == "" || AppConfig.OTPPassword == "" || AppConfig.REDISHost == "" || AppConfig.REDISPassword == "" || AppConfig.REDISExpired == 0 {
		return errors.New("required variabel environment is empty")
	}

	switch AppConfig.Environment {
	case "development":
		if AppConfig.DBHost == "" || AppConfig.DBPort == 0 || AppConfig.DBDatabase == "" || AppConfig.DBUsername == "" || AppConfig.DBPassword == "" {
			return errors.New("required variabel environment is empty")
		}
	case "production":
		if AppConfig.DBDsn == "" {
			return errors.New("required variabel environment is empty")
		}
	}

	return nil
}
