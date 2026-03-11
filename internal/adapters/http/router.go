package http

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "order-service/docs"
)

func NewRouter(handler *Handler, logger *zap.Logger) *gin.Engine {
	router := gin.New()

	// Middlewares
	router.Use(gin.Recovery())
	router.Use(RequestLogger(logger))

	router.GET("/health", handler.Health)

	// Orders
	router.POST("/orders", handler.CreateOrder)
	router.GET("/orders/:id", handler.GetOrder)
	router.PATCH("/orders/:id/status", handler.UpdateOrderStatus)

	// Swagger
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return router
}
