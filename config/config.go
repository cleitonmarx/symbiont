// Package config provides a flexible configuration management system with pluggable providers.
// It supports struct field injection via tags and includes built-in parsers for common types.
package config

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/cleitonmarx/symbiont/internal/reflectx"
	"github.com/cleitonmarx/symbiont/introspection"
)

const (
	// tagName is the struct tag key for configuration value names
	tagName = "config"
	// defaultTagName is the struct tag key for default values
	defaultTagName = "default"
)

var (
	// globalProvider wraps the active configuration provider with introspection capabilities
	globalProvider *providerInspector
	// parserRegistry maps types to their parsing functions for string value conversion
	parserRegistry map[reflect.Type]func(value string) (any, error)
)

// SetGlobalProvider sets the active provider for all configuration lookups.
// The provider should be set during application initialization before runnables start.
func SetGlobalProvider(provider Provider) {
	globalProvider = newProviderInspector(provider)
}

// Provider retrieves configuration values by key.
// Implementations can read from environment variables, files, remote services, etc.
type Provider interface {
	// Get retrieves the configuration value for the given key.
	Get(ctx context.Context, name string) (string, error)
}

// ParseFunc is a function that parses a string value into type T.
// Used to convert configuration strings into typed values.
type ParseFunc[T any] func(value string) (T, error)

// RegisterParser registers a custom parser for type T.
// Built-in parsers exist for string, bool, int, int64, float64, and time.Duration.
func RegisterParser[T any](parser ParseFunc[T]) {
	parserRegistry[reflect.TypeFor[T]()] = func(value string) (any, error) {
		return parser(value)
	}
}

// getParsedConfigValue retrieves and parses a configuration value using the active provider.
func getParsedConfigValue[T any](ctx context.Context, name string, useDefault bool) (T, error) {
	emptyType := reflectx.EmptyValue[T]()
	typeOfT := reflect.TypeFor[T]()
	parser, exist := parserRegistry[typeOfT]
	if !exist {
		return emptyType, fmt.Errorf("parser for type '%s' does not exist", reflectx.GetTypeName(typeOfT))
	}
	configValue, err := globalProvider.get(ctx, name, useDefault, nil, 4)
	if err != nil {
		return emptyType, err
	}
	value, err := parser(configValue)
	if err != nil {
		return emptyType, err
	}
	return value.(T), nil
}

// Get retrieves and parses a configuration value by key and type.
// Returns an error if the key is not found or parsing fails.
func Get[T any](ctx context.Context, name string) (T, error) {
	value, err := getParsedConfigValue[T](ctx, name, false)
	if err != nil {
		return value, fmt.Errorf("config: %s", err)
	}
	return value, nil
}

// GetWithDefault retrieves a configuration value or returns the default if not found.
// No error is returned; the default is used for any lookup or parse failure.
func GetWithDefault[T any](ctx context.Context, name string, defaultValue T) T {
	value, err := getParsedConfigValue[T](ctx, name, true)
	if err != nil {
		return defaultValue
	}
	return value
}

// LoadStruct injects configuration values into all struct fields tagged with config:"key".
// Supports default values via the default tag. Returns error if a required key is not found.
func LoadStruct[T any](ctx context.Context, target *T) error {
	return reflectx.IterateStructFields(target, loadStructFieldValue(ctx))
}

// LoadStructFieldValue returns a function that injects a single struct field's configuration value.
// Used internally during struct field injection; handles tags and default values.
func LoadStructFieldValue(ctx context.Context) reflectx.StructFieldIteratorFunc {
	return loadStructFieldValue(ctx)

}

func loadStructFieldValue(ctx context.Context) reflectx.StructFieldIteratorFunc {
	return func(fieldValue reflect.Value, structField reflect.StructField, targetType reflect.Type) error {
		configName, ok := structField.Tag.Lookup(tagName)
		if !ok {
			return nil
		}

		defaultValue := structField.Tag.Get(defaultTagName)
		parser, exists := parserRegistry[structField.Type]
		if !exists {
			return fmt.Errorf("config: parser for type '%s' does not exist", reflectx.GetTypeName(structField.Type))
		}

		var (
			valueStr string
			err      error
		)
		if defaultValue != "" {
			valueStr, err = globalProvider.get(ctx, configName, true, targetType, 5)
			if err != nil {
				valueStr = defaultValue
			}
		} else {
			valueStr, err = globalProvider.get(ctx, configName, false, targetType, 5)
			if err != nil {
				return fmt.Errorf("config: error getting value for field '%s': %s", structField.Name, err)
			}
		}

		value, parseErr := parser(valueStr)
		if parseErr != nil {
			return fmt.Errorf("config: error parsing value for field '%s': %s", structField.Name, parseErr)
		}

		if err := reflectx.SetFieldValue(fieldValue, structField, value); err != nil {
			return fmt.Errorf("config: %s", err)
		}
		return nil
	}
}

// IntrospectConfigAccesses returns detailed information about all configuration keys that have been accessed.
// Useful for debugging and verifying which configurations are actually being used.
func IntrospectConfigAccesses() []introspection.ConfigAccess {
	return globalProvider.getKeysAccessInfo()
}

// ResetGlobalProvider resets the global provider to the default environment variable provider.
// Typically used in tests to isolate configuration between test cases.
func ResetGlobalProvider() {
	// Reset the global provider to the default environment variable provider.
	globalProvider = newProviderInspector(NewEnvVarProvider())
}

func init() {
	parserRegistry = map[reflect.Type]func(value string) (any, error){
		reflect.TypeFor[string]():        func(value string) (any, error) { return value, nil },
		reflect.TypeFor[bool]():          func(value string) (any, error) { return strconv.ParseBool(value) },
		reflect.TypeFor[int]():           func(value string) (any, error) { return strconv.Atoi(value) },
		reflect.TypeFor[int64]():         func(value string) (any, error) { return strconv.ParseInt(value, 10, 64) },
		reflect.TypeFor[float64]():       func(value string) (any, error) { return strconv.ParseFloat(value, 64) },
		reflect.TypeFor[time.Duration](): func(value string) (any, error) { return time.ParseDuration(value) },
	}

	globalProvider = newProviderInspector(NewEnvVarProvider())
}
