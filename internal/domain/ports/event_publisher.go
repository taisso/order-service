package ports

import "context"

type StatusChangedEvent struct {
	OrderID    string `json:"order_id"`
	OldStatus  string `json:"old_status"`
	NewStatus  string `json:"new_status"`
	OccurredAt string `json:"occurred_at"`
}

type EventPublisher interface {
	PublishStatusChanged(ctx context.Context, event StatusChangedEvent) error
}

