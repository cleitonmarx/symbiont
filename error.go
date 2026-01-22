package symbiont

import (
	"fmt"
	"reflect"

	"github.com/cleitonmarx/symbiont/internal/reflectx"
)

// Error represents a symbiont error with context about the component that failed.
// It includes the original error, component name or function name, and source location for debugging.
type Error struct {
	Err           error
	ComponentName string
	FileLine      string
}

// NewError wraps an error with component context (type name or function name and location).
// For functions, includes source file and line number; for types, includes package.TypeName.
func NewError(err error, component any) Error {
	componentType := reflect.TypeOf(component)
	if componentType.Kind() == reflect.Func {
		functionName, fileLine := reflectx.GetFunctionNameAndFileLine(component)
		return Error{
			Err:           err,
			ComponentName: functionName,
			FileLine:      fileLine,
		}
	}

	return Error{
		Err:           err,
		ComponentName: reflectx.GetTypeName(componentType),
	}
}

// Error implements the error interface, returning a formatted error message with component context.
func (e Error) Error() string {
	if e.FileLine == "" {
		return fmt.Sprintf("error: %v, component: %s", e.Err, e.ComponentName)
	}
	return fmt.Sprintf("error: %v, function: %s, location: %s", e.Err, e.ComponentName, e.FileLine)
}
