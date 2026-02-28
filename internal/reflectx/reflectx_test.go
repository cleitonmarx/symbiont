package reflectx

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestEmptyValue(t *testing.T) {
	if EmptyValue[int]() != 0 {
		t.Fatalf("expected zero int value")
	}
	if EmptyValue[string]() != "" {
		t.Fatalf("expected zero string value")
	}
	if EmptyValue[*int]() != nil {
		t.Fatalf("expected nil pointer value")
	}
	if EmptyValue[[]string]() != nil {
		t.Fatalf("expected nil slice value")
	}
	if EmptyValue[map[string]int]() != nil {
		t.Fatalf("expected nil map value")
	}
}

func TestGetTypeName(t *testing.T) {
	if GetTypeName(reflect.TypeFor[int]()) != "int" {
		t.Fatalf("expected type name %q, got %q", "int", GetTypeName(reflect.TypeFor[int]()))
	}
	if GetTypeName(reflect.TypeFor[string]()) != "string" {
		t.Fatalf("expected type name %q, got %q", "string", GetTypeName(reflect.TypeFor[string]()))
	}
	type myStruct struct{}
	if !strings.Contains(GetTypeName(reflect.TypeFor[myStruct]()), "myStruct") {
		t.Fatalf("expected type name to contain %q", "myStruct")
	}
	if !strings.Contains(GetTypeName(reflect.TypeFor[*myStruct]()), "myStruct") {
		t.Fatalf("expected pointer type name to contain %q", "myStruct")
	}
}

func TestTypeNameOf(t *testing.T) {
	if TypeNameOf(1) != "int" {
		t.Fatalf("expected type name %q, got %q", "int", TypeNameOf(1))
	}
	if TypeNameOf("abc") != "string" {
		t.Fatalf("expected type name %q, got %q", "string", TypeNameOf("abc"))
	}
	type foo struct{}
	if TypeNameOf(foo{}) != "reflectx.foo" {
		t.Fatalf("expected type name %q, got %q", "reflectx.foo", TypeNameOf(foo{}))
	}
}

func TestGetFunctionNameAndFileLine(t *testing.T) {
	fn := func() {}
	name, fileLine := GetFunctionNameAndFileLine(fn)
	if !strings.Contains(name, "TestGetFunctionNameAndFileLine") {
		t.Fatalf("expected function name to contain test name, got %q", name)
	}
	if !strings.Contains(fileLine, ".go:") {
		t.Fatalf("expected file line to contain .go:, got %q", fileLine)
	}
}

func TestIterateStructFields(t *testing.T) {
	type testStruct struct {
		A int
		B string
	}
	s := &testStruct{A: 1, B: "x"}
	var fields []string
	err := IterateStructFields(s, func(fieldValue reflect.Value, structField reflect.StructField, targetType reflect.Type) error {
		fields = append(fields, structField.Name)
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	slices.Sort(fields)
	if !reflect.DeepEqual([]string{"A", "B"}, fields) {
		t.Fatalf("expected fields %v, got %v", []string{"A", "B"}, fields)
	}

	// Not a struct pointer
	err = IterateStructFields(testStruct{}, func(fieldValue reflect.Value, structField reflect.StructField, targetType reflect.Type) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSetFieldValue(t *testing.T) {
	type testStruct struct {
		A int
		B string
	}
	s := &testStruct{}
	val := reflect.ValueOf(s).Elem()
	field := val.FieldByName("A")
	structField, _ := val.Type().FieldByName("A")
	err := SetFieldValue(field, structField, 42)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.A != 42 {
		t.Fatalf("expected field value %d, got %d", 42, s.A)
	}

	// Unexported field
	type testStruct2 struct {
		a int
	}
	s2 := &testStruct2{a: 10}
	val2 := reflect.ValueOf(s2).Elem()
	field2 := val2.FieldByName("a")
	structField2, _ := val2.Type().FieldByName("a")
	err = SetFieldValue(field2, structField2, 99)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetCallerName(t *testing.T) {
	fn, file, line := GetCallerName(0)
	if fn == "" {
		t.Fatal("expected function name, got empty string")
	}
	if file == "" {
		t.Fatal("expected file path, got empty string")
	}
	if line <= 0 {
		t.Fatalf("expected positive line number, got %d", line)
	}
}

func TestFormatFunctionName(t *testing.T) {
	full := "github.com/cleitonmarx/symbiont/internal/reflectx.TestFormatFunctionName"
	if FormatFunctionName(full) != "reflectx.TestFormatFunctionName" {
		t.Fatalf("expected formatted name %q, got %q", "reflectx.TestFormatFunctionName", FormatFunctionName(full))
	}
}

func TestFormatFileName(t *testing.T) {
	path := "/foo/bar/baz.go"
	if FormatFileName(path) != "bar/baz.go" {
		t.Fatalf("expected formatted path %q, got %q", "bar/baz.go", FormatFileName(path))
	}
}

func TestIsPointerStruct(t *testing.T) {
	type foo struct{}
	if !IsPointerStruct(reflect.ValueOf(&foo{})) {
		t.Fatal("expected pointer struct to be true")
	}
	if IsPointerStruct(reflect.ValueOf(foo{})) {
		t.Fatal("expected non-pointer struct to be false")
	}
	if IsPointerStruct(reflect.ValueOf(42)) {
		t.Fatal("expected non-struct to be false")
	}
}

func Test_isTypeNullable(t *testing.T) {
	if !isTypeNullable(reflect.TypeFor[*int]()) {
		t.Fatal("expected pointer type to be nullable")
	}
	if isTypeNullable(reflect.TypeFor[int]()) {
		t.Fatal("expected int type to be non-nullable")
	}
	if isTypeNullable(reflect.TypeFor[struct{}]()) {
		t.Fatal("expected struct type to be non-nullable")
	}
}
