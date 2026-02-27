package introspection

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestReport_MarshalJSON(t *testing.T) {
	tests := []struct {
		name         string
		report       Report
		expectedJson string
	}{
		{
			name:         "empty-report",
			report:       Report{},
			expectedJson: `{"configs":null,"deps":null,"runners":[],"initializers":[]}`,
		},
		{
			name: "populated-report",
			report: Report{
				Configs: []ConfigAccess{
					{Key: "foo", Provider: "prov", UsedDefault: false, Order: 1},
				},
				Deps: []DepEvent{
					{Kind: DepRegistered, Type: "string", Name: "dep", Impl: "impl", Order: 2},
				},
				Runners: []RunnerInfo{
					{Type: "myRunner"},
				},
				Initializers: []InitializerInfo{
					{Type: "myInit"},
				},
			},
			expectedJson: `{"configs":[{"key":"foo","provider":"prov","usedDefault":false,"caller":{"func":"","file":"","line":0},"component":"","order":1}],"deps":[{"kind":"register","type":"string","name":"dep","impl":"impl","caller":{"func":"","file":"","line":0},"component":"","order":2}],"runners":[{"type":"myRunner"}],"initializers":[{"type":"myInit"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.report)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}
			var js map[string]any
			if err := json.Unmarshal(data, &js); err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			var expected any
			var actual any
			if err := json.Unmarshal([]byte(tt.expectedJson), &expected); err != nil {
				t.Fatalf("invalid expected JSON: %v", err)
			}
			if err := json.Unmarshal(data, &actual); err != nil {
				t.Fatalf("invalid actual JSON: %v", err)
			}
			if !reflect.DeepEqual(expected, actual) {
				t.Fatalf("expected JSON %s, got %s", tt.expectedJson, string(data))
			}
		})
	}
}
