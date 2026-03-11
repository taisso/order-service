package tests

import (
	"context"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

type MongoContainer struct {
	Pool     *dockertest.Pool
	Resource *dockertest.Resource
	URI      string
}

type RabbitMQContainer struct {
	Pool     *dockertest.Pool
	Resource *dockertest.Resource
	URI      string
}

func StartMongoContainer() (*MongoContainer, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, err
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mongo",
		Tag:        "7",
		Env:        []string{"MONGO_INITDB_DATABASE=orders_db"},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, err
	}

	resource.Expire(120)

	hostAndPort := resource.GetHostPort("27017/tcp")
	uri := "mongodb://" + hostAndPort

	return &MongoContainer{
		Pool:     pool,
		Resource: resource,
		URI:      uri,
	}, nil
}

func StartRabbitMQContainer() (*RabbitMQContainer, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, err
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "rabbitmq",
		Tag:        "3-management",
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, err
	}

	resource.Expire(120)

	hostAndPort := resource.GetHostPort("5672/tcp")
	uri := "amqp://guest:guest@" + hostAndPort + "/"

	return &RabbitMQContainer{
		Pool:     pool,
		Resource: resource,
		URI:      uri,
	}, nil
}

func Retry(pool *dockertest.Pool, fn func(ctx context.Context) error) error {
	return pool.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return fn(ctx)
	})
}
