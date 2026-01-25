package depend

import (
	"reflect"
	"testing"

	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/stretchr/testify/assert"
)

func TestGetEvents(t *testing.T) {
	resetEvents := func() {
		eventMu.Lock()
		events = nil
		order = 0
		eventMu.Unlock()
	}

	tests := []struct {
		name      string
		setup     func()
		wantLen   int
		wantKinds []introspection.DepEventKind
	}{
		{
			name:    "no-events",
			setup:   func() { resetEvents() },
			wantLen: 0,
		},
		{
			name: "some-events",
			setup: func() {
				resetEvents()
				logEvent(introspection.DepRegistered, "string", "dep1", "impl1", reflect.TypeOf(0), 1)
				logEvent(introspection.DepResolved, "int", "dep2", "impl2", reflect.TypeOf(""), 1)
			},
			wantLen:   2,
			wantKinds: []introspection.DepEventKind{introspection.DepRegistered, introspection.DepResolved},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got := GetEvents()
			assert.Len(t, got, tt.wantLen)
			if tt.wantLen > 0 {
				for i, kind := range tt.wantKinds {
					assert.Equal(t, kind, got[i].Kind)
				}
				// Ensure returned slice is a copy
				got[0].Name = "changed"
				got2 := GetEvents()
				assert.NotEqual(t, "changed", got2[0].Name)
			}
		})
	}
}
