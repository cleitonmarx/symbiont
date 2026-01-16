// Package reflectx provides reflection utilities for struct field injection and type introspection.
// It supports iterating struct fields, getting type names, and formatting caller information for diagnostics.
package reflectx

import (
	"fmt"
	"path"
	"reflect"
	"runtime"
	"strings"
)

// EmptyValue returns the zero value for type T.
// For nullable types (pointers, slices, maps, channels, functions, interfaces), returns nil.
func EmptyValue[T any]() T {
	var emptyType T
	typeOfT := reflect.TypeFor[T]()
	if !isTypeNullable(typeOfT) {
		emptyType = *new(T)
	}
	return emptyType
}

// GetTypeName returns a human-readable type name for a reflect.Type.
// Format: "package.TypeName" or just "TypeName" for built-in types.
func GetTypeName(t reflect.Type) string {
	if t.PkgPath() == "" {
		if t.Name() == "" {
			return t.String()
		}
		return t.Name()
	}
	return fmt.Sprintf("%s.%s", path.Base(t.PkgPath()), t.Name())
}

// TypeNameOf returns the type name of a value using fmt formatting.
func TypeNameOf(t any) string {
	return fmt.Sprintf("%T", t)
}

// GetFunctionNameAndFileLine returns the function name and source location (file:line) of a function value.
func GetFunctionNameAndFileLine(fn any) (string, string) {
	funcRef := runtime.FuncForPC(reflect.ValueOf(fn).Pointer())
	name := funcRef.Name()
	file, line := funcRef.FileLine(funcRef.Entry())
	// Trim to just package.Func
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		name = name[idx+1:]
	}
	fileLine := fmt.Sprintf("%s:%d", file, line)
	return name, fileLine
}

// StructFieldIteratorFunc is a callback function called for each struct field during iteration.
type StructFieldIteratorFunc func(fieldValue reflect.Value, structField reflect.StructField, targetType reflect.Type) error

// IterateStructFields calls the provided functions for each field in a struct pointer.
// Functions are called in order for each field. Returns error if target is not a struct pointer or if any function fails.
func IterateStructFields(target any, fns ...StructFieldIteratorFunc) error {
	v := reflect.ValueOf(target)
	if !IsPointerStruct(v) {
		return fmt.Errorf("target must be a struct pointer, got '%s'", GetTypeName(v.Type()))
	}
	vtype := v.Type()
	v = v.Elem()
	t := v.Type()
	for i := range v.NumField() {
		for _, fn := range fns {
			if err := fn(v.Field(i), t.Field(i), vtype); err != nil {
				return err
			}
		}
	}
	return nil
}

// SetFieldValue sets a struct field to the provided value.
// Returns error if the field is not settable (e.g., unexported field).
func SetFieldValue(field reflect.Value, structField reflect.StructField, value any) error {
	if field.CanSet() {
		field.Set(reflect.ValueOf(value))
		return nil
	}
	return fmt.Errorf("field '%s' is not settable", structField.Name)
}

// GetCallerName returns the caller's function name and source location (file:line) at the specified stack depth.
func GetCallerName(skip int) (string, string, int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", "unknown", 0
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown", "unknown", 0
	}

	return fn.Name(), file, line
}

// FormatFunctionName formats a qualified function name to just the package.FuncName portion.
func FormatFunctionName(name string) string {
	// Split the caller string to extract the function name
	parts := strings.Split(name, "/")

	return parts[len(parts)-1]
}

// FormatFileName formats a file path to show the directory and filename (e.g., "dir/file.go").
func FormatFileName(file string) string {
	dir, fileName := path.Split(file)
	lastDir := path.Base(path.Clean(dir))
	return fmt.Sprintf("%s/%s", lastDir, fileName)
}

// IsPointerStruct checks if a reflect.Value is a pointer to a struct.
func IsPointerStruct(v reflect.Value) bool {
	return v.Kind() == reflect.Pointer && v.Elem().Kind() == reflect.Struct
}

// isTypeNullable checks if a type can be nil (pointers, slices, maps, channels, functions, interfaces).
func isTypeNullable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
		return true
	default:
		return false
	}
}
