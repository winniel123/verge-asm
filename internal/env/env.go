// Package env reads the process environment: the only configuration web
// and worker take from outside the database (packaging-and-configuration.md
// §5.1).
package env

import (
	"fmt"
	"os"
)

func OrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func Require(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return v, nil
}
