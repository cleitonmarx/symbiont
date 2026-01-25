// Package depend provides a type-safe, thread-safe dependency injection container.
// Dependencies are registered by type and resolved by value or struct field injection.
// Both unnamed and named dependencies are supported.
package depend

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/cleitonmarx/symbiont/internal/reflectx"
	"github.com/cleitonmarx/symbiont/introspection"
)

const tagName = "resolve"

// container is a global map that stores registered dependencies, organized by type and name
var (
	containerMu sync.RWMutex
	container   = make(map[reflect.Type]map[string]any)
)

// RegisterNamed registers a dependency with an optional name.
// Multiple dependencies of the same type can be registered with different names.
func RegisterNamed[T any](dependency T, name string) {
	typeOfT := reflect.TypeFor[T]()
	containerMu.Lock()
	defer containerMu.Unlock()
	if _, exist := container[typeOfT]; !exist {
		container[typeOfT] = make(map[string]any)
	}
	container[typeOfT][name] = dependency

	if name != "" {
		logEvent(
			introspection.DepRegistered,
			reflectx.GetTypeName(typeOfT),
			name,
			reflectx.TypeNameOf(dependency),
			nil,
			2,
		)
	}
}

// Register registers an unnamed dependency by type.
// Only one unnamed dependency per type can be registered; use RegisterNamed for multiple instances.
func Register[T any](dependency T) {
	RegisterNamed(dependency, "")
	logEvent(
		introspection.DepRegistered,
		reflectx.GetTypeName(reflect.TypeFor[T]()),
		"",
		reflectx.TypeNameOf(dependency),
		nil,
		2,
	)
}

// RegisterNamedOnce registers a named dependency, returning an error if already registered.
func RegisterNamedOnce[T any](dependency T, name string) error {
	typeOfT := reflect.TypeFor[T]()
	containerMu.Lock()
	defer containerMu.Unlock()
	if _, exist := container[typeOfT]; !exist {
		container[typeOfT] = make(map[string]any)
	}
	if _, exists := container[typeOfT][name]; exists {
		if name == "" {
			return fmt.Errorf("depend: dependency already registered for type %s", reflectx.GetTypeName(typeOfT))
		}
		return fmt.Errorf("depend: dependency already registered for type %s and name %q", reflectx.GetTypeName(typeOfT), name)
	}
	container[typeOfT][name] = dependency
	if name != "" {
		logEvent(
			introspection.DepRegistered,
			reflectx.GetTypeName(typeOfT),
			name,
			reflectx.TypeNameOf(dependency),
			nil,
			2,
		)
	}
	return nil
}

// RegisterOnce registers an unnamed dependency, returning an error if already registered.
func RegisterOnce[T any](dependency T) error {
	logEvent(
		introspection.DepRegistered,
		reflectx.GetTypeName(reflect.TypeFor[T]()),
		"",
		reflectx.TypeNameOf(dependency),
		nil,
		2,
	)
	return RegisterNamedOnce(dependency, "")
}

// ResolveNamed retrieves a registered dependency by type and name.
func ResolveNamed[T any](name string) (T, error) {
	emptyType := reflectx.EmptyValue[T]()
	typeOfT := reflect.TypeFor[T]()
	containerMu.RLock()
	defer containerMu.RUnlock()

	if dependenciesByName, exist := container[typeOfT]; exist {
		if dependency, exist := dependenciesByName[name]; exist {
			if name != "" {
				logEvent(
					introspection.DepResolved,
					reflectx.GetTypeName(typeOfT),
					name,
					reflectx.TypeNameOf(dependency),
					nil,
					2,
				)
			}
			return dependency.(T), nil
		}
		return emptyType, fmt.Errorf("depend: the dependency '%s' of type '%s' was not registered", name, reflectx.GetTypeName(typeOfT))
	}
	return emptyType, fmt.Errorf("depend: the dependency type '%s' was not registered", reflectx.GetTypeName(typeOfT))
}

// Resolve retrieves the unnamed registered dependency of the specified type.
func Resolve[T any]() (T, error) {
	dep, err := ResolveNamed[T]("")
	logEvent(
		introspection.DepResolved,
		reflectx.GetTypeName(reflect.TypeFor[T]()),
		"",
		reflectx.TypeNameOf(dep),
		nil,
		2,
	)
	return dep, err
}

// ResolveStruct injects dependencies into all struct fields tagged with resolve:"name".
func ResolveStruct[T any](target *T) error {
	return reflectx.IterateStructFields(target, ResolveStructFieldValue)
}

// ResolveStructFieldValue injects a dependency into a single struct field based on its resolve tag.
// Used internally during struct field injection; resolves by field type and tag value.
func ResolveStructFieldValue(fieldValue reflect.Value, structField reflect.StructField, targetType reflect.Type) error {
	dependencyName, ok := structField.Tag.Lookup(tagName)
	if !ok {
		return nil
	}
	containerMu.RLock()
	defer containerMu.RUnlock()

	dependenciesByName, typeExist := container[fieldValue.Type()]
	if !typeExist {
		return fmt.Errorf("depend: the dependency type '%s' was not registered", reflectx.GetTypeName(fieldValue.Type()))
	}
	dependency, nameExist := dependenciesByName[dependencyName]
	if !nameExist {
		return fmt.Errorf("depend: the dependency '%s' of type '%s' was not registered", dependencyName, reflectx.GetTypeName(fieldValue.Type()))
	}
	if err := reflectx.SetFieldValue(fieldValue, structField, dependency); err != nil {
		return fmt.Errorf("depend: %s", err)
	}

	logEvent(
		introspection.DepResolved,
		reflectx.GetTypeName(fieldValue.Type()),
		dependencyName,
		reflectx.TypeNameOf(dependency),
		targetType,
		5,
	)
	return nil
}

// ClearContainer removes all registered dependencies and clears the event log.
// Typically used in tests to isolate dependency registrations between test cases.
func ClearContainer() {
	containerMu.Lock()
	eventMu.Lock()
	defer containerMu.Unlock()
	defer eventMu.Unlock()

	container = make(map[reflect.Type]map[string]any)
	events = make([]introspection.DepEvent, 0)
}
