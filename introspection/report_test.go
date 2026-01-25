package introspection

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReport_ToJSON(t *testing.T) {
	tests := []struct {
		name         string
		report       Report
		expectedJson string
	}{
		{
			name:         "empty-report",
			report:       Report{},
			expectedJson: `{"Configs":[],"Deps":[],"Runners":[],"Initializers":[]}`,
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
			expectedJson: `{"Configs":[{"Key":"foo","Provider":"prov","UsedDefault":false,"Order":1}],"Deps":[{"Kind":0,"Type":"string","Name":"dep","Impl":"impl","Order":2}],"Runners":[{"Type":"myRunner"}],"Initializers":[{"Type":"myInit"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.report.ToJSON()
			assert.NoError(t, err)
			// Should be valid JSON
			var js map[string]any
			assert.NoError(t, json.Unmarshal(data, &js))
			// Check for expected substrings

		})
	}
}
