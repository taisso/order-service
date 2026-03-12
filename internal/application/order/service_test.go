package order

import (
	"context"
	"testing"

	domain "order-service/internal/domain/order"
	"order-service/internal/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_CreateOrder_Success(t *testing.T) {
	repo := new(mocks.MockOrderRepository)
	publisher := new(mocks.MockEventPublisher)

	items := []domain.OrderItem{
		{
			ProductID:   "prod-01",
			ProductName: "Tênis",
			Quantity:    2,
			UnitPrice:   199.90,
		},
	}

	repo.
		On("Create", mock.Anything, mock.AnythingOfType("*order.Order")).
		Return(nil)
	svc := NewService(repo, publisher)

	o, err := svc.CreateOrder(context.TODO(), "customer-42", items)
	assert.NoError(t, err)
	assert.Equal(t, "customer-42", o.CustomerID)
	assert.Equal(t, domain.StatusCreated, o.Status)
	assert.InDelta(t, 399.80, o.TotalPrice, 0.001)

	repo.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

func TestService_CreateOrder_InvalidData(t *testing.T) {
	repo := new(mocks.MockOrderRepository)
	publisher := new(mocks.MockEventPublisher)
	svc := NewService(repo, publisher)

	o, err := svc.CreateOrder(context.TODO(), "", nil)
	assert.Error(t, err)
	assert.Nil(t, o)
}

func TestService_CreateOrder_RepoError(t *testing.T) {
	repo := new(mocks.MockOrderRepository)
	publisher := new(mocks.MockEventPublisher)
	svc := NewService(repo, publisher)

	items := []domain.OrderItem{
		{
			ProductID:   "prod-01",
			ProductName: "Tênis",
			Quantity:    1,
			UnitPrice:   100,
		},
	}

	repo.
		On("Create", mock.Anything, mock.AnythingOfType("*order.Order")).
		Return(assert.AnError)

	o, err := svc.CreateOrder(context.TODO(), "customer-1", items)
	assert.Error(t, err)
	assert.Nil(t, o)

	repo.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

func TestService_GetOrderByID_Success(t *testing.T) {
	repo := new(mocks.MockOrderRepository)
	publisher := new(mocks.MockEventPublisher)
	svc := NewService(repo, publisher)

	expected := &domain.Order{CustomerID: "customer-1"}

	repo.
		On("GetByID", mock.Anything, "id-1").
		Return(expected, nil)

	o, err := svc.GetOrderByID(context.TODO(), "id-1")
	assert.NoError(t, err)
	assert.Equal(t, expected, o)

	repo.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

func TestService_UpdateOrderStatus_ValidTransition_PublishesEvent(t *testing.T) {
	repo := new(mocks.MockOrderRepository)
	publisher := new(mocks.MockEventPublisher)
	svc := NewService(repo, publisher)

	ctx := context.TODO()

	existing := &domain.Order{
		Status: domain.StatusCreated,
	}

	repo.
		On("GetByID", mock.Anything, "id-1").
		Return(existing, nil)
	repo.
		On("Update", mock.Anything, mock.AnythingOfType("*order.Order")).
		Return(nil)
	publisher.
		On("PublishStatusChanged", mock.Anything, mock.AnythingOfType("ports.StatusChangedEvent")).
		Return(nil)

	updated, err := svc.UpdateOrderStatus(ctx, "id-1", domain.StatusProcessing)
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusProcessing, updated.Status)

	repo.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

func TestService_UpdateOrderStatus_InvalidStatus(t *testing.T) {
	repo := new(mocks.MockOrderRepository)
	publisher := new(mocks.MockEventPublisher)
	svc := NewService(repo, publisher)

	ctx := context.TODO()

	updated, err := svc.UpdateOrderStatus(ctx, "id-1", domain.Status("invalido"))
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.Equal(t, domain.ErrInvalidStatus, err)

	repo.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

func TestService_UpdateOrderStatus_InvalidTransition(t *testing.T) {
	repo := new(mocks.MockOrderRepository)
	publisher := new(mocks.MockEventPublisher)
	svc := NewService(repo, publisher)

	ctx := context.TODO()

	existing := &domain.Order{
		Status: domain.StatusCreated,
	}

	repo.
		On("GetByID", mock.Anything, "id-1").
		Return(existing, nil)

	updated, err := svc.UpdateOrderStatus(ctx, "id-1", domain.StatusCreated)
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.Equal(t, domain.ErrSameStatus, err)

	repo.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

func TestService_UpdateOrderStatus_RepoGetError(t *testing.T) {
	repo := new(mocks.MockOrderRepository)
	publisher := new(mocks.MockEventPublisher)
	svc := NewService(repo, publisher)

	ctx := context.TODO()

	repo.
		On("GetByID", mock.Anything, "id-1").
		Return((*domain.Order)(nil), assert.AnError)

	updated, err := svc.UpdateOrderStatus(ctx, "id-1", domain.StatusProcessing)
	assert.Error(t, err)
	assert.Nil(t, updated)

	repo.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

func TestService_UpdateOrderStatus_PublisherFailureReturnsError(t *testing.T) {
	repo := new(mocks.MockOrderRepository)
	publisher := new(mocks.MockEventPublisher)
	svc := NewService(repo, publisher)

	ctx := context.TODO()

	existing := &domain.Order{
		Status: domain.StatusCreated,
	}

	repo.
		On("GetByID", mock.Anything, "id-1").
		Return(existing, nil)
	repo.
		On("Update", mock.Anything, mock.AnythingOfType("*order.Order")).
		Return(nil)
	publisher.
		On("PublishStatusChanged", mock.Anything, mock.AnythingOfType("ports.StatusChangedEvent")).
		Return(assert.AnError)

	updated, err := svc.UpdateOrderStatus(ctx, "id-1", domain.StatusProcessing)
	assert.Error(t, err)
	assert.Nil(t, updated)

	repo.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

func TestService_UpdateOrderStatus_RepoUpdateError(t *testing.T) {
	repo := new(mocks.MockOrderRepository)
	publisher := new(mocks.MockEventPublisher)
	svc := NewService(repo, publisher)

	ctx := context.TODO()

	existing := &domain.Order{
		Status: domain.StatusCreated,
	}

	repo.
		On("GetByID", mock.Anything, "id-1").
		Return(existing, nil)
	repo.
		On("Update", mock.Anything, mock.AnythingOfType("*order.Order")).
		Return(assert.AnError)

	updated, err := svc.UpdateOrderStatus(ctx, "id-1", domain.StatusProcessing)
	assert.Error(t, err)
	assert.Nil(t, updated)

	repo.AssertExpectations(t)
	publisher.AssertExpectations(t)
}
