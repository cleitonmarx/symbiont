package mermaid

import (
	"strings"
	"testing"
)

func TestGraph_RenderTD(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "dep1", Label: "Dep1", Type: NodeDependency},
			{ID: "caller1", Label: "Caller1", Type: NodeCaller},
		},
		Edges: []Edge{{From: "dep1", To: "caller1"}},
	}

	out := g.RenderTD()
	if !strings.Contains(out, "dep1") || !strings.Contains(out, "caller1") {
		t.Fatalf("expected node labels in output, got: %s", out)
	}
	if !strings.Contains(out, "dep1 --> caller1") {
		t.Fatalf("expected edge in output, got: %s", out)
	}
}
