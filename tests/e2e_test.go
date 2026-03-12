package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	httpadapter "order-service/internal/adapters/http"
	"order-service/internal/adapters/http/dto"
	"order-service/internal/adapters/mongodb"
	"order-service/internal/adapters/rabbitmq"
	apporder "order-service/internal/application/order"
	"order-service/internal/config"
	mongoclient "order-service/internal/pkg/mongo"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type E2ESuite struct {
	suite.Suite
	mongo       *MongoContainer
	rabbit      *RabbitMQContainer
	cfg         *config.Config
	router      *gin.Engine
	mongoClient mongoclient.Client
}

func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ESuite))
}

func (s *E2ESuite) SetupSuite() {
	mc, err := StartMongoContainer()
	s.Require().NoError(err)
	s.mongo = mc

	rc, err := StartRabbitMQContainer()
	s.Require().NoError(err)
	s.rabbit = rc

	cfg := &config.Config{}
	cfg.App.Port = 8080
	cfg.App.Env = "test"
	cfg.MongoDB.URI = mc.URI
	cfg.MongoDB.Database = "orders_db"
	cfg.MongoDB.TimeoutSeconds = 10
	cfg.RabbitMQ.URI = rc.URI
	cfg.RabbitMQ.Exchange = "orders"
	cfg.RabbitMQ.Queue = "order-status-events"
	cfg.RabbitMQ.RoutingKey = "order.status.updated"
	cfg.Logger.Level = "info"
	s.cfg = cfg

	logger := zap.NewNop()

	var repo *mongodb.OrderRepository

	s.Require().NoError(Retry(mc.Pool, func(ctx context.Context) error {
		client, err := mongoclient.New(ctx, cfg)
		if err != nil {
			return err
		}

		s.mongoClient = client
		repo = mongodb.NewOrderRepository(client.Database())
		return nil
	}))

	var pub *rabbitmq.Publisher
	s.Require().NoError(Retry(rc.Pool, func(ctx context.Context) error {
		p, err := rabbitmq.NewPublisher(cfg, logger)
		if err != nil {
			return err
		}
		pub = p
		return nil
	}))

	service := apporder.NewService(repo, pub)
	handler := httpadapter.NewHandler(service, s.mongoClient)
	router := httpadapter.NewRouter(handler, logger)

	s.router = router
}

func (s *E2ESuite) TearDownSuite() {
	if s.mongoClient != nil {
		_ = s.mongoClient.Close(context.Background())
	}
	if s.mongo != nil && s.mongo.Pool != nil && s.mongo.Resource != nil {
		_ = s.mongo.Pool.Purge(s.mongo.Resource)
	}
	if s.rabbit != nil && s.rabbit.Pool != nil && s.rabbit.Resource != nil {
		_ = s.rabbit.Pool.Purge(s.rabbit.Resource)
	}
}

func (s *E2ESuite) ExecuteRequest(req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)
	return w
}

func (s *E2ESuite) TestOrderFlow() {
	// POST /orders
	orderRequest := dto.CreateOrderRequest{
		CustomerID: "customer-42",
		Items: []dto.CreateOrderItem{
			{
				ProductID:   "prod-01",
				ProductName: "Tênis Runner X",
				Quantity:    2,
				UnitPrice:   199.90,
			},
		},
	}

	body, err := json.Marshal(orderRequest)
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	resp := s.ExecuteRequest(req)
	s.Equal(http.StatusCreated, resp.Code)

	var created struct {
		ID string `json:"id"`
	}
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&created))
	s.NotEmpty(created.ID)

	// PATCH /orders/:id/status para em_processamento
	updateStatusRequest := dto.UpdateStatusRequest{
		Status: "em_processamento",
	}
	body, err = json.Marshal(updateStatusRequest)
	s.Require().NoError(err)

	path := fmt.Sprintf("/orders/%s/status", created.ID)
	req = httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(body))
	resp = s.ExecuteRequest(req)
	s.Equal(http.StatusOK, resp.Code)

	// PATCH /orders/:id/status pulando diretamente para enviado
	updateStatusRequest = dto.UpdateStatusRequest{
		Status: "enviado",
	}
	body, err = json.Marshal(updateStatusRequest)
	s.Require().NoError(err)

	req = httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(body))
	resp = s.ExecuteRequest(req)
	s.Equal(http.StatusOK, resp.Code)

	// GET /orders/:id
	path = fmt.Sprintf("/orders/%s", created.ID)
	req = httptest.NewRequest(http.MethodGet, path, nil)
	resp = s.ExecuteRequest(req)
	s.Equal(http.StatusOK, resp.Code)

	var got struct {
		Status string `json:"status"`
	}
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&got))
	s.Equal("enviado", got.Status)

	// GET /health
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	resp = s.ExecuteRequest(req)
	s.Equal(http.StatusOK, resp.Code)
}
