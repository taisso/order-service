package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"order-service/internal/adapters/http/dto"
	domain "order-service/internal/domain/order"
	"order-service/internal/domain/ports"
	"order-service/internal/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

func TestNewHandler(t *testing.T) {
	svc := new(mocks.MockOrderService)
	db := new(mocks.MockMongoClient)

	h := NewHandler(svc, db)

	assert.NotNil(t, h)
}

func setupRouter(svc ports.Service) *gin.Engine {
	db := new(mocks.MockMongoClient)
	db.On("Ping", mock.Anything).Return(nil)
	handler := NewHandler(svc, db)
	logger := zap.NewNop()
	return NewRouter(handler, logger)
}

func TestHandler_Health(t *testing.T) {
	svc := new(mocks.MockOrderService)
	logger := zap.NewNop()

	// caso saudável
	dbHealthy := new(mocks.MockMongoClient)
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
	dbUnhealthy := new(mocks.MockMongoClient)
	dbUnhealthy.On("Ping", mock.Anything).Return(errors.New("db down"))
	handlerUnhealthy := NewHandler(svc, dbUnhealthy)
	routerUnhealthy := NewRouter(handlerUnhealthy, logger)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/health", nil)

	routerUnhealthy.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "unhealthy", resp.Status)
}

func TestHandler_CreateOrder_Success(t *testing.T) {
	svc := new(mocks.MockOrderService)
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

	orderRequest := dto.CreateOrderRequest{
		CustomerID: "customer-42",
		Items: []dto.CreateOrderItem{
			{ProductID: "prod-01", ProductName: "Tênis", Quantity: 2, UnitPrice: 199.90},
		},
	}

	body, err := json.Marshal(orderRequest)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_CreateOrder_InvalidBody(t *testing.T) {
	svc := new(mocks.MockOrderService)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(`{invalid json`)))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateOrder_ValidationError(t *testing.T) {
	svc := new(mocks.MockOrderService)
	router := setupRouter(svc)

	// JSON válido mas sem items (viola min=1 em binding)
	orderRequest := dto.CreateOrderRequest{
		CustomerID: "customer-42",
		Items:      []dto.CreateOrderItem{},
	}
	body, err := json.Marshal(orderRequest)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateOrder_ServiceError(t *testing.T) {
	svc := new(mocks.MockOrderService)
	router := setupRouter(svc)

	svc.
		On("CreateOrder", mock.Anything, "customer-42", mock.AnythingOfType("[]order.OrderItem")).
		Return((*domain.Order)(nil), assert.AnError)

	orderRequest := dto.CreateOrderRequest{
		CustomerID: "customer-42",
		Items: []dto.CreateOrderItem{
			{ProductID: "prod-01", ProductName: "Tênis", Quantity: 2, UnitPrice: 199.90},
		},
	}
	body, err := json.Marshal(orderRequest)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_GetOrder_NotFound(t *testing.T) {
	svc := new(mocks.MockOrderService)
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
	svc := new(mocks.MockOrderService)
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
	svc := new(mocks.MockOrderService)
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
	svc := new(mocks.MockOrderService)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader([]byte(`{invalid json`)))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateOrderStatus_Success(t *testing.T) {
	svc := new(mocks.MockOrderService)
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

	updateStatusRequest := dto.UpdateStatusRequest{
		Status: "em_processamento",
	}
	body, err := json.Marshal(updateStatusRequest)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateOrderStatus_ErrOrderNotFound(t *testing.T) {
	svc := new(mocks.MockOrderService)
	router := setupRouter(svc)

	svc.
		On("UpdateOrderStatus", mock.Anything, "id-1", domain.StatusProcessing).
		Return((*domain.Order)(nil), domain.ErrOrderNotFound)

	updateStatusRequest := dto.UpdateStatusRequest{
		Status: "em_processamento",
	}
	body, err := json.Marshal(updateStatusRequest)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateOrderStatus_ErrInvalidStatus(t *testing.T) {
	svc := new(mocks.MockOrderService)
	router := setupRouter(svc)

	svc.
		On("UpdateOrderStatus", mock.Anything, "id-1", domain.Status("invalido")).
		Return((*domain.Order)(nil), domain.ErrInvalidStatus)

	updateStatusRequest := dto.UpdateStatusRequest{
		Status: "invalido",
	}
	body, err := json.Marshal(updateStatusRequest)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateOrderStatus_ErrInvalidStatusTransition(t *testing.T) {
	svc := new(mocks.MockOrderService)
	router := setupRouter(svc)

	svc.
		On("UpdateOrderStatus", mock.Anything, "id-1", domain.StatusProcessing).
		Return((*domain.Order)(nil), domain.ErrInvalidStatusTransition)

	updateStatusRequest := dto.UpdateStatusRequest{
		Status: "em_processamento",
	}
	body, err := json.Marshal(updateStatusRequest)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateOrderStatus_ErrSameStatus(t *testing.T) {
	svc := new(mocks.MockOrderService)
	router := setupRouter(svc)

	svc.
		On("UpdateOrderStatus", mock.Anything, "id-1", domain.StatusProcessing).
		Return((*domain.Order)(nil), domain.ErrSameStatus)

	updateStatusRequest := dto.UpdateStatusRequest{
		Status: "em_processamento",
	}
	body, err := json.Marshal(updateStatusRequest)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateOrderStatus_OtherError(t *testing.T) {
	svc := new(mocks.MockOrderService)
	router := setupRouter(svc)

	svc.
		On("UpdateOrderStatus", mock.Anything, "id-1", domain.StatusProcessing).
		Return((*domain.Order)(nil), errors.New("db error"))

	updateStatusRequest := dto.UpdateStatusRequest{
		Status: "em_processamento",
	}
	body, err := json.Marshal(updateStatusRequest)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateOrderStatus_ConcurrentUpdate(t *testing.T) {
	svc := new(mocks.MockOrderService)
	router := setupRouter(svc)

	svc.
		On("UpdateOrderStatus", mock.Anything, "id-1", domain.StatusProcessing).
		Return((*domain.Order)(nil), domain.ErrConcurrentUpdate)

	updateStatusRequest := dto.UpdateStatusRequest{
		Status: "em_processamento",
	}
	body, err := json.Marshal(updateStatusRequest)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orders/id-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	svc.AssertExpectations(t)
}
