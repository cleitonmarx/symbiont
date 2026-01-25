package config

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockProvider is a mock type for the Provider type
type mockProvider struct {
	mock.Mock
}

func (m *mockProvider) Get(ctx context.Context, name string) (string, error) {
	args := m.Called(ctx, name)
	return args.String(0), args.Error(1)
}

func TestGet(t *testing.T) {
	RegisterParser(func(name string) ([]string, error) {
		return strings.Split(name, ","), nil
	})

	tests := map[string]struct {
		key             string
		setExpectations func(p *mockProvider)
		expected        any
		expectErr       error
	}{
		"string": {
			key: "stringKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "stringKey").Return("value", nil)
			},
			expected: "value",
		},
		"bool": {
			key: "boolKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "boolKey").Return("true", nil)
			},
			expected: true,
		},
		"int": {
			key: "intKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "intKey").Return("42", nil)
			},
			expected: 42,
		},
		"int64": {
			key: "int64Key",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "int64Key").Return("64", nil)
			},
			expected: int64(64),
		},
		"float64": {
			key: "floatKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "floatKey").Return("3.14", nil)
			},
			expected: 3.14,
		},
		"duration": {
			key: "durationKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "durationKey").Return("1h", nil)
			},
			expected: time.Hour,
		},
		"custom_string_slice_parser": {
			key: "sliceKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "sliceKey").Return("1,2,3", nil)
			},
			expected: []string{"1", "2", "3"},
		},
		"error_parsing_string_to_int": {
			key: "errorIntKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "errorIntKey").Return("string_value", nil)
			},
			expected:  0,
			expectErr: errors.New("config: strconv.Atoi: parsing \"string_value\": invalid syntax"),
		},
		"error_when_no_parser_for_type": {
			key:       "nonExistentKey",
			expected:  uint(0),
			expectErr: errors.New("config: parser for type 'uint' does not exist"),
		},
		"error_when_config_not_found": {
			key: "nonExistentKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "nonExistentKey").Return("", errors.New("key 'nonExistentKey' does not exist"))
			},
			expected:  0,
			expectErr: errors.New("config: key 'nonExistentKey' does not exist"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockProvider := &mockProvider{}
			if tt.setExpectations != nil {
				tt.setExpectations(mockProvider)
			}
			SetGlobalProvider(mockProvider)
			ctx := context.Background()

			var (
				result any
				err    error
			)
			switch tt.expected.(type) {
			case string:
				result, err = Get[string](ctx, tt.key)
			case bool:
				result, err = Get[bool](ctx, tt.key)
			case int:
				result, err = Get[int](ctx, tt.key)
			case int64:
				result, err = Get[int64](ctx, tt.key)
			case float64:
				result, err = Get[float64](ctx, tt.key)
			case time.Duration:
				result, err = Get[time.Duration](ctx, tt.key)
			case []string:
				result, err = Get[[]string](ctx, tt.key)
			case uint:
				result, err = Get[uint](ctx, tt.key)
			}

			assert.Equal(t, tt.expectErr, err)
			assert.Equal(t, tt.expected, result)
			mock.AssertExpectationsForObjects(t, mockProvider)
		})
	}
}

