package env_test

import (
	"os"
	"testing"

	"github.com/90poe/kafkaobjects-operator/internal/env"
	"github.com/stretchr/testify/assert"
)

type testEnv struct {
	key      string
	value    string
	getValue interface{}
	err      string
}

func TestMustGetEnv(t *testing.T) {
	t.Parallel()

	// Strings
	tests := []testEnv{
		{
			key:      "test1",
			value:    "test1",
			getValue: "test1",
		},
		{
			err: "environment variable",
		},
	}
	for _, test := range tests {
		if len(test.key) > 0 {
			os.Setenv(test.key, test.value)
		}
		ret, err := env.GetEnv[string](test.key)
		if len(test.err) != 0 {
			assert.ErrorContains(t, err, test.err)
			break
		}
		assert.Equal(t, test.getValue.(string), ret) // nolint: errcheck
	}
	// ints
	tests = []testEnv{
		{
			key:      "test1",
			value:    "100",
			getValue: 100,
		},
		{
			key:   "test1",
			value: "Wqss",
			err:   "can't",
		},
	}
	for _, test := range tests {
		os.Setenv(test.key, test.value)
		ret, err := env.GetEnv[int](test.key)
		if len(test.err) != 0 {
			assert.ErrorContains(t, err, test.err)
			break
		}
		assert.Equal(t, test.getValue.(int), ret) // nolint: errcheck
	}
	// bools
	tests = []testEnv{
		{
			key:      "test1",
			value:    "true",
			getValue: true,
		},
		{
			key:      "test1",
			value:    "1",
			getValue: true,
		},
		{
			key:   "test1",
			value: "Wssgqq",
			err:   "can't",
		},
	}
	for _, test := range tests {
		os.Setenv(test.key, test.value)
		ret, err := env.GetEnv[bool](test.key)
		if len(test.err) != 0 {
			assert.ErrorContains(t, err, test.err)
			break
		}
		assert.Equal(t, test.getValue.(bool), ret) // nolint: errcheck
	}
}
