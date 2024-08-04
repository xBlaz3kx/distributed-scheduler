package cmd

import (
	"os"

	"github.com/GLCharge/otelzap"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xBlaz3kx/distributed-scheduler/internal/pkg/logger"
)

var rootCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "CLI tool for managing the scheduler.",
}

func setupLogging() {
	logLevel := viper.GetString("log.level")
	logger, err := logger.New(logLevel)
	if err == nil {
		otelzap.ReplaceGlobals(logger)
	}
}

func Execute() {
	cobra.OnInitialize(setupLogging)
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
