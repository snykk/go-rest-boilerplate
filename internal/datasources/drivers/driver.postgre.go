package drivers

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	configEnv "github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type ConfigPostgreSQL struct {
	DB_Username string
	DB_Password string
	DB_Host     string
	DB_Port     int
	DB_Database string
	DB_DSN      string
}

func dbMigrate(db *gorm.DB) (err error) {
	return
}

func (config *ConfigPostgreSQL) InitializeDatabasePostgreSQL() (*gorm.DB, error) {
	var dsn string

	if configEnv.AppConfig.Environment == constants.EnvironmentDevelopment {
		dsn = fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
			config.DB_Host, config.DB_Port, config.DB_Database,
			config.DB_Username, config.DB_Password)
	} else if configEnv.AppConfig.Environment == constants.EnvironmentProduction {
		dsn = config.DB_DSN
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		return nil, errors.New("failed connecting to PostgreSQL")
	}
	logger.Info("connected to PostgreSQL", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryInit})

	// if configEnv.AppConfig.Environment == constants.EnvironmentDevelopment {
	// 	if err = db.Migrator().DropTable("users", "roles", "genders", "books", "reviews"); err != nil {
	// 		return nil, errors.New("failed droping tables:" + err.Error())
	// 	}
	// 	logger.Info("droping tables success", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryInit})
	// }

	err = dbMigrate(db)
	if err != nil {
		return nil, errors.New("failed when running migration")
	}

	logger.Info("migration success", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryInit})

	// if configEnv.AppConfig.Environment == constants.EnvironmentDevelopment {
	// 	logger.Info("lazy seeders success")
	// }

	return db, nil
}
