package config

import (
	"context"
	"os"
	"testing"
)

func TestEnvVarProvider_Get(t *testing.T) {
	os.Setenv("EXISTING_KEY", "some_value") //nolint:errcheck

	t.Cleanup(func() {
		os.Unsetenv("EXISTING_KEY") //nolint:errcheck
	})

	tests := map[string]struct {
		envKey      string
		want        string
		expectedErr string
	}{
		"existing_key": {
			envKey: "EXISTING_KEY",
			want:   "some_value",
		},
		"missing_key": {
			envKey:      "MISSING_KEY",
			want:        "",
			expectedErr: "environment variable 'MISSING_KEY' is not set",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := NewEnvVarProvider()
			got, err := p.Get(context.Background(), tt.envKey)
			assertErrorMessage(t, err, tt.expectedErr)
			if got != tt.want {
				t.Fatalf("expected value %q, got %q", tt.want, got)
			}
		})
	}
}
