// Package config loads the service configuration from environment variables and an optional .env file, and
// gives modules the same reader for their own settings.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/netip"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const (
	envDevelopment = "development"
	envProduction  = "production"
)

// Source looks a setting up by name; ok is false when the setting is not set.
type Source func(key string) (value string, ok bool)

// Config is the platform's configuration. Each module reads its own settings from the same Source.
type Config struct {
	// Env is "development" or "production".
	Env string
	// Service names the service in log records.
	Service  string
	HTTP     HTTP
	Log      Log
	Database Database
	// Swagger serves the Swagger UI and the OpenAPI document under /swagger/.
	Swagger bool
}

// HTTP configures the HTTP server.
type HTTP struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	// BodyLimitBytes caps the size of a request body.
	BodyLimitBytes int64
	// TrustedProxies are the reverse proxies whose X-Forwarded-For header is believed; nil trusts none.
	TrustedProxies []netip.Prefix
	// CORSAllowedOrigins lists the origins a browser may call the API from; "*" allows any origin.
	CORSAllowedOrigins []string
}

// Log configures logging.
type Log struct {
	Level slog.Level
	// Format is "text" or "json".
	Format string
	// Source adds the file and line of the logging call to every record.
	Source bool
}

// Database configures the Postgres connection.
type Database struct {
	// URL is a pgx connection string.
	URL string
}

// Production reports whether the service runs in production.
func (c Config) Production() bool { return c.Env == envProduction }

// FromEnv builds a Config from source. A variable that is unset or empty takes its default. Every invalid or
// missing setting is reported in one error.
func FromEnv(source Source) (Config, error) {
	reader := NewReader(source)
	config := Config{
		Env:     reader.OneOf("APP_ENV", envDevelopment, envDevelopment, envProduction),
		Service: reader.String("SERVICE_NAME", "go-rest-boilerplate"),
	}
	production := config.Production()

	config.HTTP = HTTP{
		Addr:               reader.String("HTTP_ADDR", ":8080"),
		ReadHeaderTimeout:  reader.Duration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout:        reader.Duration("HTTP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:       reader.Duration("HTTP_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:        reader.Duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout:    reader.Duration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		BodyLimitBytes:     reader.PositiveInt("HTTP_BODY_LIMIT_BYTES", 1<<20),
		TrustedProxies:     reader.prefixes("HTTP_TRUSTED_PROXIES"),
		CORSAllowedOrigins: reader.List("HTTP_CORS_ALLOWED_ORIGINS"),
	}
	if config.HTTP.CORSAllowedOrigins == nil {
		if production {
			reader.Fail("HTTP_CORS_ALLOWED_ORIGINS", "is required in production")
		} else {
			config.HTTP.CORSAllowedOrigins = []string{"*"}
		}
	}

	format := "text"
	if production {
		format = "json"
	}
	config.Log = Log{
		Level:  reader.level("LOG_LEVEL", slog.LevelInfo),
		Format: reader.OneOf("LOG_FORMAT", format, "text", "json"),
		Source: reader.Bool("LOG_SOURCE", true),
	}

	config.Swagger = reader.Bool("SWAGGER_ENABLED", !production)
	config.Database = Database{URL: reader.Required("DATABASE_URL")}

	if err := reader.Err(); err != nil {
		return Config{}, err
	}
	return config, nil
}

// LoadDatabase reads only DATABASE_URL, from LoadSource.
func LoadDatabase() (Database, error) {
	source, err := LoadSource()
	if err != nil {
		return Database{}, err
	}

	reader := NewReader(source)
	database := Database{URL: reader.Required("DATABASE_URL")}
	if err := reader.Err(); err != nil {
		return Database{}, err
	}
	return database, nil
}

// LoadSource returns the process environment, backed by ./.env when that file exists. A variable set in the
// process environment wins over the file, and the file's values are never written into the process
// environment. It returns an error when ./.env exists but cannot be read.
func LoadSource() (Source, error) {
	file, err := godotenv.Read(".env")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read .env: %w", err)
	}

	return func(key string) (string, bool) {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			return value, true
		}
		value, ok := file[key]
		return value, ok
	}, nil
}
