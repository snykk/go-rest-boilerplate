package drivers

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"github.com/uptrace/opentelemetry-go-extra/otelsqlx"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// SQLXConfig holds the configuration for the database instance
type SQLXConfig struct {
	DriverName     string
	DataSourceName string
	MaxOpenConns   int
	MaxIdleConns   int
	MaxLifetime    time.Duration
}

// InitializeSQLXDatabase returns a new DBInstance. The handle is
// instrumented with OpenTelemetry — every Query/Exec emits a span
// tagged with semantic-convention attributes (db.system, db.statement)
// and the underlying *sql.DB stats are exposed as OTel metrics.
func (config *SQLXConfig) InitializeSQLXDatabase() (*sqlx.DB, error) {
	db, err := otelsqlx.Open(
		config.DriverName,
		config.DataSourceName,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithDBName(config.DriverName),
	)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// set maximum number of open connections to database
	logger.Info(fmt.Sprintf("Setting maximum number of open connections to %d", config.MaxOpenConns), logger.Fields{constants.LoggerCategory: constants.LoggerCategoryDatabase})
	db.SetMaxOpenConns(config.MaxOpenConns)

	// set maximum number of idle connections in the pool
	logger.Info(fmt.Sprintf("Setting maximum number of idle connections to %d", config.MaxIdleConns), logger.Fields{constants.LoggerCategory: constants.LoggerCategoryDatabase})
	db.SetMaxIdleConns(config.MaxIdleConns)

	// set maximum time to wait for new connection
	logger.Info(fmt.Sprintf("Setting maximum lifetime for a connection to %s", config.MaxLifetime), logger.Fields{constants.LoggerCategory: constants.LoggerCategoryDatabase})
	db.SetConnMaxLifetime(config.MaxLifetime)

	// set maximum idle time for connections
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging database: %v", err)
	}

	return db, nil
}