func TestGetWithDefault(t *testing.T) {
	RegisterParser(func(name string) ([]string, error) {
		return strings.Split(name, ","), nil
	})

	tests := map[string]struct {
		key             string
		setExpectations func(p *mockProvider)
		defaultValue    any
		expected        any
	}{
		"bool_with_default": {
			key: "boolKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "boolKey").Return("true", nil)
			},
			defaultValue: false,
			expected:     true,
		},
		"int_with_default": {
			key: "intKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "intKey").Return("3", nil)
			},
			defaultValue: 0,
			expected:     3,
		},
		"int64_with_default": {
			key: "int64Key",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "int64Key").Return("4", nil)
			},
			defaultValue: int64(0),
			expected:     int64(4),
		},
		"float64_with_default": {
			key: "floatKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "floatKey").Return("5.25", nil)
			},
			defaultValue: 0.0,
			expected:     5.25,
		},
		"duration_with_default": {
			key: "durationKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "durationKey").Return("6h", nil)
			},
			defaultValue: time.Minute,
			expected:     6 * time.Hour,
		},
		"custom_string_slice_parser_with_default": {
			key: "sliceKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "sliceKey").Return("a,b,c", nil)
			},
			defaultValue: []string{},
			expected:     []string{"a", "b", "c"},
		},
		"default_value_when_config_not_found": {
			key: "nonFoundKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "nonFoundKey").Return("", errors.New("config 'nonFoundKey' does not exist"))
			},
			defaultValue: 100,
			expected:     100,
		},
		"default_value_when_error_on_parsing": {
			key: "errorIntKey",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "errorIntKey").Return("string_value", nil)
			},
			defaultValue: 0,
			expected:     0,
		},
		"default_value_when_no_parser_for_type": {
			key:          "cantParseKey",
			defaultValue: uint(0),
			expected:     uint(0),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockProvider := &mockProvider{}
			if tt.setExpectations != nil {
				tt.setExpectations(mockProvider)
			}
			SetGlobalProvider(mockProvider)
			ctx := context.Background()

			var result any
			switch tt.defaultValue.(type) {
			case bool:
				result = GetWithDefault(ctx, tt.key, tt.defaultValue.(bool))
			case int:
				result = GetWithDefault(ctx, tt.key, tt.defaultValue.(int))
			case int64:
				result = GetWithDefault(ctx, tt.key, tt.defaultValue.(int64))
			case float64:
				result = GetWithDefault(ctx, tt.key, tt.defaultValue.(float64))
			case time.Duration:
				result = GetWithDefault(ctx, tt.key, tt.defaultValue.(time.Duration))
			case []string:
				result = GetWithDefault(ctx, tt.key, tt.defaultValue.([]string))
			case uint:
				result = GetWithDefault(ctx, tt.key, tt.defaultValue.(uint))
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadStruct(t *testing.T) {
	RegisterParser(func(name string) ([]string, error) {
		return strings.Split(name, ","), nil
	})

	// Define test structs inside the test function
	type (
		validConfig struct {
			BoolValue      bool          `config:"boolKey"`
			IntValue       int           `config:"intKey"`
			Int64Value     int64         `config:"int64Key"`
			FloatValue     float64       `config:"floatKey"`
			DurationValue  time.Duration `config:"durationKey"`
			SliceValue     []string      `config:"sliceKey"`
			DefaultValue   bool          `config:"defaultKey" default:"true"`
			NotLoadedValue string
		}
		configNotFound struct {
			IntValue int `config:"missingKey"`
		}
		parserNotFound struct {
			UintValue uint `config:"intKey"`
		}
		invalidConfigParameterType struct {
			IntValue int `config:"boolKey"`
		}
		fieldNotSettable struct {
			IntValue   int     `config:"intKey"`
			floatValue float64 `config:"floatKey"`
		}
	)

	tests := map[string]struct {
		structToLoad    any
		setExpectations func(p *mockProvider)
		expected        any
		expectedErr     error
	}{
		"valid_config": {
			structToLoad: &validConfig{},
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "boolKey").Return("true", nil)
				p.On("Get", mock.Anything, "intKey").Return("42", nil)
				p.On("Get", mock.Anything, "int64Key").Return("64", nil)
				p.On("Get", mock.Anything, "floatKey").Return("3.14", nil)
				p.On("Get", mock.Anything, "durationKey").Return("1h", nil)
				p.On("Get", mock.Anything, "sliceKey").Return("1,2,3", nil)
				p.On("Get", mock.Anything, "defaultKey").Return("", errors.New("key not found"))
			},
			expected: &validConfig{
				BoolValue:      true,
				IntValue:       42,
				Int64Value:     64,
				FloatValue:     3.14,
				DurationValue:  time.Hour,
				SliceValue:     []string{"1", "2", "3"},
				NotLoadedValue: "",
				DefaultValue:   true,
			},
		},
		"config_not_found": {
			structToLoad: &configNotFound{},
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "missingKey").Return("", fmt.Errorf("'missingKey' does not exist"))
			},
			expected:    &configNotFound{},
			expectedErr: fmt.Errorf("config: error getting value for field 'IntValue': 'missingKey' does not exist"),
		},
		"parser_not_found": {
			structToLoad: &parserNotFound{},
			expected:     &parserNotFound{},
			expectedErr:  errors.New("config: parser for type 'uint' does not exist"),
		},
		"invalid_config_parameter_type": {
			structToLoad: &invalidConfigParameterType{},
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "boolKey").Return("true", nil)
			},
			expected:    &invalidConfigParameterType{},
			expectedErr: errors.New("config: error parsing value for field 'IntValue': strconv.Atoi: parsing \"true\": invalid syntax"),
		},
		"field_not_settable": {
			structToLoad: &fieldNotSettable{},
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "intKey").Return("42", nil)
				p.On("Get", mock.Anything, "floatKey").Return("3.14", nil)
			},
			expected: &fieldNotSettable{
				IntValue:   42,
				floatValue: 0,
			},
			expectedErr: fmt.Errorf("config: field 'floatValue' is not settable"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockProvider := &mockProvider{}
			if tt.setExpectations != nil {
				tt.setExpectations(mockProvider)
			}
			SetGlobalProvider(mockProvider)
			ctx := context.Background()
			switch structToLoad := tt.structToLoad.(type) {
			case *validConfig:
				loadStructAndAssert(t, ctx, structToLoad, tt.expected.(*validConfig), nil)
			case *configNotFound:
				loadStructAndAssert(t, ctx, structToLoad, tt.expected.(*configNotFound), tt.expectedErr)
			case *parserNotFound:
				loadStructAndAssert(t, ctx, structToLoad, tt.expected.(*parserNotFound), tt.expectedErr)
			case *invalidConfigParameterType:
				loadStructAndAssert(t, ctx, structToLoad, tt.expected.(*invalidConfigParameterType), tt.expectedErr)
			case *fieldNotSettable:
				loadStructAndAssert(t, ctx, structToLoad, tt.expected.(*fieldNotSettable), tt.expectedErr)
			default:
				t.Fatalf("unsupported target type")
			}
			mock.AssertExpectationsForObjects(t, mockProvider)
		})
	}
}

