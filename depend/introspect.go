package depend

import (
	"reflect"
	"strings"
	"sync"

	"github.com/cleitonmarx/symbiont/internal/reflectx"
	"github.com/cleitonmarx/symbiont/introspection"
)

// Event tracking provides observability into dependency registration and resolution.
// All registration and resolution events are logged for debugging and diagnostics.

var (
	eventMu sync.Mutex
	events  []introspection.DepEvent
	order   int
)

// logEvent records a dependency event with caller information for observability.
func logEvent(action introspection.DepEventKind, depTypeName, depName, implName string, componentType reflect.Type, level int) {
	callerFunc, file, line := reflectx.GetCallerName(level + 1)
	caller := reflectx.FormatFunctionName(callerFunc)
	if strings.Contains(caller, "symbiont.(*App).") {
		caller = ""
	}

	componentName := ""
	if componentType != nil {
		componentName = reflectx.GetTypeName(componentType)
	}

	eventMu.Lock()
	defer eventMu.Unlock()
	order++

	events = append(events, introspection.DepEvent{
		Kind: action,
		Type: depTypeName,
		Name: depName,
		Impl: implName,
		Caller: introspection.Caller{
			Func: caller,
			File: reflectx.FormatFileName(file),
			Line: line,
		},
		Component: componentName,
		Order:     order,
	})
}

// GetEvents returns a copy of all recorded dependency events.
// Useful for testing and verifying dependency registration/resolution behavior.
func GetEvents() []introspection.DepEvent {
	eventMu.Lock()
	defer eventMu.Unlock()
	cpy := make([]introspection.DepEvent, len(events))
	copy(cpy, events)
	return cpy
}
