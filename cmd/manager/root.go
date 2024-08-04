package main

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Scheduler manager",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		panic(err)
	}
}
