package mongo

import (
	"context"
	"time"

	"order-service/internal/config"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Client interface {
	Database() *mongo.Database
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
}

type client struct {
	client   *mongo.Client
	database *mongo.Database
	cfg      *config.Config
}

func New(ctx context.Context, cfg *config.Config) (Client, error) {
	clientOpts := options.Client().ApplyURI(cfg.MongoDB.URI)

	if cfg.MongoDB.MaxPoolSize > 0 {
		clientOpts.SetMaxPoolSize(cfg.MongoDB.MaxPoolSize)
	}
	if cfg.MongoDB.MinPoolSize > 0 {
		clientOpts.SetMinPoolSize(cfg.MongoDB.MinPoolSize)
	}
	if cfg.MongoDB.MaxConnIdleTimeSeconds > 0 {
		clientOpts.SetMaxConnIdleTime(time.Duration(cfg.MongoDB.MaxConnIdleTimeSeconds) * time.Second)
	}

	mongoClient, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, err
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, time.Duration(cfg.MongoDB.TimeoutSeconds)*time.Second)
	defer pingCancel()

	if err := mongoClient.Ping(pingCtx, nil); err != nil {
		_ = mongoClient.Disconnect(context.Background())
		return nil, err
	}

	db := mongoClient.Database(cfg.MongoDB.Database)

	return &client{
		client:   mongoClient,
		database: db,
		cfg:      cfg,
	}, nil
}

func (c *client) Database() *mongo.Database {
	return c.database
}

func (c *client) Ping(ctx context.Context) error {
	ctx, pingCancel := context.WithTimeout(ctx, time.Duration(c.cfg.MongoDB.TimeoutSeconds)*time.Second)
	defer pingCancel()

	return c.client.Ping(ctx, nil)
}

func (c *client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}
