package kafka

import (
	"crypto/tls"
	"fmt"
	"regexp"
	"strings"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kversion"
)

const (
	TopicNamePatternStr = `^(dev|test|int|prod)_(op|perf|chart|onboard|mms|onradar|crew|oos|ci|proc|devops|hsqe)_(dms|ben|voe)_(.+)_(.+)_(.+).*$`
)

// Option is a type of options for ClusterConfig
type (
	// ClusterConfig will configure settings for Kafka cluster and it will get a new ClusterCient
	ClusterConfig struct {
		kOpts                 []kgo.Opt
		tlsCACert             string
		tlsCert               string
		tlsKey                string
		brokers               []string
		tlsEnabled            bool
		tlsInsecureSkipVerify bool
		maxPartsPerTopic      uint
		topicNamePattern      *regexp.Regexp
	}
	Option func(*ClusterConfig) error
)

// Brokers is option function to set list of brokers
func Brokers(brokers string) Option {
	return func(m *ClusterConfig) error {
		if len(brokers) == 0 {
			return fmt.Errorf("brokers of ClusterConfig can't be empty")
		}
		m.brokers = strings.Split(brokers, ",")
		m.kOpts = append(m.kOpts, kgo.SeedBrokers(m.brokers...))
		return nil
	}
}

// TLSEnabled is option function to set if we need TLS or not
func TLSEnabled(tlsEnabled bool) Option {
	return func(m *ClusterConfig) error {
		m.tlsEnabled = tlsEnabled
		return nil
	}
}

// TLSSkipVerify is option function to set if we need to verify clusters cert
func TLSSkipVerify(tlsSkip bool) Option {
	return func(m *ClusterConfig) error {
		m.tlsInsecureSkipVerify = tlsSkip
		return nil
	}
}

// TLSCACert is option function to set CA cert
func TLSCACert(ca string) Option {
	return func(m *ClusterConfig) error {
		m.tlsCACert = ca
		return nil
	}
}

// TLSCert is option function to set TLS cert
func TLSCert(cert string) Option {
	return func(m *ClusterConfig) error {
		m.tlsCert = cert
		return nil
	}
}

// TLSKey is option function to set TLS key
func TLSKey(key string) Option {
	return func(m *ClusterConfig) error {
		m.tlsKey = key
		return nil
	}
}

// MaxPartsPerTopic is option function to set Maximum Partitions per Topic
func MaxPartsPerTopic(max uint) Option {
	return func(m *ClusterConfig) error {
		m.maxPartsPerTopic = max
		return nil
	}
}

// NewClusterConfig would return ClusterClient or would return error if error occured
func NewClusterConfig(options ...Option) (*ClusterConfig, error) {
	// initialise cluster client
	c := &ClusterConfig{
		kOpts: make([]kgo.Opt, 0),
	}
	// max version will keep our client talking lower protocol version
	// for better compatibility
	c.kOpts = append(c.kOpts, kgo.MaxVersions(kversion.V2_5_0()))
	// settup from options
	for _, option := range options {
		err := option(c)
		if err != nil {
			return nil, fmt.Errorf("can't make new ClusterClient: %w", err)
		}
	}
	// verify all settings are correct
	err := c.verify()
	if err != nil {
		return nil, fmt.Errorf("can't verify ClusterClient: %w", err)
	}
	// if we have tls connection to Kafka cluster enabled, set it up
	if c.tlsEnabled {
		var tlsConfig *tls.Config
		tlsConfig, err = parseCerts(c.tlsCACert, c.tlsKey, c.tlsCert)
		if err != nil {
			return nil, fmt.Errorf("failed to create tls config for Kafka: %w", err)
		}
		tlsConfig.InsecureSkipVerify = c.tlsInsecureSkipVerify
		c.kOpts = append(c.kOpts, kgo.DialTLSConfig(tlsConfig))
	}
	// if all settings are correct - we will test getting Kafka client
	_, err = c.GetClient()
	if err != nil {
		return nil, err
	}
	c.topicNamePattern = regexp.MustCompile(TopicNamePatternStr)
	return c, nil
}

func (c *ClusterConfig) verify() error {
	if c.tlsEnabled && (len(c.tlsCACert) == 0 || len(c.tlsCert) == 0 || len(c.tlsKey) == 0) {
		return fmt.Errorf("TLS enabled but not all certs provided")
	}
	return nil
}

// GetClient will make a new Kafka clients
func (c *ClusterConfig) GetClient() (*ClusterClient, error) {
	var err error
	cl := &ClusterClient{
		maxPartsPerTopic: c.maxPartsPerTopic,
		topicNamePattern: c.topicNamePattern,
	}
	cl.kCl, err = kgo.NewClient(c.kOpts...)
	if err != nil {
		return nil, fmt.Errorf("can't make KafkaCluster client: %w", err)
	}
	return cl, nil
}
