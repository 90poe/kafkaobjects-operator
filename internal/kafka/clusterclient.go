package kafka

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	api "github.com/90poe/kafkaobjects-operator/api/v1alpha1"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type (
	// ClusterClient will abstract work with Kafka clusters
	ClusterClient struct {
		kCl              *kgo.Client
		maxPartsPerTopic uint
		topicNamePattern *regexp.Regexp
	}
)

// Close will close kafka connection to cleanup resources
func (c *ClusterClient) Close() {
	c.kCl.Close()
}

// TopicExists will check if topic exists in Kafka cluster
func (c *ClusterClient) TopicExists(topic *api.KafkaTopicSpec) (bool, error) {
	// 1. Lets get topic list. We don't insert topic if it already exists
	topics, err := c.getTopics()
	if err != nil {
		return false, fmt.Errorf("can't get topics: %w", err)
	}
	for _, t := range topics {
		if t == topic.Name {
			// we do not insert topic if it already exists
			return true, nil
		}
	}
	return false, nil
}

// getTopics would return Kafka topics
func (c *ClusterClient) getTopics() ([]string, error) {
	kAdm := kadm.NewClient(c.kCl)
	list, err := kAdm.ListTopics(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list topics: %w", err)
	}
	topics := make([]string, 0, len(list))
	for nOfTopic := range list {
		topics = append(topics, nOfTopic)
	}
	return topics, nil
}

// CreateTopic is going to create Kafka topic from data from Structures
func (c *ClusterClient) CreateTopic(topic *api.KafkaTopicSpec) error {
	if topic.Partitions > c.maxPartsPerTopic {
		return fmt.Errorf("%s can't have more partitions than %d", topic.Name, c.maxPartsPerTopic)
	}
	// https://ninetypercent.atlassian.net/browse/DEVOPS-6286
	if !c.topicNamePattern.MatchString(topic.Name) {
		return fmt.Errorf("topic name `%s` doesn't match pattern `%s`", topic.Name, c.topicNamePattern.String())
	}
	if c.kCl == nil {
		return fmt.Errorf("we don't have connection to Kafka cluster")
	}
	kAdm := kadm.NewClient(c.kCl)
	// topic config
	configs := make(map[string]*string, 6)
	configs["min.insync.replicas"] = kadm.StringPtr(
		fmt.Sprintf("%d", topic.MinInSyncReplicas))
	// Retention MS
	retentionMS := "-1"
	if topic.RetentionHours > 0 {
		retentionMS = fmt.Sprintf("%d", topic.RetentionHours*3600*1000)
	}
	configs["retention.ms"] = &retentionMS
	// Retention Bytes
	retentionBytes := "-1"
	if topic.RetentionBytes > 0 {
		retentionBytes = fmt.Sprintf("%d", topic.RetentionBytes)
	}
	configs["retention.bytes"] = &retentionBytes

	if topic.Segment.Bytes != 0 {
		segmentBytes := fmt.Sprintf("%d", topic.Segment.Bytes)
		configs["segment.bytes"] = &segmentBytes
	}

	if topic.Segment.MS != 0 {
		segmentMS := fmt.Sprintf("%d", topic.Segment.MS)
		configs["segment.ms"] = &segmentMS
	}

	if len(topic.CleanupPolicy) != 0 {
		policy := strings.ToLower(topic.CleanupPolicy)
		configs["cleanup.policy"] = &policy
	}

	maxMessageBytes := "1048576"
	if topic.MaxMessageBytes != 1048576 {
		maxMessageBytes = fmt.Sprintf("%d", topic.MaxMessageBytes)
	}
	configs["max.message.bytes"] = &maxMessageBytes

	resp, err := kAdm.CreateTopic(
		context.Background(),
		int32(topic.Partitions),  // nolint: gosec
		int16(topic.Replication), // nolint: gosec
		configs,
		topic.Name,
	)

	if err != nil {
		return fmt.Errorf("can't create topic: %w", err)
	}
	if resp.Err != nil {
		return fmt.Errorf("can't create topic, cluster err: %w", err)
	}
	return nil
}

// alertConfig would return AlterConfig for topic
func (c *ClusterClient) alertConfig(name, value string) kadm.AlterConfig {
	return kadm.AlterConfig{
		Op:    kadm.SetConfig,
		Name:  name,
		Value: &value,
	}
}

// UpdateTopic is going to update Kafka topic from data from Structures
func (c *ClusterClient) UpdateTopic(topic *api.KafkaTopicSpec) error {
	if topic.Partitions > c.maxPartsPerTopic {
		return fmt.Errorf("%s can't have more partitions than %d", topic.Name, c.maxPartsPerTopic)
	}
	if c.kCl == nil {
		return fmt.Errorf("we don't have connection to Kafka cluster")
	}
	kAdm := kadm.NewClient(c.kCl)
	// topic config
	configs := make([]kadm.AlterConfig, 0, 5)
	configs = append(configs, c.alertConfig("min.insync.replicas", fmt.Sprintf("%d", topic.MinInSyncReplicas)))
	// Retention MS
	retentionMS := "-1"
	if topic.RetentionHours > 0 {
		retentionMS = fmt.Sprintf("%d", topic.RetentionHours*3600*1000)
	}
	configs = append(configs, c.alertConfig("retention.ms", retentionMS))
	// Retention Bytes
	retentionBytes := "-1"
	if topic.RetentionBytes > 0 {
		retentionBytes = fmt.Sprintf("%d", topic.RetentionBytes)
	}
	configs = append(configs, c.alertConfig("retention.bytes", retentionBytes))

	if topic.Segment.Bytes != 0 {
		configs = append(configs, c.alertConfig("segment.bytes", fmt.Sprintf("%d", topic.Segment.Bytes)))
	}

	if topic.Segment.MS != 0 {
		configs = append(configs, c.alertConfig("segment.ms", fmt.Sprintf("%d", topic.Segment.MS)))
	}

	if len(topic.CleanupPolicy) != 0 {
		configs = append(configs, c.alertConfig("cleanup.policy", strings.ToLower(topic.CleanupPolicy)))
	}

	maxMessageBytes := "1048576"
	if topic.MaxMessageBytes != 1048576 {
		maxMessageBytes = fmt.Sprintf("%d", topic.MaxMessageBytes)
	}
	configs = append(configs, c.alertConfig("max.message.bytes", maxMessageBytes))

	resp, err := kAdm.AlterTopicConfigsState(
		context.Background(),
		configs,
		topic.Name,
	)
	// read any other errors
	for _, r := range resp {
		if r.Err != nil {
			err = errors.Join(err, r.Err)
		}
	}
	if err != nil {
		return fmt.Errorf("can't create topic: %w", err)
	}

	return nil
}
