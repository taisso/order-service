package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"order-service/internal/config"
	"order-service/internal/domain/ports"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

var _ ports.EventPublisher = (*Publisher)(nil)

type Publisher struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	cfg     *config.Config
	logger  *zap.Logger
}

func NewPublisher(cfg *config.Config, logger *zap.Logger) (*Publisher, error) {
	conn, err := amqp091.Dial(cfg.RabbitMQ.URI)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}

	if err = ch.ExchangeDeclare(
		cfg.RabbitMQ.Exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq exchange declare: %w", err)
	}

	if _, err = ch.QueueDeclare(
		cfg.RabbitMQ.Queue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq queue declare: %w", err)
	}

	if err = ch.QueueBind(
		cfg.RabbitMQ.Queue,
		cfg.RabbitMQ.RoutingKey,
		cfg.RabbitMQ.Exchange,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq queue bind: %w", err)
	}

	return &Publisher{
		conn:    conn,
		channel: ch,
		cfg:     cfg,
		logger:  logger,
	}, nil
}

func (p *Publisher) PublishStatusChanged(ctx context.Context, event ports.StatusChangedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal status changed event: %w", err)
	}

	err = p.channel.PublishWithContext(ctx,
		p.cfg.RabbitMQ.Exchange,
		p.cfg.RabbitMQ.RoutingKey,
		false,
		false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		p.logger.Error("failed to publish status changed event", zap.Error(err))
		return err
	}

	return nil
}

func (p *Publisher) Close() error {
	if err := p.channel.Close(); err != nil {
		_ = p.conn.Close()
		return err
	}
	return p.conn.Close()
}
