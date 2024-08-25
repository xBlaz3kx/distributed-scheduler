package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/GLCharge/otelzap"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	devxCfg "github.com/xBlaz3kx/DevX/configuration"
	devxHttp "github.com/xBlaz3kx/DevX/http"
	"github.com/xBlaz3kx/DevX/observability"
	"github.com/xBlaz3kx/distributed-scheduler/internal/executor"
	"github.com/xBlaz3kx/distributed-scheduler/internal/pkg/database"
	"github.com/xBlaz3kx/distributed-scheduler/internal/pkg/logger"
	"github.com/xBlaz3kx/distributed-scheduler/internal/pkg/security"
	"github.com/xBlaz3kx/distributed-scheduler/internal/runner"
	"github.com/xBlaz3kx/distributed-scheduler/internal/service/job"
	"github.com/xBlaz3kx/distributed-scheduler/internal/store/postgres"
	"go.uber.org/zap"
)

const serviceName = "runner"

var serviceInfo = observability.ServiceInfo{
	Name:    serviceName,
	Version: "0.1.2",
}

type config struct {
	Observability        observability.Config        `mapstructure:"observability" yaml:"observability"`
	Http                 devxHttp.Configuration      `mapstructure:"http" yaml:"http"`
	DB                   database.Config             `mapstructure:"db" yaml:"db"`
	ID                   string                      `conf:"default:instance1" yaml:"id"`
	JobExecutionSettings runner.JobExecutionSettings `mapstructure:"job_execution_settings" yaml:"jobExecutionSettings"`
}

var rootCmd = &cobra.Command{
	Use:   "runner",
	Short: "Scheduler runner",
	PreRun: func(cmd *cobra.Command, args []string) {
		devxCfg.SetupEnv(serviceName)
		devxCfg.SetDefaults(serviceName)

		viper.SetDefault("storage.encryption.key", "ishouldreallybechanged")
		viper.SetDefault("db.disable_tls", true)
		viper.SetDefault("db.max_open_conns", 1)
		viper.SetDefault("db.max_idle_conns", 10)
		viper.SetDefault("observability.logging.level", observability.LogLevelInfo)

		viper.SetDefault("job_execution_settings.max_concurrent_jobs", 100)
		viper.SetDefault("job_execution_settings.interval", time.Second*10)
		viper.SetDefault("job_execution_settings.max_job_lock_time", time.Minute)

		devxCfg.InitConfig("", "./config", ".")

		postgres.SetEncryptor(security.NewEncryptorFromEnv())
	},
	Run: runCmd,
}

func main() {
	cobra.OnInitialize(func() {
		logger.SetupLogging()
	})
	err := rootCmd.Execute()
	if err != nil {
		panic(err)
	}
}

func runCmd(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	// Read the configuration
	cfg := &config{}
	devxCfg.GetConfiguration(viper.GetViper(), cfg)

	obs, err := observability.NewObservability(ctx, serviceInfo, cfg.Observability)
	if err != nil {
		otelzap.L().Fatal("failed to initialize observability", zap.Error(err))
	}

	log := obs.Log()

	// App Starting
	log.Info("Starting the runner", zap.String("version", serviceInfo.Version))
	defer log.Info("shutdown complete")

	log.Info("Using config", zap.Any("config", cfg))

	// Database
	log.Info("Connecting to the database", zap.String("host", cfg.DB.Host))
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
		log.Info("Closing the database connection")
		_ = db.Close()
	}()

	// Start Runner Service
	log.Info("Starting runner service")

	store := postgres.New(db, log)

	jobService := job.NewService(store, log)

	executorFactory := executor.NewFactory(&http.Client{Timeout: 30 * time.Second})

	runner := runner.New(runner.Config{
		JobService:      jobService,
		Log:             log,
		ExecutorFactory: executorFactory,
		InstanceId:      cfg.ID,
		JobExecution:    cfg.JobExecutionSettings,
	})
	runner.Start()

	httpServer := devxHttp.NewServer(cfg.Http, observability.NewNoopObservability())
	go func() {
		log.Info("Started HTTP server", zap.String("address", cfg.Http.Address))
		databaseCheck := database.NewHealthChecker(db)
		httpServer.Run(databaseCheck)
	}()

	//nolint:all
	select {
	case _ = <-ctx.Done():
		log.Info("Shutting down the runner")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// stop the runner
		runner.Stop(ctx)
	}
}
