package schemaregistry

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/90poe/kafkaobjects-operator/api/v1alpha1"
	"github.com/riferrei/srclient"
)

const (
	// SchemaRegistry specific constant
	// Compatibility modes - see https://docs.confluent.io/platform/current/schema-registry/develop/api.html#schemaregistry-api
	KafkaSchemaRegistryCompatibilityBackward      = "BACKWARD"
	KafkaSchemaRegistryCompatibilityBackwardTrans = "BACKWARD_TRANSITIVE"
	KafkaSchemaRegistryCompatibilityForward       = "FORWARD"
	KafkaSchemaRegistryCompatibilityForwardTrans  = "FORWARD_TRANSITIVE"
	KafkaSchemaRegistryCompatibilityFull          = "FULL"
	KafkaSchemaRegistryCompatibilityFullTrans     = "FULL_TRANSITIVE"
	KafkaSchemaRegistryCompatibilityNone          = "NONE"
	// Cache timeout
	CacheConnTimeout = 10 * time.Second
)

type Client struct {
	c            *srclient.SchemaRegistryClient
	schemaRegURL string
}

// Option is a type of options for Client
type Option func(*Client) error

// SchemeRegistryURL allows the Schema Registry URL to be set
func URL(url string) Option {
	return func(m *Client) error {

		// schemaregistry.NewClient usefully does a lot of
		// validation on the url - probably worth doing here
		m.c = srclient.CreateSchemaRegistryClient(url)
		m.schemaRegURL = url

		return nil
	}
}

func NewClient(options ...Option) (*Client, error) {

	client := &Client{}

	for _, option := range options {
		err := option(client)
		if err != nil {
			return nil, fmt.Errorf("error creating new schema registry client: %w", err)
		}
	}

	return client, nil
}

// CreateSchema creates or updates a schema or returns an error
func (c *Client) CreateSchema(schema *v1alpha1.KafkaSchemaSpec) error {
	// Create Schema itself
	_, err := c.c.CreateSchema(schema.Name, schema.Schema, srclient.Avro)
	if err != nil {
		return err
	}
	// Amend default compatibility mode
	if schema.Compatibility == KafkaSchemaRegistryCompatibilityBackward {
		// default compatibility nothing to do
		return nil
	}
	// curl -X PUT -H "Content-Type: application/vnd.schemaregistry.v1+json" --data '{"compatibility": "FULL"}' http://localhost:8081/config/my-kafka-value
	// https://docs.confluent.io/platform/current/schema-registry/develop/using.html#update-compatibility-requirements-on-a-subject
	data := []byte(fmt.Sprintf(`{"compatibility": "%s"}`, schema.Compatibility))
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/config/%s", c.schemaRegURL, schema.Name), bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to register request for compatibility %s for %s: %w", schema.Compatibility, schema.Name, err)
	}
	// Set headers
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")

	// Set client timeout
	client := &http.Client{Timeout: CacheConnTimeout}
	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register compatibility %s for %s: %w", schema.Compatibility, schema.Name, err)
	}
	defer resp.Body.Close()

	return err
}

// SchemaExists will check if a schema exists or return an error
func (c *Client) SchemaExists(schemaName string) (bool, error) {
	_, err := c.c.GetLatestSchema(schemaName)
	if err != nil {
		return false, nil
	}
	return true, nil
}
