package introspection

import (
	"encoding/json"
	"reflect"
)

// Report aggregates introspection data for configs, dependencies, and runners.
type Report struct {
	Configs []ConfigAccess
	Deps    []DepEvent
	Runners []RunnerInfo
}

// ConfigAccess captures a single configuration key access.
type ConfigAccess struct {
	Key         string
	Provider    string // empty when a default was used
	UsedDefault bool
	Caller      Caller
	Component   string // optional runnable/initializer type name
	Order       int    // monotonic order of access within the run
}

// DepEventKind describes the type of dependency event.
type DepEventKind string

const (
	DepRegistered DepEventKind = "register"
	DepResolved   DepEventKind = "resolve"
)

// DepEvent represents a dependency registration or resolution.
type DepEvent struct {
	Kind      DepEventKind
	Type      string // dependency interface/type name
	Name      string // optional named binding
	Impl      string // concrete implementation type
	Caller    Caller
	Component string // consumer/owner type if known
	Order     int    // monotonic order of events within the run
}

// RunnerInfo describes a runnable that was registered with the app.
type RunnerInfo struct {
	Type      string       // type name
	Component reflect.Type // raw type if needed for reflection
}

// Caller identifies the code location that produced an event.
type Caller struct {
	Func string
	File string
	Line int
}

// SerializableReport is a JSON-friendly representation of Report.
// It omits reflection-heavy fields that do not marshal cleanly.
type SerializableReport struct {
	Configs []ConfigAccess           `json:"configs"`
	Deps    []DepEvent               `json:"deps"`
	Runners []SerializableRunnerInfo `json:"runners"`
}

// SerializableRunnerInfo is a JSON-friendly representation of RunnerInfo.
type SerializableRunnerInfo struct {
	Type string `json:"type"`
}

// ToSerializable converts Report into a JSON-friendly representation.
func (r Report) ToSerializable() SerializableReport {
	runners := make([]SerializableRunnerInfo, 0, len(r.Runners))
	for _, rn := range r.Runners {
		runners = append(runners, SerializableRunnerInfo{Type: rn.Type})
	}
	return SerializableReport{
		Configs: r.Configs,
		Deps:    r.Deps,
		Runners: runners,
	}
}

// ToJSON marshals the report into JSON using a serializable view.
func (r Report) ToJSON() ([]byte, error) {
	return json.Marshal(r.ToSerializable())
}
