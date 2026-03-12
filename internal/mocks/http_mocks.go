package mocks

import (
	"context"

	domain "order-service/internal/domain/order"
	"order-service/internal/domain/ports"
	mongoclient "order-service/internal/pkg/mongo"

	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// MockOrderService implementa apporder.Service para testes de camada HTTP.
type MockOrderService struct {
	mock.Mock
}

// Garante em tempo de compilação que MockOrderService implementa ports.Service.
var _ ports.Service = (*MockOrderService)(nil)

func (m *MockOrderService) CreateOrder(
	ctx context.Context,
	customerID string,
	items []domain.OrderItem,
) (*domain.Order, error) {
	args := m.Called(ctx, customerID, items)
	order, _ := args.Get(0).(*domain.Order)
	return order, args.Error(1)
}

func (m *MockOrderService) GetOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	args := m.Called(ctx, id)
	order, _ := args.Get(0).(*domain.Order)
	return order, args.Error(1)
}

func (m *MockOrderService) UpdateOrderStatus(
	ctx context.Context,
	id string,
	newStatus domain.Status,
) (*domain.Order, error) {
	args := m.Called(ctx, id, newStatus)
	order, _ := args.Get(0).(*domain.Order)
	return order, args.Error(1)
}

// MockMongoClient implementa mongoclient.Client para testes.
type MockMongoClient struct {
	mock.Mock
}

var _ mongoclient.Client = (*MockMongoClient)(nil)

func (m *MockMongoClient) Database() *mongo.Database { return nil }

func (m *MockMongoClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMongoClient) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

