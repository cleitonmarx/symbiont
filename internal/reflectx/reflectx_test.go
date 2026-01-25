package reflectx

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyValue(t *testing.T) {
	assert.Equal(t, 0, EmptyValue[int]())
	assert.Equal(t, "", EmptyValue[string]())
	assert.Nil(t, EmptyValue[*int]())
	assert.Nil(t, EmptyValue[[]string]())
	assert.Nil(t, EmptyValue[map[string]int]())
}

func TestGetTypeName(t *testing.T) {
	assert.Equal(t, "int", GetTypeName(reflect.TypeFor[int]()))
	assert.Equal(t, "string", GetTypeName(reflect.TypeFor[string]()))
	type myStruct struct{}
	assert.Contains(t, GetTypeName(reflect.TypeFor[myStruct]()), "myStruct")
	assert.Contains(t, GetTypeName(reflect.TypeFor[*myStruct]()), "myStruct")
}

func TestTypeNameOf(t *testing.T) {
	assert.Equal(t, "int", TypeNameOf(1))
	assert.Equal(t, "string", TypeNameOf("abc"))
	type foo struct{}
	assert.Equal(t, "reflectx.foo", TypeNameOf(foo{}))
}

func TestGetFunctionNameAndFileLine(t *testing.T) {
	fn := func() {}
	name, fileLine := GetFunctionNameAndFileLine(fn)
	assert.Contains(t, name, "TestGetFunctionNameAndFileLine")
	assert.Contains(t, fileLine, ".go:")
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
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"A", "B"}, fields)

	// Not a struct pointer
	err = IterateStructFields(testStruct{}, func(fieldValue reflect.Value, structField reflect.StructField, targetType reflect.Type) error {
		return nil
	})
	assert.Error(t, err)
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
	assert.NoError(t, err)
	assert.Equal(t, 42, s.A)

	// Unexported field
	type testStruct2 struct {
		a int
	}
	s2 := &testStruct2{a: 10}
	val2 := reflect.ValueOf(s2).Elem()
	field2 := val2.FieldByName("a")
	structField2, _ := val2.Type().FieldByName("a")
	err = SetFieldValue(field2, structField2, 99)
	assert.Error(t, err)
}

func TestGetCallerName(t *testing.T) {
	fn, file, line := GetCallerName(0)
	assert.NotEmpty(t, fn)
	assert.NotEmpty(t, file)
	assert.True(t, line > 0)
}

func TestFormatFunctionName(t *testing.T) {
	full := "github.com/cleitonmarx/symbiont/internal/reflectx.TestFormatFunctionName"
	assert.Equal(t, "reflectx.TestFormatFunctionName", FormatFunctionName(full))
}

func TestFormatFileName(t *testing.T) {
	path := "/foo/bar/baz.go"
	assert.Equal(t, "bar/baz.go", FormatFileName(path))
}

func TestIsPointerStruct(t *testing.T) {
	type foo struct{}
	assert.True(t, IsPointerStruct(reflect.ValueOf(&foo{})))
	assert.False(t, IsPointerStruct(reflect.ValueOf(foo{})))
	assert.False(t, IsPointerStruct(reflect.ValueOf(42)))
}

func Test_isTypeNullable(t *testing.T) {
	assert.True(t, isTypeNullable(reflect.TypeFor[*int]()))
	assert.False(t, isTypeNullable(reflect.TypeFor[int]()))
	assert.False(t, isTypeNullable(reflect.TypeFor[struct{}]()))
}
