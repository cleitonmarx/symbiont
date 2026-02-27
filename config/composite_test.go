package config

import (
	"context"
	"errors"
	"testing"
)

type testProvider1 struct {
	*stubProvider
}
type testProvider2 struct {
	*stubProvider
}

func TestCompositeProvider_Get(t *testing.T) {
	tests := map[string]struct {
		setStubs      func(p1 *stubProvider, p2 *stubProvider)
		expectedValue string
		expectedError string
	}{
		"exits_in_first_provider": {
			setStubs: func(p1 *stubProvider, p2 *stubProvider) {
				p1.set("key", "value", nil)
			},
			expectedValue: "value",
		},
		"exits_in_second_provider": {
			setStubs: func(p1 *stubProvider, p2 *stubProvider) {
				p1.set("key", "", errors.New("config 'key' does not exist"))
				p2.set("key", "value", nil)
			},
			expectedValue: "value",
		},
		"does_not_exist": {
			expectedValue: "",
			setStubs: func(p1 *stubProvider, p2 *stubProvider) {
				p1.set("key", "", errors.New("config 'key' does not exist"))
				p2.set("key", "", errors.New("config 'key' does not exist"))
			},
			expectedError: "config.testProvider1: config 'key' does not exist\nconfig.testProvider2: config 'key' does not exist",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p1 := testProvider1{
				stubProvider: &stubProvider{},
			}
			p2 := testProvider2{
				stubProvider: &stubProvider{},
			}
			if tt.setStubs != nil {
				tt.setStubs(p1.stubProvider, p2.stubProvider)
			}
			p := NewCompositeProvider(
				p1,
				p2,
			)
			got, err := p.Get(context.Background(), "key")
			assertErrorMessage(t, err, tt.expectedError)
			if got != tt.expectedValue {
				t.Fatalf("expected value %q, got %q", tt.expectedValue, got)
			}
		})
	}
}

func TestCompositeProvider_GetAndReportProvider(t *testing.T) {
	tests := map[string]struct {
		setStubs         func(p1 *stubProvider, p2 *stubProvider)
		expectedValue    string
		expectedProvider string
		expectedError    string
	}{
		"found_in_first_provider": {
			setStubs: func(p1 *stubProvider, p2 *stubProvider) {
				p1.set("key", "value1", nil)
			},
			expectedValue:    "value1",
			expectedProvider: "config.testProvider1",
		},
		"found_in_second_provider": {
			setStubs: func(p1 *stubProvider, p2 *stubProvider) {
				p1.set("key", "", errors.New("not found"))
				p2.set("key", "value2", nil)
			},
			expectedValue:    "value2",
			expectedProvider: "config.testProvider2",
		},
		"not_found_in_any_provider": {
			setStubs: func(p1 *stubProvider, p2 *stubProvider) {
				p1.set("key", "", errors.New("not found"))
				p2.set("key", "", errors.New("not found"))
			},
			expectedValue:    "",
			expectedProvider: "",
			expectedError:    "config.testProvider1: not found\nconfig.testProvider2: not found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p1 := testProvider1{
				stubProvider: &stubProvider{},
			}
			p2 := testProvider2{
				stubProvider: &stubProvider{},
			}
			if tt.setStubs != nil {
				tt.setStubs(p1.stubProvider, p2.stubProvider)
			}
			p := NewCompositeProvider(
				p1,
				p2,
			)
			gotValue, gotProvider, err := p.GetWithSource(context.Background(), "key")
			if gotValue != tt.expectedValue {
				t.Fatalf("expected value %q, got %q", tt.expectedValue, gotValue)
			}
			if gotProvider != tt.expectedProvider {
				t.Fatalf("expected provider %q, got %q", tt.expectedProvider, gotProvider)
			}
			assertErrorMessage(t, err, tt.expectedError)
		})
	}
}
