package depend

import (
	"reflect"
	"testing"

	"github.com/cleitonmarx/symbiont/introspection"
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
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d events, got %d", tt.wantLen, len(got))
			}
			if tt.wantLen > 0 {
				for i, kind := range tt.wantKinds {
					if got[i].Kind != kind {
						t.Fatalf("expected event kind %v at index %d, got %v", kind, i, got[i].Kind)
					}
				}
				// Ensure returned slice is a copy
				got[0].Name = "changed"
				got2 := GetEvents()
				if got2[0].Name == "changed" {
					t.Fatal("expected GetEvents to return a copy")
				}
			}
		})
	}
}
