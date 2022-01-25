package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/EgorBessonov/gRPC/internal/brokers"
	"github.com/EgorBessonov/gRPC/internal/model"
	log "github.com/sirupsen/logrus"
	"sync"
)

type OrderCache struct {
	orders      map[string]*model.Order
	rabbitCli   *brokers.RabbitClient
	kafkaReader *brokers.KafkaReader
	kafkaCli    *brokers.KafkaClient
	mutex       sync.Mutex
}

func NewCache(ctx context.Context, kafkaCli *brokers.KafkaClient, kafkaReader *brokers.KafkaReader, rabbitQueueName string, rabbitCli *brokers.RabbitClient) *OrderCache {
	var cache OrderCache
	cache.orders = make(map[string]*model.Order)
	cache.rabbitCli = rabbitCli
	cache.kafkaCli = kafkaCli
	cache.kafkaReader = kafkaReader
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msgs, err := rabbitCli.Channel.Consume(
					rabbitQueueName,
					"",
					true,
					false,
					false,
					false,
					nil)
				if err != nil {
					log.Errorf("rabbitmq consumer: %e", err)
				}
				for d := range msgs {
					message := model.OrderMessage{}
					if err := json.Unmarshal(d.Body, &message); err != nil {
						log.Errorf("rabbitmq consumer: error while parsing message - %e", err)
					}
					if message.Method != "" {
						err = cache.brokerHandler(message.Method, message.Data)
						if err != nil {
							log.Errorf("rebbitmq handler: %e", err)
						}
					}
				}
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := kafkaReader.Reader.ReadMessage(context.Background())
				if err != nil {
					log.Errorf("kafka consumer: %e", err)
				}
				message := model.OrderMessage{}
				err = json.Unmarshal(msg.Value, &message)
				if err != nil {
					log.Errorf("kafka consumer: error while parsing message - %e", err)
				}
				if message.Method != "" {
					err := cache.brokerHandler(message.Method, message.Data)
					if err != nil {
						log.Errorf("kafka handler: %e", err)
					}
				}
			}
		}
	}()
	return &cache
}

func (orderCache *OrderCache) Get(orderID string) (*model.Order, bool) {
	orderCache.mutex.Lock()
	defer orderCache.mutex.Unlock()
	order, found := orderCache.orders[orderID]
	return order, found
}

//Save method send message to redis stream for saving order
func (orderCache *OrderCache) Save(order *model.Order) error {
	return orderCache.rabbitCli.PublishMessage("save", order)
}

// Update method send message to redis stream for updating order
func (orderCache *OrderCache) Update(order *model.Order) error {
	return orderCache.rabbitCli.PublishMessage("update", order)
}

// Delete method send message to redis stream for removing order
func (orderCache *OrderCache) Delete(orderID string) error {
	return orderCache.rabbitCli.PublishMessage("delete", &model.Order{OrderID: orderID})
}

func (orderCache *OrderCache) brokerHandler(method string, order *model.Order) error {
	orderCache.mutex.Lock()
	defer orderCache.mutex.Unlock()
	switch method {
	case "save", "update":
		orderCache.orders[order.OrderID] = order
		return nil
	case "delete":
		delete(orderCache.orders, order.OrderID)
		return nil
	default:
		return fmt.Errorf("cache handler: invalid method type")
	}
}
