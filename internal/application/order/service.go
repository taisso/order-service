package order

import (
	"context"
	"time"

	domain "order-service/internal/domain/order"
	"order-service/internal/domain/ports"
)

var _ ports.Service = (*Service)(nil)

type Service struct {
	repo      ports.OrderRepository
	publisher ports.EventPublisher
}

func NewService(repo ports.OrderRepository, publisher ports.EventPublisher) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
	}
}

func (s *Service) CreateOrder(
	ctx context.Context,
	customerID string,
	items []domain.OrderItem,
) (*domain.Order, error) {
	order, err := domain.NewOrder(customerID, items, time.Now())
	if err != nil {
		return nil, err
	}

	if err = s.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *Service) GetOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) UpdateOrderStatus(
	ctx context.Context,
	id string,
	newStatus domain.Status,
) (*domain.Order, error) {
	if !newStatus.IsValid() {
		return nil, domain.ErrInvalidStatus
	}

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldStatus := order.Status
	if err = order.UpdateStatus(newStatus); err != nil {
		return nil, err
	}

	if err = s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	event := ports.StatusChangedEvent{
		OrderID:    id,
		OldStatus:  string(oldStatus),
		NewStatus:  string(newStatus),
		OccurredAt: order.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if err := s.publisher.PublishStatusChanged(ctx, event); err != nil {
		return nil, err
	}

	return order, nil
}
