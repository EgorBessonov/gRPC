package main

import (
	"context"
	"fmt"
	"github.com/EgorBessonov/gRPC/internal/broker"
	"github.com/EgorBessonov/gRPC/internal/cache"
	"github.com/EgorBessonov/gRPC/internal/config"
	ordercrud "github.com/EgorBessonov/gRPC/internal/protocol"
	"github.com/EgorBessonov/gRPC/internal/repository"
	"github.com/EgorBessonov/gRPC/internal/server"
	"github.com/EgorBessonov/gRPC/internal/service"
	"github.com/caarlos0/env"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"net"
	"time"
)

const (
	kafkaDeadLine = 10
	kafkaMinBytes = 10e3
	kafkaMaxBytes = 10e6
)

func main() {
	cfg := config.Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("config parsing failed")
	}
	repos := dbConnection(cfg)
	conn, rabbitCli, err := rabbitConnection(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := rabbitCli.Channel.Close(); err != nil {
			log.Errorf("rabbitmq: error while closing channel - %e", err)
		}
		if err := conn.Close(); err != nil {
			log.Errorf("rabbitmq: error while closing connection - %e, err")
		}
	}()
	cacheContext := context.Background()
	kafkaConn, err := kafkaConnection(&cfg)
	if err != nil {
		log.Fatal("kafka: connection failed - %e", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Error("kafka: error while closing connection")
		}
	}()
	kReader, err := kafkaReader(&cfg)
	if err != nil {
		log.Fatal("kafka: error while creating reader - %e", err)
	}
	orderCache := cache.NewCache(cacheContext, broker.NewKafkaClient(kafkaConn), broker.NewKafkaReader(kReader), cfg.RabbitQueueName, rabbitCli)
	orderService := service.NewService(repos, orderCache)
	gRPCServer := server.NewServer(orderService)
	newgRPCServer(cfg.PortgRPC, gRPCServer)
}

// return new rabbit client instance
func rabbitConnection(cfg config.Config) (*amqp.Connection, *broker.RabbitClient, error) {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.RabbitUser, cfg.RabbitPassword, cfg.RabbitHost, cfg.RabbitPort))
	if err != nil {
		return conn, nil, fmt.Errorf("rabbitmq: error while creating connection - %e", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		return conn, nil, fmt.Errorf("rabbitmq: error while creating channel - %e", err)
	}
	q, err := ch.QueueDeclare(
		cfg.RabbitQueueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return conn, nil, fmt.Errorf("rabbitmq: error while creating queue - %e", err)
	}
	rabbitClient := broker.NewRabbit(ch, &q)
	log.Printf("rabbitmq: succsfully connected at %s:%s", cfg.RabbitHost, cfg.RabbitPort)
	return conn, rabbitClient, nil
}

// return new kafka connection
func kafkaConnection(cfg *config.Config) (*kafka.Conn, error) {
	conn, err := kafka.DialLeader(context.Background(), "tcp", fmt.Sprint(cfg.KafkaHost+":"+cfg.KafkaPort), cfg.KafkaTopic, 0)
	if err != nil {
		return nil, err
	}
	err = conn.SetWriteDeadline(time.Now().Add(time.Second * kafkaDeadLine))
	if err != nil {
		return nil, err
	}
	log.Printf("kafka: successfully connected at %s:%s", cfg.KafkaHost, cfg.KafkaPort)
	return conn, nil
}

// return kafka reader instance
func kafkaReader(cfg *config.Config) (*kafka.Reader, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{fmt.Sprint("%s:%s", cfg.KafkaHost, cfg.KafkaPort)},
		GroupID:        cfg.KafkaGroupID,
		Topic:          cfg.KafkaTopic,
		MinBytes:       kafkaMinBytes,
		MaxBytes:       kafkaMaxBytes,
		CommitInterval: time.Second,
	})
	log.Print("kafka: successfully created reader")
	return reader, nil
}

// create new postgresdb connection
func dbConnection(cfg config.Config) repository.Repository {
	conn, err := pgxpool.Connect(context.Background(), cfg.PostgresdbURL)
	if err != nil {
		log.WithFields(log.Fields{
			"status": "connection to postgres database failed.",
			"err":    err,
		}).Info("postgres repository info.")
	} else {
		log.WithFields(log.Fields{
			"status": "successfully connected to postgres database.",
		}).Info("postgres repository info.")
	}
	return repository.PostgresRepository{DBconn: conn}
}

// start gRPC server
func newgRPCServer(port string, s *server.Server) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("gRPC server failed - ", err)
	}
	gServer := grpc.NewServer(grpc.UnaryInterceptor(unaryInterceptor))
	ordercrud.RegisterCRUDServer(gServer, s)
	fmt.Printf("gRPC server listening at %v", lis.Addr())
	if err = gServer.Serve(lis); err != nil {
		log.Fatal("gRPC server failed - ", err)
		return
	}
}

// create interceptor for jwt authentication
func unaryInterceptor(ctx context.Context, request interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	switch info.FullMethod {
	case "/protocol.CRUD/GetOrder", "/protocol.CRUD/SaveOrder", "/protocol.CRUD/UpdateOrder":
		if ok, err := service.ValidateToken(ctx); !ok {
			log.Errorf("server: %e", err)
			return nil, err
		}
		return handler(ctx, request)
	default:
		return handler(ctx, request)
	}
}
