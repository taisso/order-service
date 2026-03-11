package dto

import domain "order-service/internal/domain/order"

type CreateOrderRequest struct {
	CustomerID string            `json:"customer_id" binding:"required"`
	Items      []CreateOrderItem `json:"items" binding:"required,min=1,dive"`
}

type CreateOrderItem struct {
	ProductID   string  `json:"product_id" binding:"required"`
	ProductName string  `json:"product_name" binding:"required"`
	Quantity    int     `json:"quantity" binding:"required,gt=0"`
	UnitPrice   float64 `json:"unit_price" binding:"required,gte=0"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (r CreateOrderRequest) ToDomainItems() []domain.OrderItem {
	items := make([]domain.OrderItem, len(r.Items))
	for i, it := range r.Items {
		items[i] = domain.OrderItem{
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
		}
	}
	return items
}

