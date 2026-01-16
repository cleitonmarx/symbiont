package config

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/cleitonmarx/symbiont/internal/reflectx"
)

// ProviderWithSource is an optional interface that providers can implement to report their source.
// For example, CompositeProvider reports which sub-provider supplied the value.
type ProviderWithSource interface {
	// GetWithSource retrieves a configuration value and reports its provider source.
	GetWithSource(ctx context.Context, key string) (string, string, error)
}

// KeyAccessInfo records metadata about a configuration key access for debugging.
// It includes the key, provider, whether a default was used, and the call site.
type KeyAccessInfo struct {
	Default       bool
	Key           string
	Provider      string
	Caller        string
	ComponentType reflect.Type
	File          string
	Line          int
}

// providerInspector wraps a Provider and tracks all accessed keys and their sources for introspection.
type providerInspector struct {
	provider     Provider
	providerName string
	cache        map[string]string
	mu           sync.Mutex
	usedKeys     map[string][]KeyAccessInfo
}

// newProviderInspector creates a new inspector wrapper for introspection and caching.
func newProviderInspector(p Provider) *providerInspector {
	return &providerInspector{
		provider:     p,
		usedKeys:     make(map[string][]KeyAccessInfo),
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
	i.mu.Lock()
	defer i.mu.Unlock()

	if isDefaultConfigured {
		provider = ""
	}

	info := KeyAccessInfo{
		Key:           key,
		Provider:      provider,
		Default:       isDefaultConfigured,
		Caller:        caller,
		ComponentType: componentType,
		File:          reflectx.FormatFileName(file),
		Line:          line,
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

	return val, i.usedKeys[key][0].Provider, true
}

// getKeysAccessInfo returns all accessed keys sorted by key name, file, and line number.
func (i *providerInspector) getKeysAccessInfo() []KeyAccessInfo {
	i.mu.Lock()
	defer i.mu.Unlock()
	out := make([]KeyAccessInfo, 0)
	for _, arr := range i.usedKeys {
		out = append(out, arr...)
	}
	sort.Sort(byKey(out))
	return out
}

// byKey implements sorting for KeyAccessInfo by key name, then file, then line number.
type byKey []KeyAccessInfo

func (k byKey) Len() int { return len(k) }
func (k byKey) Less(i, j int) bool {
	if k[i].Key == k[j].Key {
		if k[i].File == k[j].File {
			return k[i].Line < k[j].Line
		}
		return k[i].File < k[j].File
	}
	return k[i].Key < k[j].Key
}
func (k byKey) Swap(i, j int) { k[i], k[j] = k[j], k[i] }
