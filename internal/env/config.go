package env

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	KafkaBrokers             string `env:"KAFKA_BROKERS" env-required` // nolint
	MaxKafkaTopicsPartitions uint   `env:"KAFKA_TOPIC_MAX_PARTITIONS" env-default:"3"`
	KafkaTopicNameRegexp     string `env:"KAFKA_TOPIC_NAME_REGEXP" env-default:".*"`
	SchemaRegistryURL        string `env:"SCHEMA_REGISTRY_URL" env-required` // nolint
	MaxConcurrentReconciles  int    `env:"MAX_CONCURRENT_RECONCILES" env-default:"2"`
	LabelSelectorsInt        string `env:"LABEL_SELECTOR" env-required` // nolint
	SlackToken               string `env:"SLACK_TOKEN"`
	SlackChannel             string `env:"SLACK_CHANNEL" env-default:"empty"`
	LabelSelectors           *metav1.LabelSelector
}

func NewConfig() (*Config, error) {
	// get config from env
	conf := &Config{}
	err := cleanenv.ReadEnv(conf)
	if err != nil {
		return nil, fmt.Errorf("can't make Config: %w", err)
	}
	// parse label selector
	conf.LabelSelectors, err = metav1.ParseToLabelSelector(conf.LabelSelectorsInt)
	if err != nil {
		return nil, fmt.Errorf("can't parse label selector %s: %w", conf.LabelSelectorsInt, err)
	}
	return conf, nil
}
