package mermaid

import (
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont/introspection"
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
	if !strings.Contains(out, sanitizeID("initLogger")+" --o "+sanitizeID(dependencyNodeID(introspection.DepEvent{Type: "Dep", Name: "depName", Impl: "DepImpl"}))) {
		t.Fatal("expected initializer to dependency edge for initLogger")
	}
	if !strings.Contains(out, sanitizeID("initOther")+" --o "+sanitizeID(dependencyNodeID(introspection.DepEvent{Type: "UnusedDep", Name: "unused", Impl: "UnusedDepImpl"}))) {
		t.Fatal("expected initializer to dependency edge for initOther")
	}

	// Dep to runner
	if !strings.Contains(out, sanitizeID(dependencyNodeID(introspection.DepEvent{Type: "Dep", Name: "depName", Impl: "DepImpl"}))+" -.-> run1") {
		t.Fatal("expected dependency to run1 edge")
	}
	if !strings.Contains(out, sanitizeID(dependencyNodeID(introspection.DepEvent{Type: "GhostDep", Name: "ghost", Impl: "GhostDepImpl"}))+" -.-> run2") {
		t.Fatal("expected ghost dependency to run2 edge")
	}

	// Config to initializer/caller
	if !strings.Contains(out, "cfg -.-> "+sanitizeID("initLogger")) ||
		!strings.Contains(out, "cfgDefault -.-> "+sanitizeID("initOther")) ||
		!strings.Contains(out, "cfgOrphan -.-> orphanInit") {
		t.Fatal("expected config edges in graph output")
	}

	// Runner to Symbiont
	if !strings.Contains(out, "run1 --- SymbiontApp") || !strings.Contains(out, "run2 --- SymbiontApp") {
		t.Fatal("expected runnable edges to app")
	}

	// graph type
	if !strings.Contains(out, "graph TD") {
		t.Fatal("expected graph TD header")
	}

	// No duplicate edges for duplicate runners/initializers
	if strings.Count(out, sanitizeID("initLogger")+" --o "+sanitizeID(dependencyNodeID(introspection.DepEvent{Type: "Dep", Name: "depName", Impl: "DepImpl"}))) != 1 {
		t.Fatal("expected one initLogger edge")
	}
	if strings.Count(out, sanitizeID(dependencyNodeID(introspection.DepEvent{Type: "Dep", Name: "depName", Impl: "DepImpl"}))+" -.-> run1") != 1 {
		t.Fatal("expected one dependency to run1 edge")
	}
}

func TestGenerateIntrospectionGraph_DistinctDepsForSameImpl(t *testing.T) {
	initType := "*infra.InitSharedClient"
	initCaller := "infra.InitSharedClient.Initialize"
	impl := "infra.SharedClient"
	types := []string{
		"contracts.Client",
		"contracts.Reader",
		"contracts.Writer",
		"contracts.HealthChecker",
	}

	deps := make([]introspection.DepEvent, 0, len(types))
	for _, depType := range types {
		deps = append(deps, introspection.DepEvent{
			Kind:   introspection.DepRegistered,
			Type:   depType,
			Impl:   impl,
			Caller: introspection.Caller{Func: initCaller, File: "shared_client.go", Line: 120},
		})
	}

	out := GenerateIntrospectionGraph(introspection.Report{
		Deps: deps,
		Initializers: []introspection.InitializerInfo{
			{Type: initType},
		},
	})

	for _, depType := range types {
		depID := sanitizeID(dependencyNodeID(introspection.DepEvent{Type: depType, Impl: impl}))
		if !strings.Contains(out, sanitizeID(initType)+" --o "+depID) {
			t.Fatalf("expected edge for dependency type %q", depType)
		}
	}
}
