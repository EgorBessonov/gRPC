package brokers

import (
	"encoding/json"
	"fmt"
	"github.com/EgorBessonov/gRPC/internal/model"
	"github.com/streadway/amqp"
)

// RabbitClient represent rabbitmq client structure
type RabbitClient struct {
	Channel *amqp.Channel
	Queue   *amqp.Queue
}

// NewRabbit return new RabbitClient instance
func NewRabbit(channel *amqp.Channel, queue *amqp.Queue) *RabbitClient {
	return &RabbitClient{Channel: channel, Queue: queue}
}

// PublishMessage send message to rabbitmq queue
func (rCli *RabbitClient) PublishMessage(method string, order *model.Order) error {
	message := model.OrderMessage{
		Method: method,
		Data:   order,
	}
	msg, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("rabbitmq: publishing failed - %e", err)
	}
	err = rCli.Channel.Publish("", rCli.Queue.Name, false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg)})
	if err != nil {
		return fmt.Errorf("rabbitmq: publishing failed - %e", err)
	}
	return nil
}
