package drivers

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
)

// SetupSQLXPostgres builds and pings a *sqlx.DB pointed at Postgres,
// reading the DB_POSTGRE_* keys from config.AppConfig. The two-part
// name encodes both layers: the driver (sqlx) and the engine
// (Postgres). A future MySQL-via-sqlx wiring would land alongside
// this as SetupSQLXMySQL; a Mongo wiring would be SetupMongo in its
// own file. Lives in the drivers package because it's composition
// that turns config values into a connected driver — same layer as
// the driver implementation.
func SetupSQLXPostgres() (*sqlx.DB, error) {
	var dsn string
	switch config.AppConfig.Environment {
	case constants.EnvironmentDevelopment:
		dsn = config.AppConfig.DBPostgreDsn
	case constants.EnvironmentProduction:
		dsn = config.AppConfig.DBPostgreURL
	}

	cfg := SQLXConfig{
		DriverName:     config.AppConfig.DBPostgreDriver,
		DataSourceName: dsn,
		MaxOpenConns:   config.AppConfig.DBMaxOpenConns,
		MaxIdleConns:   config.AppConfig.DBMaxIdleConns,
		MaxLifetime:    time.Duration(config.AppConfig.DBConnMaxLifeMins) * time.Minute,
	}
	return cfg.InitializeSQLXDatabase()
}
