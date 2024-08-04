package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GLCharge/otelzap"
	"github.com/ardanlabs/conf/v3"
	api "github.com/xBlaz3kx/distributed-scheduler/internal/api/http"
	"github.com/xBlaz3kx/distributed-scheduler/internal/executor"
	"github.com/xBlaz3kx/distributed-scheduler/internal/pkg/database"
	"github.com/xBlaz3kx/distributed-scheduler/internal/pkg/logger"
	"github.com/xBlaz3kx/distributed-scheduler/internal/runner"
	"github.com/xBlaz3kx/distributed-scheduler/internal/service/job"
	"github.com/xBlaz3kx/distributed-scheduler/internal/store/postgres"
	"go.uber.org/zap"
)

var build = "develop"

func main() {
	logLevel := os.Getenv("RUNNER_LOG_LEVEL")
	log, err := logger.New(logLevel)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer func(log *otelzap.Logger) {
		_ = log.Sync()
	}(log)

	if err := run(log); err != nil {
		log.Error("startup", zap.Error(err))
		_ = log.Sync()
		os.Exit(1)
	}
}

type Configuration struct {
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

func run(log *otelzap.Logger) error {

	// -------------------------------------------------------------------------
	// Configuration

	cfg := Configuration{
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
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	// -------------------------------------------------------------------------
	// App Starting

	log.Info("starting service", zap.String("version", build))
	defer log.Info("shutdown complete")

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config for output: %w", err)
	}
	log.Info("startup", zap.String("config", out))

	// -------------------------------------------------------------------------
	// Database Support

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
		return fmt.Errorf("connecting to db: %w", err)
	}
	defer func() {
		log.Info("shutdown", zap.String("status", "stopping database support"), zap.String("host", cfg.DB.Host))
		_ = db.Close()
	}()

	// -------------------------------------------------------------------------
	// Start Runner Service

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	log.Info("startup", zap.String("status", "initializing runner"))

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

	//
	// API
	apiMux := api.RunnerAPI(api.APIMuxConfig{
		Log: log,
		DB:  db,
	})

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      apiMux,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     zap.NewStdLog(log.Logger),
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info("startup", zap.String("status", "api router started"), zap.String("host", api.Addr))
		serverErrors <- api.ListenAndServe()
	}()

	// -------------------------------------------------------------------------
	// Shutdown

	//nolint:all
	select {
	case sig := <-shutdown:
		log.Info("shutdown", zap.String("status", "shutdown started"), zap.Any("signal", sig))
		defer log.Info("shutdown", zap.String("status", "shutdown complete"), zap.Any("signal", sig))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// stop the runner
		runner.Stop(ctx)
	}

	return nil
}
