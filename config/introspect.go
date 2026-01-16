package config

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/cleitonmarx/symbiont/internal/reflectx"
	"github.com/cleitonmarx/symbiont/introspection"
)

// ProviderWithSource is an optional interface that providers can implement to report their source.
// For example, CompositeProvider reports which sub-provider supplied the value.
type ProviderWithSource interface {
	// GetWithSource retrieves a configuration value and reports its provider source.
	GetWithSource(ctx context.Context, key string) (string, string, error)
}

// providerInspector wraps a Provider and tracks all accessed keys and their sources for introspection.
type providerInspector struct {
	provider     Provider
	providerName string
	cache        map[string]string
	mu           sync.Mutex
	usedKeys     map[string][]introspection.ConfigAccess
	order        int
}

// newProviderInspector creates a new inspector wrapper for introspection and caching.
func newProviderInspector(p Provider) *providerInspector {
	return &providerInspector{
		provider:     p,
		usedKeys:     make(map[string][]introspection.ConfigAccess),
		cache:        make(map[string]string),
		providerName: reflectx.TypeNameOf(p),
	}
}

// recordKeyAccess records metadata about a configuration key access for introspection and debugging.
func (i *providerInspector) recordKeyAccess(key, provider string, isDefaultConfigured bool, componentType reflect.Type, level int) {
	callerFunc, file, line := reflectx.GetCallerName(level + 1)
	caller := reflectx.FormatFunctionName(callerFunc)
	if strings.Contains(caller, "symbiont.(*App).") {
		caller = ""
	}

	componentName := ""
	if componentType != nil {
		componentName = reflectx.GetTypeName(componentType)
	}

	if isDefaultConfigured {
		provider = ""
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	i.order++

	info := introspection.ConfigAccess{
		Key:         key,
		Provider:    provider,
		UsedDefault: isDefaultConfigured,
		Caller: introspection.Caller{
			Func: caller,
			File: reflectx.FormatFileName(file),
			Line: line,
		},
		Component: componentName,
		Order:     i.order,
	}
	i.usedKeys[key] = append(i.usedKeys[key], info)
}

// get retrieves a configuration value from the provider, caching results and recording access metadata.
func (i *providerInspector) get(ctx context.Context, key string, isUsingDefaultConfig bool, componentType reflect.Type, level int) (string, error) {
	if val, providerName, ok := i.getFromCache(key); ok {
		i.recordKeyAccess(key, providerName, isUsingDefaultConfig, componentType, level)
		return val, nil
	}

	var (
		val          string
		providerName string
		err          error
	)

	if srp, ok := i.provider.(ProviderWithSource); ok {
		val, providerName, err = srp.GetWithSource(ctx, key)
	} else {
		val, err = i.provider.Get(ctx, key)
		providerName = i.providerName
	}

	if isUsingDefaultConfig || err == nil {
		i.recordKeyAccess(key, providerName, isUsingDefaultConfig, componentType, level)
	}

	// Return error only if not using a default configuration and an error occurred.
	if err != nil && !isUsingDefaultConfig {
		return "", err
	}

	i.mu.Lock()
	i.cache[key] = val
	i.mu.Unlock()

	return val, err
}

// getFromCache retrieves a cached configuration value if available.
func (i *providerInspector) getFromCache(key string) (string, string, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	val, ok := i.cache[key]
	if !ok {
		return "", "", false
	}

	providerName := ""
	if keys, ok := i.usedKeys[key]; ok && len(keys) > 0 {
		providerName = keys[0].Provider
	}

	return val, providerName, true
}

// getKeysAccessInfo returns all accessed keys sorted by key name, file, and line number.
func (i *providerInspector) getKeysAccessInfo() []introspection.ConfigAccess {
	i.mu.Lock()
	defer i.mu.Unlock()
	out := make([]introspection.ConfigAccess, 0)
	for _, arr := range i.usedKeys {
		out = append(out, arr...)
	}
	sortConfigAccesses(out)
	return out
}

// sortConfigAccesses orders config accesses by key, then file, then line, then order.
func sortConfigAccesses(accesses []introspection.ConfigAccess) {
	sort.Slice(accesses, func(a, b int) bool {
		if accesses[a].Key == accesses[b].Key {
			if accesses[a].Caller.File == accesses[b].Caller.File {
				if accesses[a].Caller.Line == accesses[b].Caller.Line {
					return accesses[a].Order < accesses[b].Order
				}
				return accesses[a].Caller.Line < accesses[b].Caller.Line
			}
			return accesses[a].Caller.File < accesses[b].Caller.File
		}
		return accesses[a].Key < accesses[b].Key
	})
}