func loadStructAndAssert[T any](t *testing.T, ctx context.Context, target *T, expected *T, expectedErr error) {
	err := LoadStruct(ctx, target)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, expected, target)
}

func TestIntrospectConfigAccesses(t *testing.T) {
	RegisterParser(func(name string) ([]string, error) {
		return strings.Split(name, ","), nil
	})

	tests := map[string]struct {
		key             string
		setExpectations func(p *mockProvider)
		getFunc         func(ctx context.Context, key string) any
		expectDefault   bool
	}{
		"normal_access": {
			key: "foo",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "foo").Return("bar", nil)
			},
			getFunc: func(ctx context.Context, key string) any {
				val, _ := Get[string](ctx, key)
				return val
			},
			expectDefault: false,
		},
		"default_value": {
			key: "missing",
			setExpectations: func(p *mockProvider) {
				p.On("Get", mock.Anything, "missing").Return("", errors.New("not found"))
			},
			getFunc: func(ctx context.Context, key string) any {
				return GetWithDefault(ctx, key, "default")
			},
			expectDefault: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockProvider := &mockProvider{}
			if tt.setExpectations != nil {
				tt.setExpectations(mockProvider)
			}
			SetGlobalProvider(mockProvider)
			ctx := context.Background()

			_ = tt.getFunc(ctx, tt.key)

			keys := IntrospectConfigAccesses()
			var found *introspection.ConfigAccess
			for i := range keys {
				if keys[i].Key == tt.key {
					found = &keys[i]
					break
				}
			}
			assert.NotNil(t, found, "Expected key %s to be introspected", tt.key)
			if found != nil {
				assert.Equal(t, tt.expectDefault, found.UsedDefault)
			}
			mock.AssertExpectationsForObjects(t, mockProvider)
		})
	}
}
