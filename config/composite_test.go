package config

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testProvider1 struct {
	*mockProvider
}
type testProvider2 struct {
	*mockProvider
}

func TestCompositeProvider_Get(t *testing.T) {
	tests := map[string]struct {
		setMocksExpectations func(p1 *mockProvider, p2 *mockProvider)
		expectedValue        string
		expectedError        error
	}{
		"exits_in_first_provider": {
			setMocksExpectations: func(p1 *mockProvider, p2 *mockProvider) {
				p1.On("Get", mock.Anything, "key").
					Return("value", nil)
			},
			expectedValue: "value",
			expectedError: nil,
		},
		"exits_in_second_provider": {
			setMocksExpectations: func(p1 *mockProvider, p2 *mockProvider) {
				p1.On("Get", mock.Anything, "key").
					Return("", errors.New("config 'key' does not exist"))
				p2.On("Get", mock.Anything, "key").
					Return("value", nil)
			},
			expectedValue: "value",
			expectedError: nil,
		},
		"does_not_exist": {
			expectedValue: "",
			setMocksExpectations: func(p1 *mockProvider, p2 *mockProvider) {
				p1.On("Get", mock.Anything, "key").
					Return("", errors.New("config 'key' does not exist"))
				p2.On("Get", mock.Anything, "key").
					Return("", errors.New("config 'key' does not exist"))
			},
			expectedError: fmt.Errorf("config.testProvider1: config 'key' does not exist\nconfig.testProvider2: config 'key' does not exist"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p1 := testProvider1{
				mockProvider: &mockProvider{},
			}
			p2 := testProvider2{
				mockProvider: &mockProvider{},
			}
			if tt.setMocksExpectations != nil {
				tt.setMocksExpectations(p1.mockProvider, p2.mockProvider)
			}
			p := NewCompositeProvider(
				p1,
				p2,
			)
			got, err := p.Get(context.Background(), "key")
			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedValue, got)
			mock.AssertExpectationsForObjects(t, p1.mockProvider, p2.mockProvider)
		})
	}
}

func TestCompositeProvider_GetAndReportProvider(t *testing.T) {
	tests := map[string]struct {
		setMocksExpectations func(p1 *mockProvider, p2 *mockProvider)
		expectedValue        string
		expectedProvider     string
		expectedError        error
	}{
		"found_in_first_provider": {
			setMocksExpectations: func(p1 *mockProvider, p2 *mockProvider) {
				p1.On("Get", mock.Anything, "key").
					Return("value1", nil)
			},
			expectedValue:    "value1",
			expectedProvider: "config.testProvider1",
			expectedError:    nil,
		},
		"found_in_second_provider": {
			setMocksExpectations: func(p1 *mockProvider, p2 *mockProvider) {
				p1.On("Get", mock.Anything, "key").
					Return("", errors.New("not found"))
				p2.On("Get", mock.Anything, "key").
					Return("value2", nil)
			},
			expectedValue:    "value2",
			expectedProvider: "config.testProvider2",
			expectedError:    nil,
		},
		"not_found_in_any_provider": {
			setMocksExpectations: func(p1 *mockProvider, p2 *mockProvider) {
				p1.On("Get", mock.Anything, "key").
					Return("", errors.New("not found"))
				p2.On("Get", mock.Anything, "key").
					Return("", errors.New("not found"))
			},
			expectedValue:    "",
			expectedProvider: "",
			expectedError:    fmt.Errorf("config.testProvider1: not found\nconfig.testProvider2: not found"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p1 := testProvider1{
				mockProvider: &mockProvider{},
			}
			p2 := testProvider2{
				mockProvider: &mockProvider{},
			}
			if tt.setMocksExpectations != nil {
				tt.setMocksExpectations(p1.mockProvider, p2.mockProvider)
			}
			p := NewCompositeProvider(
				p1,
				p2,
			)
			gotValue, gotProvider, err := p.GetWithSource(context.Background(), "key")
			assert.Equal(t, tt.expectedValue, gotValue)
			assert.Equal(t, tt.expectedProvider, gotProvider)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
			mock.AssertExpectationsForObjects(t, p1.mockProvider, p2.mockProvider)
		})
	}
}
