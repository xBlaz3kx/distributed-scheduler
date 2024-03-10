package handlers

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/xBlaz3kx/distributed-scheduler/docs"
)

type OpenApiConfig struct {
	Scheme  string
	Host    string
	Enabled bool
}

func OpenApiRoute(cfg OpenApiConfig, router *gin.Engine) {

	if !cfg.Enabled {
		return
	}
	// Swagger API documentation
	docs.SwaggerInfo.Schemes = []string{cfg.Scheme}
	docs.SwaggerInfo.Host = cfg.Host

	persist := ginSwagger.PersistAuthorization(true)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, persist))
}
