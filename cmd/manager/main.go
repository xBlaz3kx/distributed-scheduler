package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/GLCharge/otelzap"
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

const serviceName = "manager"

var serviceInfo = observability.ServiceInfo{
	Name:    serviceName,
	Version: "0.1.2",
}

type config struct {
	Observability observability.Config   `mapstructure:"observability" yaml:"observability"`
	Http          devxHttp.Configuration `mapstructure:"http" yaml:"http"`
	DB            database.Config        `mapstructure:"db" yaml:"db"`
	OpenAPI       struct {
		Scheme string `conf:"default:http"`
		Enable bool   `conf:"default:true"`
		Host   string `conf:"default:localhost:8000"`
	}
}

var rootCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Scheduler manager",
	PreRun: func(cmd *cobra.Command, args []string) {
		devxCfg.SetupEnv(serviceName)
		devxCfg.SetDefaults(serviceName)

		viper.SetDefault("storage.encryption.key", "ishouldreallybechanged")
		viper.SetDefault("db.disable_tls", true)
		viper.SetDefault("db.max_open_conns", 1)
		viper.SetDefault("db.max_idle_conns", 10)
		viper.SetDefault("observability.logging.level", observability.LogLevelInfo)

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

	// Configuration
	cfg := &config{}
	devxCfg.GetConfiguration(viper.GetViper(), cfg)

	// Setup observability
	obs, err := observability.NewObservability(ctx, serviceInfo, cfg.Observability)
	if err != nil {
		otelzap.L().Fatal("failed to initialize observability", zap.Error(err))
	}

	log := obs.Log()

	// App Starting
	log.Info("Starting the manager", zap.String("version", serviceInfo.Version), zap.Any("config", cfg))
	defer log.Info("shutdown complete")

	// Database Support
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
		log.Fatal("failed to connect to the database", zap.Error(err))
	}

	defer func() {
		log.Info("Closing the database connection")
		_ = db.Close()
	}()

	httpServer := devxHttp.NewServer(cfg.Http, obs)
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
		log.Info("Starting HTTP server", zap.String("host", cfg.Http.Address))
		databaseCheck := database.NewHealthChecker(db)
		httpServer.Run(databaseCheck)
	}()

	// Shutdown
	<-ctx.Done()
	log.Info("Shutting down the manager")
}
