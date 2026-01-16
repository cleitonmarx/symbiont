package config

import (
	"context"
	"fmt"
	"os"
)

// EnvVarProvider retrieves configuration values from environment variables.
// This is the default provider if no custom provider is set.
type EnvVarProvider struct{}

// NewEnvVarProvider creates a new environment variable configuration provider.
func NewEnvVarProvider() EnvVarProvider {
	return EnvVarProvider{}
}

// Get retrieves the environment variable value for the given name.
func (p EnvVarProvider) Get(_ context.Context, name string) (string, error) {
	value, exists := os.LookupEnv(name)
	if !exists {
		return "", fmt.Errorf("environment variable '%s' is not set", name)
	}
	return value, nil
}
