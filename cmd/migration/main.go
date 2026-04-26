package main

import (
	"context"
	"flag"

	"github.com/sirupsen/logrus"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/drivers"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/migration"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

const migrationsDir = "cmd/migration/migrations"

var (
	up   bool
	down bool
)

func init() {
	if err := config.InitializeAppConfig(); err != nil {
		logger.Fatal(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryConfig})
	}
	logger.Info("configuration loaded", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryConfig})
}

func main() {
	flag.BoolVar(&up, "up", false, "apply new tables, columns, or other structures")
	flag.BoolVar(&down, "down", false, "drop tables, columns, or other structures")
	flag.Parse()

	db, err := drivers.SetupSQLXPostgres()
	if err != nil {
		logger.Panic(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
	}
	defer db.Close()

	runner := migration.New(db, migrationsDir)
	ctx := context.Background()

	if up {
		if err := runner.Up(ctx); err != nil {
			logger.Fatal(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
		}
	}
	if down {
		if err := runner.Down(ctx); err != nil {
			logger.Fatal(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
		}
	}
}
