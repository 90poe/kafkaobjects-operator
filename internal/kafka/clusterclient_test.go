package kafka

import (
	"regexp"
	"testing"

	api "github.com/90poe/kafkaobjects-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestCreateTopic(t *testing.T) {
	tests := []struct {
		name          string
		topic         *api.KafkaTopicSpec
		maxPartitions uint
		namePattern   string
		wantErr       string
	}{
		{
			name: "too many partitions",
			topic: &api.KafkaTopicSpec{
				Name:       "test-topic",
				Partitions: 10,
			},
			maxPartitions: 5,
			namePattern:   ".*",
			wantErr:       "test-topic can't have more partitions than 5",
		},
		{
			name: "invalid topic name pattern",
			topic: &api.KafkaTopicSpec{
				Name:       "invalid@topic",
				Partitions: 3,
			},
			maxPartitions: 5,
			namePattern:   "^[a-zA-Z0-9-_.]+$",
			wantErr:       "topic name `invalid@topic` doesn't match pattern `^(dev|test|int|prod)_(op|perf|chart|onboard|mms|onradar|crew|oos|ci|proc|devops|hsqe)_(dms|ben|voe)_(.+)_(.+)_(.+).*$`",
		},
		{
			name: "nil kafka client",
			topic: &api.KafkaTopicSpec{
				Name:       "test-topic",
				Partitions: 3,
			},
			maxPartitions: 5,
			namePattern:   ".*",
			wantErr:       "we don't have connection to Kafka cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterClient{
				maxPartsPerTopic: tt.maxPartitions,
				topicNamePattern: regexp.MustCompile(tt.namePattern),
			}

			err := c.CreateTopic(tt.topic)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}
