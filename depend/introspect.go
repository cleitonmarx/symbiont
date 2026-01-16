package depend

import (
	"reflect"
	"strings"
	"sync"

	"github.com/cleitonmarx/symbiont/internal/reflectx"
)

// Event tracking provides observability into dependency registration and resolution.
// All registration and resolution events are logged for debugging and diagnostics.

var (
	eventMu sync.Mutex
	events  []Event
)

// EventAction identifies the type of dependency event.
type EventAction string

const (
	// ActionRegister indicates a dependency registration event
	ActionRegister EventAction = "register"
	// ActionResolve indicates a dependency resolution event
	ActionResolve EventAction = "resolve"
)

// Event represents a single dependency registration or resolution event.
// It includes type information, names, implementation details, and source location for debugging.
type Event struct {
	Action         EventAction
	DepTypeName    string
	DepName        string
	Implementation string
	Caller         string
	ComponentType  reflect.Type
	File           string
	Line           int
}

// logEvent records a dependency event with caller information for observability.
func logEvent(action EventAction, depTypeName, depName, implName string, componentType reflect.Type, level int) {
	callerFunc, file, line := reflectx.GetCallerName(level + 1)
	caller := reflectx.FormatFunctionName(callerFunc)
	if strings.Contains(caller, "symbiont.(*App).") {
		caller = ""
	}

	eventMu.Lock()
	defer eventMu.Unlock()

	events = append(events, Event{
		Action:         action,
		DepTypeName:    depTypeName,
		DepName:        depName,
		Implementation: implName,
		Caller:         caller,
		ComponentType:  componentType,
		File:           reflectx.FormatFileName(file),
		Line:           line,
	})
}

// GetEvents returns a copy of all recorded dependency events.
// Useful for testing and verifying dependency registration/resolution behavior.
func GetEvents() []Event {
	eventMu.Lock()
	defer eventMu.Unlock()
	cpy := make([]Event, len(events))
	copy(cpy, events)
	return cpy
}
