package order

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Order struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	CustomerID string        `bson:"customer_id" json:"customer_id"`
	Items      []OrderItem   `bson:"items" json:"items"`
	Status     Status        `bson:"status" json:"status"`
	TotalPrice float64       `bson:"total_price" json:"total_price"`
	CreatedAt  time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time     `bson:"updated_at" json:"updated_at"`
}

type OrderItem struct {
	ProductID   string  `bson:"product_id" json:"product_id"`
	ProductName string  `bson:"product_name" json:"product_name"`
	Quantity    int     `bson:"quantity" json:"quantity"`
	UnitPrice   float64 `bson:"unit_price" json:"unit_price"`
}

func NewOrder(customerID string, items []OrderItem, now time.Time) (*Order, error) {
	if customerID == "" {
		return nil, ErrInvalidOrder
	}

	if len(items) == 0 {
		return nil, ErrEmptyItems
	}

	var total float64
	for _, item := range items {
		if item.ProductID == "" || item.ProductName == "" {
			return nil, ErrInvalidItem
		}
		if item.Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}
		if item.UnitPrice < 0 {
			return nil, ErrInvalidUnitPrice
		}
		total += float64(item.Quantity) * item.UnitPrice
	}

	order := &Order{
		ID:         bson.NilObjectID,
		CustomerID: customerID,
		Items:      items,
		Status:     StatusCreated,
		TotalPrice: total,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return order, nil
}

func (o *Order) CanTransitionTo(newStatus Status) bool {
	if !newStatus.IsValid() || !o.Status.IsValid() {
		return false
	}

	return newStatus != o.Status
}

func (o *Order) UpdateStatus(newStatus Status, now time.Time) error {
	if !newStatus.IsValid() {
		return ErrInvalidStatus
	}

	if newStatus == o.Status {
		return ErrSameStatus
	}

	o.Status = newStatus
	o.UpdatedAt = now

	return nil
}
