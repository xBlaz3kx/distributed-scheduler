package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/xBlaz3kx/distributed-scheduler/internal/pkg/logger"
)

var rootCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "CLI tool for managing the scheduler.",
}

func Execute() {
	cobra.OnInitialize(logger.SetupLogging)
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
