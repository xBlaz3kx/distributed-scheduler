package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/xBlaz3kx/distributed-scheduler/foundation/database"
	"github.com/xBlaz3kx/distributed-scheduler/foundation/database/dbmigrate"
	"time"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate db to latest version.",
	Run:   migrateRun,
}

var dbConfig database.Config

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().StringVar(&dbConfig.User, "user", "scheduler", "database user")
	migrateCmd.Flags().StringVar(&dbConfig.Password, "pass", "scheduler", "database password")
	migrateCmd.Flags().StringVar(&dbConfig.Host, "host", "localhost:5432", "database host")
	migrateCmd.Flags().StringVar(&dbConfig.Name, "name", "scheduler", "database name")
	migrateCmd.Flags().BoolVar(&dbConfig.DisableTLS, "disable_tls", true, "database sslmode disabled")
	migrateCmd.Flags().IntVar(&dbConfig.MaxIdleConns, "max_idle_conns", 3, "database max idle connections")
	migrateCmd.Flags().IntVar(&dbConfig.MaxOpenConns, "max_open_conns", 2, "database max open connections")
}

func migrateRun(cmd *cobra.Command, args []string) {
	db, err := database.Open(dbConfig)
	if err != nil {
		fmt.Printf("open database: %v", err)
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := dbmigrate.Migrate(ctx, db); err != nil {
		fmt.Printf("migrate database: %v", err)
		return
	}

	fmt.Println("migrations complete")
}
