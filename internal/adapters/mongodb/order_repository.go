package mongodb

import (
	"context"
	"errors"
	"fmt"

	"order-service/internal/domain/order"
	"order-service/internal/domain/ports"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var _ ports.OrderRepository = (*OrderRepository)(nil)

type OrderRepository struct {
	collection *mongo.Collection
}

func NewOrderRepository(db *mongo.Database) *OrderRepository {
	return &OrderRepository{
		collection: db.Collection("orders"),
	}
}

func (r *OrderRepository) Create(ctx context.Context, o *order.Order) error {
	if o.ID == bson.NilObjectID {
		o.ID = bson.NewObjectID()
	}

	_, err := r.collection.InsertOne(ctx, o)
	if err != nil {
		return fmt.Errorf("mongo insert order: %w", err)
	}
	return nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (*order.Order, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, order.ErrInvalidOrder
	}

	var o order.Order
	if err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&o); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, order.ErrOrderNotFound
		}
		return nil, fmt.Errorf("mongo find order: %w", err)
	}

	return &o, nil
}

func (r *OrderRepository) Update(ctx context.Context, o *order.Order) error {
	if o.ID == bson.NilObjectID {
		return order.ErrInvalidOrder
	}

	filter := bson.M{"_id": o.ID}
	update := bson.M{"$set": bson.M{
		"customer_id": o.CustomerID,
		"items":       o.Items,
		"status":      o.Status,
		"total_price": o.TotalPrice,
		"created_at":  o.CreatedAt,
		"updated_at":  o.UpdatedAt,
	}}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("mongo update order: %w", err)
	}
	if res.MatchedCount == 0 {
		return order.ErrOrderNotFound
	}
	return nil
}
