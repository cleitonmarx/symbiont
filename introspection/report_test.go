package introspection

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
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
			assert.NoError(t, err)
			// Should be valid JSON
			var js map[string]any
			assert.NoError(t, json.Unmarshal(data, &js))
			// Check for expected substrings
			assert.JSONEq(t, tt.expectedJson, string(data))
		})
	}
}
