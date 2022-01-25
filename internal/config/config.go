package config

// Config type store all env info
type Config struct {
	SecretKey       string `env:"SECRETKEY"`
	PostgresdbURL   string `env:"POSTGRESDB_URL"`
	PortgRPC        string `env:"PORTGRPC"`
	RabbitUser      string `env:"RABBITUSER"`
	RabbitPassword  string `env:"RABBITPSSWD"`
	RabbitHost      string `env:"RABBITHOST"`
	RabbitPort      string `env:"RABBITPORT"`
	RabbitQueueName string `env:"RABBITQNAME"`
	KafkaPort       string `env:"KAFKAPORT"`
	KafkaHost       string `env:"KAFKAHOST"`
	KafkaTopic      string `env:"KAFKATOPIC"`
	KafkaGroupID    string `env:"KafkaGID"`
}
