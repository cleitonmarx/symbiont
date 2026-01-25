package depend

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Greeter interface {
	Greet() string
}

type EnglishGreeter struct{}

func (EnglishGreeter) Greet() string {
	return "Hello!"
}

type PortugueseGreeter struct{}

func (PortugueseGreeter) Greet() string {
	return "Olá!"
}

func TestResolveNamed(t *testing.T) {
	ClearContainer()

	englishGreeter := EnglishGreeter{}
	Register(englishGreeter)
	RegisterNamed[Greeter](englishGreeter, "englishGreeter")
	spanishGreeter := PortugueseGreeter{}
	Register(spanishGreeter)
	RegisterNamed[Greeter](spanishGreeter, "spanishGreeter")

	tests := map[string]struct {
		resolveFunc   func() (any, error)
		expectedValue any
		expectedErr   error
	}{
		"resolve_spanish_greeter": {
			resolveFunc: func() (any, error) {
				return Resolve[PortugueseGreeter]()
			},
			expectedValue: spanishGreeter,
		},
		"resolve_english_greeter": {
			resolveFunc: func() (any, error) {
				return Resolve[EnglishGreeter]()
			},
			expectedValue: englishGreeter,
		},
		"resolve_interface_dependency": {
			resolveFunc: func() (any, error) {
				return ResolveNamed[Greeter]("englishGreeter")
			},
			expectedValue: englishGreeter,
		},
		"resolve_second_interface_dependency": {
			resolveFunc: func() (any, error) {
				return ResolveNamed[Greeter]("spanishGreeter")
			},
			expectedValue: spanishGreeter,
		},
		"resolve_non_existent_named_interface": {
			resolveFunc: func() (any, error) {
				return ResolveNamed[Greeter]("nonexistent")
			},
			expectedValue: nil,
			expectedErr:   errors.New("depend: the dependency 'nonexistent' of type 'depend.Greeter' was not registered"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := tc.resolveFunc()
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedValue, result)
		})
	}
}

func TestResolve(t *testing.T) {
	ClearContainer()

	Register("Unnamed Greeting")
	Register(100)

	Register[Greeter](PortugueseGreeter{})

	tests := map[string]struct {
		resolveFunc   func() (any, error)
		expectedValue any
		expectedErr   error
	}{
		"resolve_string_dependency": {
			resolveFunc: func() (any, error) {
				return Resolve[string]()
			},
			expectedValue: "Unnamed Greeting",
		},
		"resolve_int_dependency": {
			resolveFunc: func() (any, error) {
				return Resolve[int]()
			},
			expectedValue: 100,
		},
		"resolve_interface_dependency": {
			resolveFunc: func() (any, error) {
				greeter, err := Resolve[Greeter]()
				if err != nil {
					return nil, err
				}
				return greeter.Greet(), nil
			},
			expectedValue: "Olá!",
		},
		"resolve_non_existent_dependency": {
			resolveFunc: func() (any, error) {
				return Resolve[float64]()
			},
			expectedValue: float64(0),
			expectedErr:   errors.New("depend: the dependency type 'float64' was not registered"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := tc.resolveFunc()
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedValue, result)
		})
	}
}

func TestRegisterNamedOnce_Greeter(t *testing.T) {
	ClearContainer()

	err := RegisterNamedOnce[Greeter](EnglishGreeter{}, "english")
	require.NoError(t, err)

	err = RegisterNamedOnce[Greeter](PortugueseGreeter{}, "english")
	require.EqualError(t, err, `depend: dependency already registered for type depend.Greeter and name "english"`)

	err = RegisterNamedOnce[Greeter](PortugueseGreeter{}, "portuguese")
	require.NoError(t, err)

	g, err := ResolveNamed[Greeter]("english")
	require.NoError(t, err)
	require.Equal(t, "Hello!", g.Greet())

	g, err = ResolveNamed[Greeter]("portuguese")
	require.NoError(t, err)
	require.Equal(t, "Olá!", g.Greet())
}

func TestRegisterOnce_Greeter(t *testing.T) {
	ClearContainer()

	err := RegisterOnce[Greeter](EnglishGreeter{})
	require.NoError(t, err)

	err = RegisterOnce[Greeter](PortugueseGreeter{})
	require.EqualError(t, err, "depend: dependency already registered for type depend.Greeter")

	g, err := Resolve[Greeter]()
	require.NoError(t, err)
	require.Equal(t, "Hello!", g.Greet())
}

func TestResolveStruct(t *testing.T) {
	ClearContainer()

	type (
		test struct {
			Answer            int     `resolve:""`
			EnglishGreeter    Greeter `resolve:"englishGreeter"`
			PortugueseGreeter Greeter `resolve:"portugueseGreeter"`
			NotTagged         string
		}
		testMissingType struct {
			Greeter Greeter `resolve:""`
		}
		testMissingNamed struct {
			MissingDep string `resolve:"missing"`
		}

		testNotSettable struct {
			notSettableField int `resolve:""`
		}
	)

	// Register dependencies
	Register(int(42))
	RegisterNamed[Greeter](EnglishGreeter{}, "englishGreeter")
	RegisterNamed[Greeter](PortugueseGreeter{}, "portugueseGreeter")

	tests := map[string]struct {
		target      any
		expected    any
		expectedErr error
	}{
		"resolve_all_dependencies": {
			target: &test{},
			expected: test{
				Answer:            42,
				EnglishGreeter:    EnglishGreeter{},
				PortugueseGreeter: PortugueseGreeter{},
			},
		},
		"error_resolving_missing_dependency": {
			target:      &testMissingType{},
			expected:    testMissingType{},
			expectedErr: errors.New("depend: the dependency '' of type 'depend.Greeter' was not registered"),
		},
		"error_resolving_missing_named_dependency": {
			target:      &testMissingNamed{},
			expected:    testMissingNamed{},
			expectedErr: errors.New("depend: the dependency type 'string' was not registered"),
		},
		"error_resolving_non_struct": {
			target:      &[]string{"not a struct"},
			expected:    []string{"not a struct"},
			expectedErr: errors.New("target must be a struct pointer, got '*[]string'"),
		},
		"error_resolving_field_not_settable": {
			target:      &testNotSettable{},
			expected:    testNotSettable{},
			expectedErr: errors.New("depend: field 'notSettableField' is not settable"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			switch target := tc.target.(type) {
			case *test:
				resolveStructAndAssert(t, target, tc.expected.(test), tc.expectedErr)
			case *testMissingType:
				resolveStructAndAssert(t, target, tc.expected.(testMissingType), tc.expectedErr)
			case *testMissingNamed:
				resolveStructAndAssert(t, target, tc.expected.(testMissingNamed), tc.expectedErr)
			case *testNotSettable:
				_ = target.notSettableField // Use the field to avoid unused warning
				resolveStructAndAssert(t, target, tc.expected.(testNotSettable), tc.expectedErr)
			case *[]string:
				resolveStructAndAssert(t, target, tc.expected.([]string), tc.expectedErr)
			default:
				t.Fatalf("unsupported target type")
			}
		})
	}
}

func resolveStructAndAssert[T any](t *testing.T, target *T, expected T, expectedErr error) {
	err := ResolveStruct(target)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, expected, *target)
}
