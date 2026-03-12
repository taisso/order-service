package mocks

import (
	"context"

	domain "order-service/internal/domain/order"
	"order-service/internal/domain/ports"

	"github.com/stretchr/testify/mock"
)

type MockOrderRepository struct {
	mock.Mock
}

var _ ports.OrderRepository = (*MockOrderRepository)(nil)

func (m *MockOrderRepository) Create(ctx context.Context, o *domain.Order) error {
	args := m.Called(ctx, o)
	return args.Error(0)
}

func (m *MockOrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	args := m.Called(ctx, id)
	order, _ := args.Get(0).(*domain.Order)
	return order, args.Error(1)
}

func (m *MockOrderRepository) Update(ctx context.Context, o *domain.Order) error {
	args := m.Called(ctx, o)
	return args.Error(0)
}

type MockEventPublisher struct {
	mock.Mock
}

var _ ports.EventPublisher = (*MockEventPublisher)(nil)

func (m *MockEventPublisher) PublishStatusChanged(ctx context.Context, e ports.StatusChangedEvent) error {
	args := m.Called(ctx, e)
	return args.Error(0)
}
