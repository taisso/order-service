package http

import (
	"net/http"
	"time"

	"order-service/internal/adapters/http/dto"
	apporder "order-service/internal/application/order"
	domain "order-service/internal/domain/order"
	mongoclient "order-service/internal/pkg/mongo"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service apporder.Service
	db      mongoclient.Client
}

func NewHandler(service apporder.Service, db mongoclient.Client) *Handler {
	return &Handler{service: service, db: db}
}

// @Summary      Healthcheck
// @Description  Verifica se o serviço está saudável (inclui ping ao MongoDB)
// @Tags         health
// @Produce      json
// @Success      200  {object}  dto.HealthResponse
// @Failure      503  {object}  dto.HealthResponse
// @Router       /health [get]
func (h *Handler) Health(c *gin.Context) {
	status := "ok"
	code := http.StatusOK

	if err := h.db.Ping(c.Request.Context()); err != nil {
		status = "unhealthy"
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, dto.HealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// @Summary      Criar pedido
// @Description  Cria um novo pedido
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateOrderRequest  true  "Pedido"
// @Success      201   {object}  dto.OrderResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Router       /orders [post]
func (h *Handler) CreateOrder(c *gin.Context) {
	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items := req.ToDomainItems()

	order, err := h.service.CreateOrder(c.Request.Context(), req.CustomerID, items)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.FromDomain(order))
}

// @Summary      Obter pedido
// @Description  Retorna um pedido pelo ID
// @Tags         orders
// @Produce      json
// @Param        id   path      string  true  "Order ID"
// @Success      200  {object}  dto.OrderResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /orders/{id} [get]
func (h *Handler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	order, err := h.service.GetOrderByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.FromDomain(order))
}

// @Summary      Atualizar status do pedido
// @Description  Atualiza o status de um pedido
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        id    path      string                  true  "Order ID"
// @Param        body  body      dto.UpdateStatusRequest true  "Novo status"
// @Success      200   {object}  dto.OrderResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      422   {object}  dto.ErrorResponse
// @Router       /orders/{id}/status [patch]
func (h *Handler) UpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := domain.Status(req.Status)
	order, err := h.service.UpdateOrderStatus(c.Request.Context(), id, status)
	if err != nil {
		switch err {
		case domain.ErrOrderNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		case domain.ErrInvalidStatus:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		case domain.ErrSameStatus:
			c.JSON(http.StatusBadRequest, gin.H{"error": "same status"})
		case domain.ErrInvalidStatusTransition:
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid status transition"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, dto.FromDomain(order))
}
