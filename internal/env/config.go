package env

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// KafkaBrokersEnvKey holds env key, which has value for Kafka brokers
	KafkaBrokersEnvKey = "KAFKA_BROKERS"
	// MaxKafkaTopicsPartitionsEnvKey holds env key, which has value for
	// Kafka topic max partitions
	MaxKafkaTopicsPartitionsEnvKey = "KAFKA_TOPIC_MAX_PARTITIONS"
	// DefaultMaxParts is default for Max kafka topic partitions
	DefaultMaxParts = 3
	// SchemaRegistryURLEnvKey holds env key, which has value for Schema registry URL
	SchemaRegistryURLEnvKey = "SCHEMA_REGISTRY_URL"
	// MaxConcurrentReconcilesEnvKey holds env key, which has value for Maximum parallel reconciliations done
	MaxConcurrentReconcilesEnvKey  = "MAX_CONCURRENT_RECONCILES"
	DefaultMaxConcurrentReconciles = 2
	// LabelSelectorEnvKey will have label selector, which will make us work only with objects, which have those labels added. If not defined, we will work with ALL objects in K8S cluster.
	LabelSelectorEnvKey = "LABEL_SELECTOR"
	// SlackWebhookURLEnvKey holds env key, which has value for Slack webhook URL
	SlackTokenEnvKey   = "SLACK_TOKEN"
	SlackChannelEnvKey = "SLACK_CHANNEL"
)

type Config struct {
	KafkaBrokers             string
	MaxKafkaTopicsPartitions uint
	SchemaRegistryURL        string
	MaxConcurrentReconciles  int
	LabelSelectors           *metav1.LabelSelector
	SlackToken               string
	SlackChannel             string
}

func NewConfig() (*Config, error) {
	conf := &Config{
		MaxKafkaTopicsPartitions: DefaultMaxParts,
		MaxConcurrentReconciles:  DefaultMaxConcurrentReconciles,
		// LabelSelectors is initialized with empty predicate
		LabelSelectors: &metav1.LabelSelector{},
	}
	var err error
	// get logger
	logger := log.FromContext(context.Background())
	// Get Kafka brokers list
	conf.KafkaBrokers, err = GetEnv[string](KafkaBrokersEnvKey)
	if err != nil {
		return nil, fmt.Errorf("can't make Config: %w", err)
	}
	// Get Kafka MaxKafkaTopicsPartitions
	max, err := GetEnv[int](MaxKafkaTopicsPartitionsEnvKey)
	if (err != nil) || (max <= 0) {
		logger.Info(fmt.Sprintf("setting max topics partitions to default %d", DefaultMaxParts))
	} else {
		conf.MaxKafkaTopicsPartitions = uint(max)
	}
	// Get Operator max reconcile loops
	max, err = GetEnv[int](MaxConcurrentReconcilesEnvKey)
	if (err != nil) || (max <= 0) {
		logger.Info(fmt.Sprintf("setting max reconcilers to default %d", DefaultMaxConcurrentReconciles))
	} else {
		conf.MaxConcurrentReconciles = max
	}
	// Get Kafka Schema Regsitry URL
	conf.SchemaRegistryURL, err = GetEnv[string](SchemaRegistryURLEnvKey)
	if err != nil {
		return nil, fmt.Errorf("can't make Config: %w", err)
	}
	// Get optional label selector
	labelSelectors, err := GetEnv[string](LabelSelectorEnvKey)
	if err == nil {
		// let's parse it
		conf.LabelSelectors, err = metav1.ParseToLabelSelector(labelSelectors)
		if err != nil {
			return nil, fmt.Errorf("can't parse `%s` as label selectors: %w", labelSelectors, err)
		}
	}
	// Get optional Slack webhook URL
	conf.SlackToken, _ = GetEnv[string](SlackTokenEnvKey) // nolint:errcheck
	conf.SlackChannel, err = GetEnv[string](SlackChannelEnvKey)
	if err != nil {
		conf.SlackChannel = "empty"
	}
	return conf, nil
}
