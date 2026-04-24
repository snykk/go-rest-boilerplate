package config

import (
	"fmt"
	"net/url"
	"strings"

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

	DBMaxOpenConns    int `mapstructure:"DB_MAX_OPEN_CONNS"`
	DBMaxIdleConns    int `mapstructure:"DB_MAX_IDLE_CONNS"`
	DBConnMaxLifeMins int `mapstructure:"DB_CONN_MAX_LIFE_MINS"`

	JWTSecret  string `mapstructure:"JWT_SECRET"`
	JWTExpired int    `mapstructure:"JWT_EXPIRED"`
	JWTIssuer  string `mapstructure:"JWT_ISSUER"`

	OTPEmail       string `mapstructure:"OTP_EMAIL"`
	OTPPassword    string `mapstructure:"OTP_PASSWORD"`
	OTPMaxAttempts int    `mapstructure:"OTP_MAX_ATTEMPTS"`

	MailerWorkers   int `mapstructure:"MAILER_WORKERS"`
	MailerQueueSize int `mapstructure:"MAILER_QUEUE_SIZE"`
	MailerRetries   int `mapstructure:"MAILER_RETRIES"`

	REDISHost     string `mapstructure:"REDIS_HOST"`
	REDISPassword string `mapstructure:"REDIS_PASS"`
	REDISExpired  int    `mapstructure:"REDIS_EXPIRED"`

	AllowedOrigins string `mapstructure:"ALLOWED_ORIGINS"`
}

// AllowedOriginsList returns CORS origins as a slice. Defaults to ["*"] only when empty AND environment is not production.
func (c *Config) AllowedOriginsList() []string {
	if c.AllowedOrigins == "" {
		if c.Environment == constants.EnvironmentProduction {
			return nil
		}
		return []string{"*"}
	}
	parts := strings.Split(c.AllowedOrigins, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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

	applyDefaults()

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
	if len(AppConfig.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters (got %d) — HS256 requires 256-bit entropy", len(AppConfig.JWTSecret))
	}
	if AppConfig.REDISExpired < 1 {
		return fmt.Errorf("REDIS_EXPIRED must be at least 1 minute, got %d", AppConfig.REDISExpired)
	}
	if AppConfig.DBMaxOpenConns < 1 || AppConfig.DBMaxIdleConns < 0 || AppConfig.DBMaxIdleConns > AppConfig.DBMaxOpenConns {
		return fmt.Errorf("invalid DB pool config: open=%d idle=%d", AppConfig.DBMaxOpenConns, AppConfig.DBMaxIdleConns)
	}
	if AppConfig.OTPMaxAttempts < 1 {
		return fmt.Errorf("OTP_MAX_ATTEMPTS must be >= 1, got %d", AppConfig.OTPMaxAttempts)
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
		if _, err := url.Parse(AppConfig.DBPostgreURL); err != nil {
			return fmt.Errorf("DB_POSTGRE_URL is not a valid URL: %w", err)
		}
		if AppConfig.AllowedOrigins == "" {
			return fmt.Errorf("ALLOWED_ORIGINS must be set in production (comma-separated origins)")
		}
	default:
		return fmt.Errorf("ENVIRONMENT must be 'development' or 'production', got %q", AppConfig.Environment)
	}

	return nil
}

// applyDefaults fills in sane defaults for optional config values.
func applyDefaults() {
	if AppConfig.DBMaxOpenConns == 0 {
		AppConfig.DBMaxOpenConns = 25
	}
	if AppConfig.DBMaxIdleConns == 0 {
		AppConfig.DBMaxIdleConns = 5
	}
	if AppConfig.DBConnMaxLifeMins == 0 {
		AppConfig.DBConnMaxLifeMins = 15
	}
	if AppConfig.OTPMaxAttempts == 0 {
		AppConfig.OTPMaxAttempts = 5
	}
	if AppConfig.MailerWorkers == 0 {
		AppConfig.MailerWorkers = 2
	}
	if AppConfig.MailerQueueSize == 0 {
		AppConfig.MailerQueueSize = 64
	}
	if AppConfig.MailerRetries == 0 {
		AppConfig.MailerRetries = 3
	}
}
