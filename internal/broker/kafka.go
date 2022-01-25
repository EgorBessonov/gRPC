package brokers

import (
	"encoding/json"
	"fmt"
	"github.com/EgorBessonov/gRPC/internal/model"
	"github.com/segmentio/kafka-go"
)

// KafkaClient represent kafka client structure
type KafkaClient struct {
	Connection *kafka.Conn
}

// KafkaReader represent kafka reader structure
type KafkaReader struct {
	Reader *kafka.Reader
}

// NewKafkaClient return new KafkaClient instance
func NewKafkaClient(conn *kafka.Conn) *KafkaClient {
	return &KafkaClient{Connection: conn}
}

// NewKafkaReader return new KafkaReader instance
func NewKafkaReader(reader *kafka.Reader) *KafkaReader {
	return &KafkaReader{Reader: reader}
}

// PublishMessage send message to kafka queue
func (kCli *KafkaClient) PublishMessage(method string, order *model.Order) error {
	message := model.OrderMessage{
		Method: method,
		Data:   order,
	}
	msg, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("kafka: publishing failed - %e", err)
	}
	_, err = kCli.Connection.WriteMessages(
		kafka.Message{Value: []byte(msg)})
	return nil
}
