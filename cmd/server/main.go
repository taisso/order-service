package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	httpadapter "order-service/internal/adapters/http"
	"order-service/internal/adapters/mongodb"
	"order-service/internal/adapters/rabbitmq"
	apporder "order-service/internal/application/order"
	"order-service/internal/config"
	pkglogger "order-service/internal/pkg/logger"
	mongoclient "order-service/internal/pkg/mongo"

	"go.uber.org/zap"
)

// @title           Order Service API
// @version         1.0
// @description     Serviço de gerenciamento de pedidos de e-commerce.
// @BasePath        /
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := pkglogger.New(cfg.App.Env, cfg.Logger.Level)
	defer func() {
		_ = logger.Sync()
	}()

	ctx := context.Background()

	mongoClient, err := mongoclient.New(ctx, cfg)
	if err != nil {
		logger.Fatal("failed to init mongo client", zap.Error(err))
	}
	defer func() {
		_ = mongoClient.Close(context.Background())
	}()

	repo := mongodb.NewOrderRepository(mongoClient.Database())

	publisher, err := rabbitmq.NewPublisher(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init rabbitmq publisher", zap.Error(err))
	}
	defer func() {
		_ = publisher.Close()
	}()

	service := apporder.NewService(repo, publisher)
	handler := httpadapter.NewHandler(service, mongoClient)
	router := httpadapter.NewRouter(handler, logger)

	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.App.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.App.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.App.WriteTimeoutSeconds) * time.Second,
		IdleTimeout:  time.Duration(cfg.App.IdleTimeoutSeconds) * time.Second,
	}

	go func() {
		logger.Info("starting HTTP server", zap.Int("port", cfg.App.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server listen", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}
}
