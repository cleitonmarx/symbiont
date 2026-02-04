package introspection

import (
	"encoding/json"
	"reflect"
)

// Report aggregates introspection data for configs, dependencies, and runners.
type Report struct {
	Configs      []ConfigAccess    `json:"configs"`
	Deps         []DepEvent        `json:"deps"`
	Runners      []RunnerInfo      `json:"runners"`
	Initializers []InitializerInfo `json:"initializers"`
}

// ConfigAccess captures a single configuration key access.
type ConfigAccess struct {
	Key         string `json:"key"`
	Provider    string `json:"provider"`
	UsedDefault bool   `json:"usedDefault"`
	Caller      Caller `json:"caller"`
	Component   string `json:"component"`
	Order       int    `json:"order"`
}

// DepEventKind describes the type of dependency event.
type DepEventKind string

const (
	DepRegistered DepEventKind = "register"
	DepResolved   DepEventKind = "resolve"
)

// DepEvent represents a dependency registration or resolution.
type DepEvent struct {
	Kind      DepEventKind `json:"kind"`
	Type      string       `json:"type"` // dependency interface/type name
	Name      string       `json:"name"` // optional named binding
	Impl      string       `json:"impl"` // concrete implementation type
	Caller    Caller       `json:"caller"`
	Component string       `json:"component"` // consumer/owner type if known
	Order     int          `json:"order"`     // monotonic order of events within the run
}

// RunnerInfo describes a runnable that was registered with the app.
type RunnerInfo struct {
	Type      string       // type name
	Component reflect.Type // raw type if needed for reflection
}

// InitializerInfo describes an initializer registered with the app.
type InitializerInfo struct {
	Type      string       // type name
	Component reflect.Type // raw type if needed for reflection
}

// Caller identifies the code location that produced an event.
type Caller struct {
	Func string `json:"func"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// SerializableReport is a JSON-friendly representation of Report.
// It omits reflection-heavy fields that do not marshal cleanly.
type SerializableReport struct {
	Configs      []ConfigAccess                `json:"configs"`
	Deps         []DepEvent                    `json:"deps"`
	Runners      []SerializableRunnerInfo      `json:"runners"`
	Initializers []SerializableInitializerInfo `json:"initializers"`
}

// SerializableRunnerInfo is a JSON-friendly representation of RunnerInfo.
type SerializableRunnerInfo struct {
	Type string `json:"type"`
}

// SerializableInitializerInfo is a JSON-friendly representation of InitializerInfo.
type SerializableInitializerInfo struct {
	Type string `json:"type"`
}

// ToSerializable converts Report into a JSON-friendly representation.
func (r Report) ToSerializable() SerializableReport {
	runners := make([]SerializableRunnerInfo, 0, len(r.Runners))
	for _, rn := range r.Runners {
		runners = append(runners, SerializableRunnerInfo{Type: rn.Type})
	}
	initializers := make([]SerializableInitializerInfo, 0, len(r.Initializers))
	for _, init := range r.Initializers {
		initializers = append(initializers, SerializableInitializerInfo{Type: init.Type})
	}
	return SerializableReport{
		Configs:      r.Configs,
		Deps:         r.Deps,
		Runners:      runners,
		Initializers: initializers,
	}
}

// MarshalJSON implements the json.Marshaler interface for Report.
func (r Report) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.ToSerializable())
}
