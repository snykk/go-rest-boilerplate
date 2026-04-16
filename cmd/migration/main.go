package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/utils"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

const (
	dir = "cmd/migration/migrations"
)

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
	flag.BoolVar(&up, "up", false, "involves creating new tables, columns, or other database structures")
	flag.BoolVar(&down, "down", false, "involves dropping tables, columns, or other structures")
	flag.Parse()

	db, err := utils.SetupPostgresConnection()
	if err != nil {
		logger.Panic(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
	}
	defer db.Close()

	if up {
		err = migrate(db, "up")
		if err != nil {
			logger.Fatal(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
		}
	}

	if down {
		err = migrate(db, "down")
		if err != nil {
			logger.Fatal(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
		}
	}
}

func migrate(db *sqlx.DB, action string) (err error) {
	logger.InfoF("running migration [%s]", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration}, action)

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(cwd, dir, fmt.Sprintf("*.%s.sql", action)))
	if err != nil {
		return errors.New("error when get files name")
	}

	for _, file := range files {
		logger.Info("Executing migration", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration, constants.LoggerFile: file})
		data, err := os.ReadFile(file)
		if err != nil {
			return errors.New("error when read file")
		}

		_, err = db.Exec(string(data))
		if err != nil {
			fmt.Println(err)
			return fmt.Errorf("error when exec query in file:%v", file)
		}
	}

	logger.InfoF("migration [%s] success", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration}, action)

	return
}
