// Package logger provides a convenience function to constructing a logger
// for use. This is required not just for applications but for testing.
package logger

import (
	"github.com/GLCharge/otelzap"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"github.com/xBlaz3kx/DevX/observability"
)

func SetupLogging() {
	logLevel := viper.GetString("log.level")
	logger := observability.NewLogging(observability.LogConfig{Level: lo.ToPtr(observability.LogLevel(logLevel))})
	otelzap.ReplaceGlobals(logger.Logger())
}
