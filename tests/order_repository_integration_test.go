package tests

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"order-service/internal/adapters/mongodb"
	domain "order-service/internal/domain/order"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type OrderRepositoryIntegrationSuite struct {
	suite.Suite
	pool     *dockertest.Pool
	resource *dockertest.Resource
	repo     *mongodb.OrderRepository
	db       *mongo.Database
	client   *mongo.Client
}

func TestOrderRepositoryIntegrationSuite(t *testing.T) {
	suite.Run(t, new(OrderRepositoryIntegrationSuite))
}

func (s *OrderRepositoryIntegrationSuite) SetupSuite() {
	mc, err := StartMongoContainer()
	s.Require().NoError(err)

	s.pool = mc.Pool
	s.resource = mc.Resource

	var client *mongo.Client
	var db *mongo.Database

	s.Require().NoError(Retry(mc.Pool, func(ctx context.Context) error {
		c, err := mongo.Connect(options.Client().ApplyURI(mc.URI))
		if err != nil {
			return err
		}
		if err := c.Ping(ctx, nil); err != nil {
			return err
		}

		client = c
		db = client.Database("orders_db")
		return nil
	}))

	s.client = client
	s.db = db
	s.repo = mongodb.NewOrderRepository(db)
}

func (s *OrderRepositoryIntegrationSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Disconnect(context.Background())
	}
	if s.pool != nil && s.resource != nil {
		_ = s.pool.Purge(s.resource)
	}
}

func (s *OrderRepositoryIntegrationSuite) SetupTest() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.db.Collection("orders").Drop(ctx)
	s.Require().NoError(err)
}

func (s *OrderRepositoryIntegrationSuite) TestCreateAndGetByID() {
	now := time.Now()
	o, err := domain.NewOrder("customer-1", []domain.OrderItem{
		{ProductID: "p1", ProductName: "Produto 1", Quantity: 1, UnitPrice: 10},
	}, now)
	s.Require().NoError(err)

	ctx := context.Background()
	err = s.repo.Create(ctx, o)
	s.Require().NoError(err)

	found, err := s.repo.GetByID(ctx, o.ID.Hex())
	s.Require().NoError(err)
	s.Equal(o.CustomerID, found.CustomerID)
	s.Equal(o.TotalPrice, found.TotalPrice)
}

func (s *OrderRepositoryIntegrationSuite) TestUpdateStatus() {
	now := time.Now()
	o, err := domain.NewOrder("customer-1", []domain.OrderItem{
		{ProductID: "p1", ProductName: "Produto 1", Quantity: 1, UnitPrice: 10},
	}, now)
	s.Require().NoError(err)

	ctx := context.Background()
	err = s.repo.Create(ctx, o)
	s.Require().NoError(err)

	err = o.UpdateStatus(domain.StatusProcessing, now.Add(time.Minute))
	s.Require().NoError(err)

	err = s.repo.Update(ctx, o)
	s.Require().NoError(err)

	found, err := s.repo.GetByID(ctx, o.ID.Hex())
	s.Require().NoError(err)
	s.Equal(domain.StatusProcessing, found.Status)
}

func (s *OrderRepositoryIntegrationSuite) TestUpdateStatusConcurrentUpdate() {
	now := time.Now()
	o, err := domain.NewOrder("customer-1", []domain.OrderItem{
		{ProductID: "p1", ProductName: "Produto 1", Quantity: 1, UnitPrice: 10},
	}, now)
	s.Require().NoError(err)

	ctx := context.Background()
	err = s.repo.Create(ctx, o)
	s.Require().NoError(err)

	var wg sync.WaitGroup
	var countConcurrentUpdates atomic.Int32

	status := []domain.Status{
		domain.StatusProcessing,
		domain.StatusDelivered,
		domain.StatusShipped,
	}
	for _, status := range status {
		wg.Go(func() {
			err = o.UpdateStatus(status, now.Add(time.Minute))
			s.Require().NoError(err)

			err = s.repo.Update(ctx, o)
			if err != nil {
				countConcurrentUpdates.Add(1)
			}
		})
	}

	wg.Wait()
	s.Require().Equal(int32(len(status)-1), countConcurrentUpdates.Load())
}

func (s *OrderRepositoryIntegrationSuite) TestGetByID_NotFound() {
	ctx := context.Background()
	_, err := s.repo.GetByID(ctx, "000000000000000000000000")
	assert.Equal(s.T(), domain.ErrOrderNotFound, err)
}
