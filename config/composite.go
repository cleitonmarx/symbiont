package config

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cleitonmarx/symbiont/internal/reflectx"
)

// namedConfigProvider wraps a Provider with its type name for error reporting.
type namedConfigProvider struct {
	ConfigProvider Provider
	Name           string
}

// CompositeProvider chains multiple providers and returns the first successful value.
// Useful for fallback scenarios, e.g., environment variables with file fallback.
type CompositeProvider struct {
	providers []namedConfigProvider
}

// NewCompositeProvider creates a provider that tries each provider in order until one succeeds.
func NewCompositeProvider(providers ...Provider) CompositeProvider {
	namedConfigProviders := make([]namedConfigProvider, len(providers))
	for i, p := range providers {
		namedConfigProviders[i] = namedConfigProvider{
			ConfigProvider: p,
			Name:           reflectx.TypeNameOf(p),
		}
	}
	return CompositeProvider{
		providers: namedConfigProviders,
	}
}

// Get retrieves a configuration value from the first provider that has it.
func (p CompositeProvider) Get(ctx context.Context, name string) (string, error) {
	value, _, err := p.GetWithSource(ctx, name)
	return value, err
}

// GetWithSource retrieves a configuration value and reports which provider provided it.
func (p CompositeProvider) GetWithSource(ctx context.Context, name string) (string, string, error) {
	var errMsgs []string
	for _, provider := range p.providers {
		value, err := provider.ConfigProvider.Get(ctx, name)
		if err == nil {
			return value, provider.Name, nil
		}
		errMsgs = append(errMsgs, fmt.Sprintf("%s: %v", provider.Name, err))
	}
	return "", "", errors.New(strings.Join(errMsgs, "\n"))
}
