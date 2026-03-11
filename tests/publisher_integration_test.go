package tests

import (
	"context"
	"testing"
	"time"

	"order-service/internal/adapters/rabbitmq"
	"order-service/internal/config"
	"order-service/internal/domain/ports"

	"github.com/ory/dockertest/v3"
	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type PublisherIntegrationSuite struct {
	suite.Suite
	pool     *dockertest.Pool
	resource *dockertest.Resource
	cfg      *config.Config
	pub      *rabbitmq.Publisher
}

func TestPublisherIntegrationSuite(t *testing.T) {
	suite.Run(t, new(PublisherIntegrationSuite))
}

func (s *PublisherIntegrationSuite) SetupSuite() {
	rc, err := StartRabbitMQContainer()
	s.Require().NoError(err)

	s.pool = rc.Pool
	s.resource = rc.Resource

	uri := rc.URI

	cfg := &config.Config{}
	cfg.RabbitMQ.URI = uri
	cfg.RabbitMQ.Exchange = "orders"
	cfg.RabbitMQ.Queue = "order-status-events"
	cfg.RabbitMQ.RoutingKey = "order.status.updated"
	cfg.App.Env = "test"
	cfg.Logger.Level = "info"
	s.cfg = cfg

	var pub *rabbitmq.Publisher

	s.Require().NoError(Retry(s.pool, func(ctx context.Context) error {
		p, err := rabbitmq.NewPublisher(cfg, zap.NewNop())
		if err != nil {
			return err
		}
		pub = p
		return nil
	}))

	s.pub = pub
}

func (s *PublisherIntegrationSuite) TearDownSuite() {
	if s.pub != nil {
		_ = s.pub.Close()
	}
	if s.pool != nil && s.resource != nil {
		_ = s.pool.Purge(s.resource)
	}
}

func (s *PublisherIntegrationSuite) TestPublishAndConsume() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	event := ports.StatusChangedEvent{
		OrderID:    "abc123",
		OldStatus:  "criado",
		NewStatus:  "em_processamento",
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
	}

	err := s.pub.PublishStatusChanged(ctx, event)
	s.Require().NoError(err)

	conn, err := amqp091.Dial(s.cfg.RabbitMQ.URI)
	s.Require().NoError(err)
	defer conn.Close()

	ch, err := conn.Channel()
	s.Require().NoError(err)
	defer ch.Close()

	msgs, err := ch.Consume(
		s.cfg.RabbitMQ.Queue,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	s.Require().NoError(err)

	timeout := time.After(5 * time.Second)
	select {
	case msg := <-msgs:
		s.Contains(string(msg.Body), `"order_id":"abc123"`)
	case <-timeout:
		s.Fail("no message consumed")
	}
}
