package ports

import (
	"context"

	"order-service/internal/domain/order"
)

type OrderRepository interface {
	Create(ctx context.Context, o *order.Order) error
	GetByID(ctx context.Context, id string) (*order.Order, error)
	Update(ctx context.Context, o *order.Order) error
}

