package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/psds-microservice/helpy/paths"
	"github.com/psds-microservice/operator-directory-service/api"
	"github.com/psds-microservice/operator-directory-service/internal/handler"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func New(directoryHandler *handler.DirectoryHandler) http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET(paths.PathHealth, handler.Health)
	r.GET(paths.PathReady, handler.Ready)
	// Swagger: один маршрут *any, чтобы не конфликтовать с openapi.json в дереве Gin
	r.GET(paths.PathSwagger+"/*any", func(c *gin.Context) {
		if strings.TrimPrefix(c.Param("any"), "/") == "openapi.json" {
			c.Data(http.StatusOK, "application/json", api.OpenAPISpec)
			return
		}
		ginSwagger.WrapHandler(swaggerFiles.Handler,
			ginSwagger.URL("/swagger/openapi.json"),
			ginSwagger.DefaultModelsExpandDepth(-1),
		)(c)
	})
	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/operators", directoryHandler.List)
		apiV1.GET("/operators/:id", directoryHandler.Get)
		apiV1.POST("/operators", directoryHandler.Create)
		apiV1.PUT("/operators/:id", directoryHandler.Update)
	}
	return r
}
