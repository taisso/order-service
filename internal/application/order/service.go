package order

import (
	"context"
	"errors"
	"time"

	domain "order-service/internal/domain/order"
	"order-service/internal/domain/ports"
)

var (
	ErrInvalidOrder            = errors.New("invalid order")
	ErrEmptyItems              = errors.New("order must have at least one item")
	ErrInvalidItem             = errors.New("invalid order item")
	ErrInvalidQuantity         = errors.New("invalid item quantity")
	ErrInvalidUnitPrice        = errors.New("invalid item unit price")
	ErrInvalidStatus           = errors.New("invalid status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrOrderNotFound           = errors.New("order not found")
)

type Service interface {
	CreateOrder(ctx context.Context, customerID string, items []domain.OrderItem) (*domain.Order, error)
	GetOrderByID(ctx context.Context, id string) (*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, id string, newStatus domain.Status) (*domain.Order, error)
}

type service struct {
	repo      ports.OrderRepository
	publisher ports.EventPublisher
}

func NewService(repo ports.OrderRepository, publisher ports.EventPublisher) Service {
	return &service{
		repo:      repo,
		publisher: publisher,
	}
}

func (s *service) CreateOrder(ctx context.Context, customerID string, items []domain.OrderItem) (*domain.Order, error) {
	order, err := domain.NewOrder(customerID, items, time.Now())
	if err != nil {
		return nil, err
	}

	if err = s.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *service) GetOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) UpdateOrderStatus(ctx context.Context, id string, newStatus domain.Status) (*domain.Order, error) {
	if !newStatus.IsValid() {
		return nil, domain.ErrInvalidStatus
	}

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldStatus := order.Status
	now := time.Now()
	if err = order.UpdateStatus(newStatus, now); err != nil {
		return nil, err
	}

	if err = s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	event := ports.StatusChangedEvent{
		OrderID:    id,
		OldStatus:  string(oldStatus),
		NewStatus:  string(newStatus),
		OccurredAt: now.UTC().Format(time.RFC3339),
	}
	if err := s.publisher.PublishStatusChanged(ctx, event); err != nil {
		return nil, err
	}

	return order, nil
}
