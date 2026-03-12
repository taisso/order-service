package order

import "errors"

var (
	ErrInvalidOrder            = errors.New("invalid order")
	ErrEmptyItems              = errors.New("order must have at least one item")
	ErrInvalidItem             = errors.New("invalid order item")
	ErrInvalidQuantity         = errors.New("invalid item quantity")
	ErrInvalidUnitPrice        = errors.New("invalid item unit price")
	ErrInvalidStatus           = errors.New("invalid status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrSameStatus              = errors.New("same status")
	ErrOrderNotFound           = errors.New("order not found")
	ErrConcurrentUpdate        = errors.New("concurrent update")
)
