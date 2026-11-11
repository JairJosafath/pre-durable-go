// Package env reads the settings a program gets from its environment.
package env

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// ErrNotSet reports a required environment variable without a value.
var ErrNotSet = errors.New("environment variable is not set")

// Require returns the value of an environment variable that must be set.
func Require(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("%w: %s", ErrNotSet, name)
	}

	return value, nil
}

// Duration returns a duration such as "24h" from the environment, or the
// fallback when the variable is not set.
func Duration(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s: %w", name, err)
	}

	return duration, nil
}
