package main

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/GLCharge/otelzap"
	"github.com/ardanlabs/conf/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	devxCfg "github.com/xBlaz3kx/DevX/configuration"
	devxHttp "github.com/xBlaz3kx/DevX/http"
	"github.com/xBlaz3kx/DevX/observability"
	api "github.com/xBlaz3kx/distributed-scheduler/internal/api/http"
	"github.com/xBlaz3kx/distributed-scheduler/internal/pkg/database"
	"github.com/xBlaz3kx/distributed-scheduler/internal/pkg/logger"
	"github.com/xBlaz3kx/distributed-scheduler/internal/pkg/security"
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
	DB      database.Config
	OpenAPI struct {
		Scheme string `conf:"default:http"`
		Enable bool   `conf:"default:true"`
		Host   string `conf:"default:localhost:8000"`
	}
}

var rootCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Scheduler manager",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.SetDefault("storage.encryption.key", "ishouldreallybechanged")
		devxCfg.InitConfig("", "./config", ".")

		postgres.SetEncryptor(security.NewEncryptorFromEnv())
	},
	Run: runCmd,
}

func main() {
	cobra.OnInitialize(func() {
		logger.SetupLogging()
		devxCfg.SetupEnv("manager")
	})
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

	// Configuration
	cfg := config{
		Version: conf.Version{
			Build: build,
			Desc:  "copyright information here",
		},
	}

	const prefix = "MANAGER"
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
		return
	}

	log.Info("startup", zap.String("config", out))

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
		return
	}

	defer func() {
		log.Info("shutdown", zap.String("status", "stopping database support"), zap.String("host", cfg.DB.Host))
		_ = db.Close()
	}()

	httpCfg := devxHttp.Configuration{Address: cfg.Web.APIHost}
	httpServer := devxHttp.NewServer(httpCfg, obs)

	api.Api(httpServer.Router(), api.APIMuxConfig{
		Log: log,
		DB:  db,
		OpenApi: api.OpenApiConfig{
			Enabled: cfg.OpenAPI.Enable,
			Scheme:  cfg.OpenAPI.Scheme,
			Host:    cfg.OpenAPI.Host,
		},
	})

	go func() {
		log.Info("Starting HTTP server", zap.String("host", httpCfg.Address))
		databaseCheck := database.NewHealthChecker(db)
		httpServer.Run(databaseCheck)
	}()

	// Start API Service
	log.Info("startup", zap.String("status", "initializing Management API support"))

	// Shutdown
	select {
	case sig := <-ctx.Done():
		log.Info("shutdown", zap.String("status", "shutdown started"), zap.Any("signal", sig))
		defer log.Info("shutdown", zap.String("status", "shutdown complete"), zap.Any("signal", sig))
	}
}
