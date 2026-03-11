package dto

import (
	"time"

	"order-service/internal/domain/order"
)

type OrderResponse struct {
	ID         string              `json:"id"`
	CustomerID string              `json:"customer_id"`
	Items      []OrderItemResponse `json:"items"`
	Status     string              `json:"status"`
	TotalPrice float64             `json:"total_price"`
	CreatedAt  string              `json:"created_at"`
	UpdatedAt  string              `json:"updated_at"`
}

type OrderItemResponse struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func FromDomain(o *order.Order) OrderResponse {
	items := make([]OrderItemResponse, len(o.Items))
	for i, it := range o.Items {
		items[i] = OrderItemResponse{
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
		}
	}

	return OrderResponse{
		ID:         o.ID.Hex(),
		CustomerID: o.CustomerID,
		Items:      items,
		Status:     string(o.Status),
		TotalPrice: o.TotalPrice,
		CreatedAt:  o.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  o.UpdatedAt.UTC().Format(time.RFC3339),
	}
}


