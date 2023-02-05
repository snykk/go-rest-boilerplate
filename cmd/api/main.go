package main

import (
	"fmt"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/snykk/go-rest-boilerplate/cmd/api/server"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

func init() {
	if err := config.InitializeAppConfig(); err != nil {
		logger.Fatal(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryInit})
	}
	logger.Info("configuration loaded", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryInit})
}

func main() {
	numCPU := runtime.NumCPU()
	logger.Info(fmt.Sprintf("The project is running on %d CPU(s)", numCPU), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryDevice})

	if runtime.NumCPU() > 2 {
		runtime.GOMAXPROCS(numCPU / 2)
	}

	app, err := server.NewApp()
	if err != nil {
		logger.Panic(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryInit})
	}
	if err := app.Run(); err != nil {
		logger.Fatal(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryClose})
	}
}
