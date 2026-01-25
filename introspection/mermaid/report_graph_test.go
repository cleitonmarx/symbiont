package mermaid

import (
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/stretchr/testify/assert"
)

func TestGenerateIntrospectionGraph_Coverage(t *testing.T) {
	report := introspection.Report{
		Configs: []introspection.ConfigAccess{
			// Config with provider and used default false
			{Key: "cfg", Provider: "provider", UsedDefault: false, Caller: introspection.Caller{Func: "initLogger", File: "f", Line: 1}},
			// Config with no provider and used default true
			{Key: "cfgDefault", Provider: "", UsedDefault: true, Caller: introspection.Caller{Func: "initOther", File: "f2", Line: 2}},
			// Config with provider but no matching initializer
			{Key: "cfgOrphan", Provider: "provider2", UsedDefault: false, Caller: introspection.Caller{Func: "orphanInit", File: "f3", Line: 3}},
		},
		Deps: []introspection.DepEvent{
			// Registered and resolved dep
			{Kind: introspection.DepRegistered, Type: "Dep", Impl: "DepImpl", Name: "depName", Caller: introspection.Caller{Func: "initLogger", File: "f", Line: 1}},
			{Kind: introspection.DepResolved, Type: "Dep", Impl: "DepImpl", Name: "depName", Caller: introspection.Caller{Func: "run1", File: "f", Line: 2}},
			// Registered but never resolved dep (unused)
			{Kind: introspection.DepRegistered, Type: "UnusedDep", Impl: "UnusedDepImpl", Name: "unused", Caller: introspection.Caller{Func: "initOther", File: "f2", Line: 3}},
			// Resolved but never registered dep (edge)
			{Kind: introspection.DepResolved, Type: "GhostDep", Impl: "GhostDepImpl", Name: "ghost", Caller: introspection.Caller{Func: "run2", File: "f2", Line: 4}},
		},
		Runners: []introspection.RunnerInfo{
			{Type: "run1"},
			{Type: "run2"},
			// Duplicate runner type (should not duplicate nodes/edges)
			{Type: "run1"},
		},
		Initializers: []introspection.InitializerInfo{
			{Type: "initLogger"},
			{Type: "initOther"},
			// Duplicate initializer type
			{Type: "initLogger"},
		},
	}

	out := GenerateIntrospectionGraph(report)

	// Initializer to dep
	assert.Contains(t, out, sanitizeID("initLogger")+" --o DepImpldepName")
	assert.Contains(t, out, sanitizeID("initOther")+" --o UnusedDepImplunused")

	// Dep to runner
	assert.Contains(t, out, "DepImpldepName -.-> run1")
	assert.Contains(t, out, "GhostDepImplghost -.-> run2")

	// Config to initializer/caller
	assert.Contains(t, out, "cfg -.-> "+sanitizeID("initLogger"))
	assert.Contains(t, out, "cfgDefault -.-> "+sanitizeID("initOther"))
	assert.Contains(t, out, "cfgOrphan -.-> orphanInit")

	// Runner to Symbiont
	assert.Contains(t, out, "run1 --- SymbiontApp")
	assert.Contains(t, out, "run2 --- SymbiontApp")

	// graph type
	assert.Contains(t, out, "graph TD")

	// No duplicate edges for duplicate runners/initializers
	assert.Equal(t, 1, strings.Count(out, sanitizeID("initLogger")+" --o DepImpldepName"))
	assert.Equal(t, 1, strings.Count(out, "DepImpldepName -.-> run1"))
}
