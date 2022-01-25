package brokers

import (
	"encoding/json"
	"fmt"
	"github.com/EgorBessonov/gRPC/internal/model"
	"github.com/segmentio/kafka-go"
)

type KafkaClient struct {
	Connection *kafka.Conn
}
type KafkaReader struct {
	Reader *kafka.Reader
}

func NewKafkaClient(conn *kafka.Conn) *KafkaClient {
	return &KafkaClient{Connection: conn}
}

func NewKafkaReader(reader *kafka.Reader) *KafkaReader {
	return &KafkaReader{Reader: reader}
}

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
