package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"order-service/internal/adapters/http/dto"
	apporder "order-service/internal/application/order"
	domain "order-service/internal/domain/order"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) CreateOrder(
	ctx context.Context,
	customerID string,
	items []domain.OrderItem,
) (*domain.Order, error) {
	args := m.Called(ctx, customerID, items)
	order, _ := args.Get(0).(*domain.Order)
	return order, args.Error(1)
}

func (m *mockService) GetOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	args := m.Called(ctx, id)
	order, _ := args.Get(0).(*domain.Order)
	return order, args.Error(1)
}

func (m *mockService) UpdateOrderStatus(
	ctx context.Context,
	id string,
	newStatus domain.Status,
) (*domain.Order, error) {
	args := m.Called(ctx, id, newStatus)
	order, _ := args.Get(0).(*domain.Order)
	return order, args.Error(1)
}

type mockMongoClient struct {
	mock.Mock
}

func (m *mockMongoClient) Database() *mongo.Database { return nil }

func (m *mockMongoClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockMongoClient) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewHandler(t *testing.T) {
	svc := new(mockService)
	db := new(mockMongoClient)

	h := NewHandler(svc, db)

	assert.NotNil(t, h)
}

func setupRouter(svc apporder.Service) *gin.Engine {
	db := new(mockMongoClient)
	db.On("Ping", mock.Anything).Return(nil)
	handler := NewHandler(svc, db)
	logger := zap.NewNop()
	return NewRouter(handler, logger)
}

func TestHandler_Health(t *testing.T) {
	svc := new(mockService)
	logger := zap.NewNop()

	// caso saudável
	dbHealthy := new(mockMongoClient)
	dbHealthy.On("Ping", mock.Anything).Return(nil)
	handlerHealthy := NewHandler(svc, dbHealthy)
	router := NewRouter(handlerHealthy, logger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.NotEmpty(t, resp.Timestamp)

	// caso banco indisponível
	dbUnhealthy := new(mockMongoClient)
	dbUnhealthy.On("Ping", mock.Anything).Return(errors.New("db down"))
	handlerUnhealthy := NewHandler(svc, dbUnhealthy)
	routerUnhealthy := NewRouter(handlerUnhealthy, logger)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/health", nil)

	routerUnhealthy.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusServiceUnavailable, w2.Code)

	var resp2 dto.HealthResponse
	err = json.Unmarshal(w2.Body.Bytes(), &resp2)
	assert.NoError(t, err)
	assert.Equal(t, "unhealthy", resp2.Status)
}

func TestHandler_CreateOrder_Success(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	now := time.Now()
	order := &domain.Order{
		ID:         bson.NewObjectID(),
		CustomerID: "customer-42",
		Items: []domain.OrderItem{
			{ProductID: "prod-01", ProductName: "Tênis", Quantity: 2, UnitPrice: 199.9},
		},
		Status:     domain.StatusCreated,
		TotalPrice: 399.8,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	svc.
		On("CreateOrder", mock.Anything, "customer-42", mock.AnythingOfType("[]order.OrderItem")).
		Return(order, nil)

	body := []byte(`{
		"customer_id": "customer-42",
		"items": [
		  {
			"product_id": "prod-01",
			"product_name": "Tênis",
			"quantity": 2,
			"unit_price": 199.90
		  }
		]
	  }`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_CreateOrder_InvalidBody(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(`{invalid json`)))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateOrder_ValidationError(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	// JSON válido mas sem items (viola min=1 em binding)
	body := []byte(`{"customer_id":"customer-42","items":[]}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateOrder_ServiceError(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	svc.
		On("CreateOrder", mock.Anything, "customer-42", mock.AnythingOfType("[]order.OrderItem")).
		Return((*domain.Order)(nil), assert.AnError)

	body := []byte(`{
		"customer_id": "customer-42",
		"items": [
		  {
			"product_id": "prod-01",
			"product_name": "Tênis",
			"quantity": 2,
			"unit_price": 199.90
		  }
		]
	  }`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_GetOrder_NotFound(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	svc.
		On("GetOrderByID", mock.Anything, "unknown").
		Return((*domain.Order)(nil), domain.ErrOrderNotFound)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orders/unknown", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_GetOrder_Success(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	now := time.Now()
	order := &domain.Order{
		ID:         bson.NewObjectID(),
		CustomerID: "customer-1",
		Status:     domain.StatusProcessing,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	svc.
		On("GetOrderByID", mock.Anything, "id-1").
		Return(order, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orders/id-1", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_GetOrder_OtherError(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	svc.
		On("GetOrderByID", mock.Anything, "id-1").
		Return((*domain.Order)(nil), errors.New("db error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orders/id-1", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateOrderStatus_InvalidStatusBody(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader([]byte(`{invalid json`)))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateOrderStatus_Success(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	now := time.Now()
	order := &domain.Order{
		ID:         bson.NewObjectID(),
		CustomerID: "customer-1",
		Status:     domain.StatusProcessing,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	svc.
		On("UpdateOrderStatus", mock.Anything, "id-1", domain.StatusProcessing).
		Return(order, nil)

	body := []byte(`{"status":"em_processamento"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateOrderStatus_ErrOrderNotFound(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	svc.
		On("UpdateOrderStatus", mock.Anything, "id-1", domain.StatusProcessing).
		Return((*domain.Order)(nil), domain.ErrOrderNotFound)

	body := []byte(`{"status":"em_processamento"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateOrderStatus_ErrInvalidStatus(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	svc.
		On("UpdateOrderStatus", mock.Anything, "id-1", domain.Status("invalido")).
		Return((*domain.Order)(nil), domain.ErrInvalidStatus)

	body := []byte(`{"status":"invalido"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateOrderStatus_ErrInvalidStatusTransition(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	svc.
		On("UpdateOrderStatus", mock.Anything, "id-1", domain.StatusProcessing).
		Return((*domain.Order)(nil), domain.ErrInvalidStatusTransition)

	body := []byte(`{"status":"em_processamento"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateOrderStatus_OtherError(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	svc.
		On("UpdateOrderStatus", mock.Anything, "id-1", domain.StatusProcessing).
		Return((*domain.Order)(nil), errors.New("db error"))

	body := []byte(`{"status":"em_processamento"}`)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateOrderStatus_ConcurrentUpdate(t *testing.T) {
	svc := new(mockService)
	router := setupRouter(svc)

	svc.
		On("UpdateOrderStatus", mock.Anything, "id-1", domain.StatusProcessing).
		Return((*domain.Order)(nil), domain.ErrConcurrentUpdate)

	body := []byte(`{"status":"em_processamento"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	svc.AssertExpectations(t)
}
