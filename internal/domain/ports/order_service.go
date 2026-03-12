package ports

import (
	"context"
	"order-service/internal/domain/order"
)

type Service interface {
	CreateOrder(ctx context.Context, customerID string, items []order.OrderItem) (*order.Order, error)
	GetOrderByID(ctx context.Context, id string) (*order.Order, error)
	UpdateOrderStatus(ctx context.Context, id string, newStatus order.Status) (*order.Order, error)
}
