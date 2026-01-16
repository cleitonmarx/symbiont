package mermaid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStyle_ToCSS(t *testing.T) {
	tests := []struct {
		name string
		s    Style
		want string
	}{
		{"empty", Style{}, ""},
		{"graph style", Style{Fill: "#fff", Stroke: "#000", StrokeWidth: "2px", Color: "#222222"}, "fill:#fff,stroke:#000,stroke-width:2px,color:#222222"},
		{"html style", Style{Color: "#888888", FontSize: "12px", IsHtml: true}, "color:#888888;font-size:12px;"},
		{"font weight", Style{FontWeight: "bold"}, "font-weight:bold"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.ToCSS()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLabelBuilder_ToHTML(t *testing.T) {
	l := LabelBuilder{
		Label:     "Main",
		FontSize:  16,
		FontColor: "#0d47a1",
		Bold:      true,
		SubLines: []string{
			"<span style='font-size:14px;color:#388e3c;'>Green Line</span>",
			"<span style='font-size:13px;color:#f57c00;'>Orange Line</span>",
		},
	}
	got := l.ToHTML()
	assert.Equal(t, "<b><span style='font-size:16px;color:#0d47a1'>Main</span></b><br/><span style='font-size:14px;color:#388e3c;'>Green Line</span><br/><span style='font-size:13px;color:#f57c00;'>Orange Line</span>", got)
}

func TestSubline(t *testing.T) {
	tests := []struct {
		name  string
		style Style
		text  string
		args  []any
		want  string
	}{
		{
			name:  "custom_style",
			style: Style{Color: "#888888", FontSize: "11px"},
			text:  "test: %d",
			args:  []any{42},
			want:  "<span style='color:#888888,font-size:11px'>test: 42</span>",
		},
		{
			name:  "empty_style",
			style: Style{Color: "", FontSize: ""},
			text:  "test: %d",
			args:  []any{42},
			want:  "<span>test: 42</span>",
		},
		{
			name:  "html_style",
			style: Style{Color: "#888888", FontSize: "11px", IsHtml: true},
			text:  "test: %d",
			args:  []any{42},
			want:  "<span style='color:#888888;font-size:11px;'>test: 42</span>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Subline(tt.style, tt.text, tt.args...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGraph_RenderTD(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "A", Label: "<b>A</b>", Style: Style{Fill: "#fff", Stroke: "#000"}},
			{ID: "B", Label: "<b>B</b>", Style: Style{Fill: "#eee", Stroke: "#111"}, Class: "special"},
		},
		Edges: []Edge{
			{From: "A", To: "B"},
		},
	}
	out := g.RenderTD()
	assert.Equal(t, "graph TD\n    subgraph DEPSUB[\" \"]\n        A[\"<b>A</b>\"]\n        B[\"<b>B</b>\"]\n    end\n    A --> B\n    style A fill:#fff,stroke:#000\n    style B fill:#eee,stroke:#111\n    class B special;\n    style DEPSUB fill:white,stroke-width:0px\n", out)
}
