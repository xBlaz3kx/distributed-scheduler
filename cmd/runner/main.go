package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/GLCharge/otelzap"
	"github.com/ardanlabs/conf/v3"
	"github.com/spf13/cobra"
	devxHttp "github.com/xBlaz3kx/DevX/http"
	"github.com/xBlaz3kx/DevX/observability"
	"github.com/xBlaz3kx/distributed-scheduler/internal/executor"
	"github.com/xBlaz3kx/distributed-scheduler/internal/pkg/database"
	"github.com/xBlaz3kx/distributed-scheduler/internal/pkg/logger"
	"github.com/xBlaz3kx/distributed-scheduler/internal/runner"
	"github.com/xBlaz3kx/distributed-scheduler/internal/service/job"
	"github.com/xBlaz3kx/distributed-scheduler/internal/store/postgres"
	"go.uber.org/zap"
)

var build = "develop"

var serviceInfo = observability.ServiceInfo{
	Name:    "manager",
	Version: build,
}

type config struct {
	conf.Version
	Web struct {
		ReadTimeout     time.Duration `conf:"default:5s"`
		WriteTimeout    time.Duration `conf:"default:10s"`
		IdleTimeout     time.Duration `conf:"default:120s"`
		ShutdownTimeout time.Duration `conf:"default:20s"`
		APIHost         string        `conf:"default:0.0.0.0:8000"`
	}
	DB                database.Config
	ID                string        `conf:"default:instance1"`
	Interval          time.Duration `conf:"default:10s"`
	MaxConcurrentJobs int           `conf:"default:100"`
	MaxJobLockTime    time.Duration `conf:"default:1m"`
}

var rootCmd = &cobra.Command{
	Use:   "runner",
	Short: "Scheduler runner",
	Run:   runCmd,
}

func main() {
	cobra.OnInitialize(logger.SetupLogging)
	err := rootCmd.Execute()
	if err != nil {
		panic(err)
	}
}

func runCmd(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	obsConfig := observability.Config{}
	obs, err := observability.NewObservability(ctx, serviceInfo, obsConfig)
	if err != nil {
		otelzap.L().Fatal("failed to initialize observability", zap.Error(err))
	}

	log := obs.Log()

	// config
	cfg := config{
		Version: conf.Version{
			Build: build,
			Desc:  "copyright information here",
		},
	}

	const prefix = "RUNNER"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return
		}
		return
	}

	// App Starting
	log.Info("starting service", zap.String("version", build))
	defer log.Info("shutdown complete")

	out, err := conf.String(&cfg)
	if err != nil {
		log.Fatal("parsing config", zap.Error(err))
	}

	log.Info("Using config", zap.Any("config", out))

	// Database
	log.Info("startup", zap.String("status", "initializing database support"), zap.String("host", cfg.DB.Host))
	db, err := database.Open(database.Config{
		User:         cfg.DB.User,
		Password:     cfg.DB.Password,
		Host:         cfg.DB.Host,
		Name:         cfg.DB.Name,
		MaxIdleConns: cfg.DB.MaxIdleConns,
		MaxOpenConns: cfg.DB.MaxOpenConns,
		DisableTLS:   cfg.DB.DisableTLS,
	})
	if err != nil {
		log.Fatal("Unable to establish DB connection", zap.Error(err))
	}
	defer func() {
		log.Info("closing database connection")
		_ = db.Close()
	}()

	// Start Runner Service
	log.Info("Starting runner service")

	store := postgres.New(db, log)

	jobService := job.NewService(store, log)

	executorFactory := executor.NewFactory(&http.Client{Timeout: 30 * time.Second})

	runner := runner.New(runner.Config{
		JobService:        jobService,
		Log:               log,
		ExecutorFactory:   executorFactory,
		InstanceId:        cfg.ID,
		Interval:          cfg.Interval,
		MaxConcurrentJobs: cfg.MaxConcurrentJobs,
		JobLockDuration:   cfg.MaxJobLockTime,
	})
	runner.Start()

	httpServer := devxHttp.NewServer(devxHttp.Configuration{Address: cfg.Web.APIHost}, observability.NewNoopObservability())
	go func() {
		log.Info("Started HTTP server", zap.String("host", cfg.Web.APIHost))
		databaseCheck := database.NewHealthChecker(db)
		httpServer.Run(databaseCheck)
	}()

	//nolint:all
	select {
	case _ = <-ctx.Done():
		log.Info("shutdown", zap.String("status", "shutdown started"))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// stop the runner
		runner.Stop(ctx)
		log.Info("shutdown", zap.String("status", "shutdown complete"))
	}
}
