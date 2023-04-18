package env

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

// generic constraint interface
type Value interface {
	string | int | bool
}

// GetEnv would get Env variable
func GetEnv[T Value](name string) (T, error) {
	var ref T
	value, found := os.LookupEnv(name)
	if !found {
		return ref, fmt.Errorf("environment variable '%s' is missing", name)
	}
	var ret interface{}
	switch reflect.ValueOf(ref).Kind() {
	case reflect.Int:
		nr, err := strconv.Atoi(value)
		if err != nil {
			return ref, fmt.Errorf("can't convert '%s' to Int: %w", value, err)
		}
		ret = nr
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return ref, fmt.Errorf("can't convert '%s' to bool: %w", value, err)
		}
		ret = b
	default:
		ret = value
	}
	return ret.(T), nil
}
