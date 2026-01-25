package config

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvVarProvider_Get(t *testing.T) {
	os.Setenv("EXISTING_KEY", "some_value") //nolint:errcheck

	t.Cleanup(func() {
		os.Unsetenv("EXISTING_KEY") //nolint:errcheck
	})

	tests := map[string]struct {
		envKey      string
		want        string
		expectedErr error
	}{
		"existing_key": {
			envKey:      "EXISTING_KEY",
			want:        "some_value",
			expectedErr: nil,
		},
		"missing_key": {
			envKey:      "MISSING_KEY",
			want:        "",
			expectedErr: errors.New("environment variable 'MISSING_KEY' is not set"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := NewEnvVarProvider()
			got, err := p.Get(context.Background(), tt.envKey)
			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
