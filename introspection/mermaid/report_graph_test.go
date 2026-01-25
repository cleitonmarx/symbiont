package mermaid

import (
	"testing"

	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/stretchr/testify/assert"
)

func TestGenerateIntrospectionGraph_IncludesInitializers(t *testing.T) {
	report := introspection.Report{
		Configs: []introspection.ConfigAccess{
			{Key: "cfg", Provider: "provider", Caller: introspection.Caller{Func: "examples.(*initLogger).Initialize", File: "f", Line: 1}},
		},
		Deps: []introspection.DepEvent{
			{Kind: introspection.DepRegistered, Type: "Dep", Impl: "DepImpl", Caller: introspection.Caller{Func: "examples.(*initLogger).Initialize", File: "f", Line: 1}},
			{Kind: introspection.DepResolved, Type: "Dep", Impl: "DepImpl", Caller: introspection.Caller{Func: "run1", File: "f", Line: 2}},
		},
		Runners:      []introspection.RunnerInfo{{Type: "run1"}},
		Initializers: []introspection.InitializerInfo{{Type: "*examples.initLogger"}},
	}

	out := GenerateIntrospectionGraph(report)
	println(out)

	assert.Contains(t, out, sanitizeID("*examples.initLogger")+" --o DepImpl")
	assert.Contains(t, out, "DepImpl -.-> run1")
	assert.Contains(t, out, "cfg -.-> "+sanitizeID("*examples.initLogger"))
	assert.Contains(t, out, "run1 --- Symbiont")
}
