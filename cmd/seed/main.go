package main

import (
	_ "github.com/lib/pq"
	"github.com/snykk/go-rest-boilerplate/cmd/seed/seeders"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/drivers"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

func init() {
	if err := config.InitializeAppConfig(); err != nil {
		logger.Fatal(err.Error(), logger.Fields{constants.LoggerCategory: constants.LoggerCategoryConfig})
	}
	logger.Info("configuration loaded", logger.Fields{constants.LoggerCategory: constants.LoggerCategoryConfig})
}

func main() {
	db, err := drivers.SetupSQLXPostgres()
	if err != nil {
		logger.Panic(err.Error(), logger.Fields{constants.LoggerCategory: constants.LoggerCategorySeeder})
	}
	defer func() { _ = db.Close() }()

	logger.Info("seeding...", logger.Fields{constants.LoggerCategory: constants.LoggerCategorySeeder})

	seeder := seeders.NewSeeder(db)
	err = seeder.UserSeeder(seeders.UserData)
	if err != nil {
		logger.Panic(err.Error(), logger.Fields{constants.LoggerCategory: constants.LoggerCategorySeeder})
	}

	logger.Info("seeding success!", logger.Fields{constants.LoggerCategory: constants.LoggerCategorySeeder})
}
