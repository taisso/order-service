package order

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewOrder_Success(t *testing.T) {
	now := time.Now()

	items := []OrderItem{
		{
			ProductID:   "prod-01",
			ProductName: "Produto 1",
			Quantity:    2,
			UnitPrice:   10.0,
		},
	}

	o, err := NewOrder("customer-1", items, now)
	assert.NoError(t, err)
	assert.Equal(t, "customer-1", o.CustomerID)
	assert.Equal(t, StatusCreated, o.Status)
	assert.Equal(t, 20.0, o.TotalPrice)
	assert.Equal(t, now, o.CreatedAt)
	assert.Equal(t, now, o.UpdatedAt)
}

func TestNewOrder_InvalidData(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		custID  string
		items   []OrderItem
		wantErr error
	}{
		{
			name:    "empty customer",
			custID:  "",
			items:   []OrderItem{{ProductID: "p", ProductName: "n", Quantity: 1, UnitPrice: 1}},
			wantErr: ErrInvalidOrder,
		},
		{
			name:    "no items",
			custID:  "c1",
			items:   nil,
			wantErr: ErrEmptyItems,
		},
		{
			name:   "invalid item quantity",
			custID: "c1",
			items: []OrderItem{
				{ProductID: "p", ProductName: "n", Quantity: 0, UnitPrice: 1},
			},
			wantErr: ErrInvalidQuantity,
		},
		{
			name:   "invalid item price",
			custID: "c1",
			items: []OrderItem{
				{ProductID: "p", ProductName: "n", Quantity: 1, UnitPrice: -1},
			},
			wantErr: ErrInvalidUnitPrice,
		},
		{
			name:   "empty product id",
			custID: "c1",
			items: []OrderItem{
				{ProductID: "", ProductName: "n", Quantity: 1, UnitPrice: 1},
			},
			wantErr: ErrInvalidItem,
		},
		{
			name:   "empty product name",
			custID: "c1",
			items: []OrderItem{
				{ProductID: "p", ProductName: "", Quantity: 1, UnitPrice: 1},
			},
			wantErr: ErrInvalidItem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := NewOrder(tt.custID, tt.items, now)
			assert.Error(t, err)
			assert.Nil(t, o)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestUpdateStatus_Transitions(t *testing.T) {
	now := time.Now()

	o, err := NewOrder("customer-1", []OrderItem{
		{ProductID: "p", ProductName: "n", Quantity: 1, UnitPrice: 10},
	}, now)
	assert.NoError(t, err)

	// criado -> em_processamento
	err = o.UpdateStatus(StatusProcessing, now)
	assert.NoError(t, err)
	assert.Equal(t, StatusProcessing, o.Status)

	// em_processamento -> entregue
	err = o.UpdateStatus(StatusDelivered, now)
	assert.NoError(t, err)
	assert.Equal(t, StatusDelivered, o.Status)
}

func TestUpdateStatus_SameStatusError(t *testing.T) {
	now := time.Now()

	o, err := NewOrder("customer-1", []OrderItem{
		{ProductID: "p", ProductName: "n", Quantity: 1, UnitPrice: 10},
	}, now)
	assert.NoError(t, err)

	// criado -> criado (mesmo status deve falhar)
	err = o.UpdateStatus(StatusCreated, now)
	assert.Error(t, err)
	assert.Equal(t, ErrSameStatus, err)
	assert.Equal(t, StatusCreated, o.Status)
}

func TestUpdateStatus_InvalidStatus(t *testing.T) {
	now := time.Now()

	o, err := NewOrder("customer-1", []OrderItem{
		{ProductID: "p", ProductName: "n", Quantity: 1, UnitPrice: 10},
	}, now)
	assert.NoError(t, err)

	// status inválido deve retornar ErrInvalidStatus e não alterar o estado
	invalid := Status("invalido")
	err = o.UpdateStatus(invalid, now)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidStatus, err)
	assert.Equal(t, StatusCreated, o.Status)
}

func TestCanTransitionTo_AllBranches(t *testing.T) {
	o := &Order{Status: StatusCreated}

	assert.True(t, o.CanTransitionTo(StatusProcessing))
	assert.True(t, o.CanTransitionTo(StatusShipped))
	assert.True(t, o.CanTransitionTo(StatusDelivered))

	o.Status = StatusProcessing
	assert.True(t, o.CanTransitionTo(StatusShipped))
	assert.True(t, o.CanTransitionTo(StatusDelivered))

	o.Status = StatusShipped
	assert.True(t, o.CanTransitionTo(StatusDelivered))

	o.Status = Status("cancelado")
	assert.False(t, o.CanTransitionTo(StatusCreated))
}

func TestStatus_IsValid(t *testing.T) {
	t.Run("valid statuses", func(t *testing.T) {
		assert.True(t, StatusCreated.IsValid())
		assert.True(t, StatusProcessing.IsValid())
		assert.True(t, StatusShipped.IsValid())
		assert.True(t, StatusDelivered.IsValid())
	})

	t.Run("invalid status", func(t *testing.T) {
		var invalid Status = "cancelado"
		assert.False(t, invalid.IsValid())
	})
}
