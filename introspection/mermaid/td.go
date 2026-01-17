package mermaid

import (
	"fmt"
	"sort"
	"strings"
)

// NodeType represents the type of a node in the graph.
type NodeType int

const (
	// NodeDependency represents a dependency node.
	NodeDependency NodeType = iota
	// NodeConfig represents a configuration node.
	NodeConfig
	// NodeApp represents the app root node.
	NodeApp
	// NodeRunnable represents a runnable node.
	NodeRunnable
	// NodeInitializer represents a initializer node.
	NodeInitializer
	// NodeCaller represents a generic caller node.
	NodeCaller
)

// Node represents a node in the Mermaid graph.
type Node struct {
	ID    string
	Label string
	Type  NodeType
	Style Style
	Class string
}

// Edge represents a directed edge in the Mermaid graph.
// It connects two nodes by their IDs.
type Edge struct {
	From  string
	To    string
	Arrow string // Optional arrow style (e.g., "---|>", "---o>")
}

// Graph represents a Mermaid graph with nodes and edges.
type Graph struct {
	Nodes []Node
	Edges []Edge
}

// Style represents the style of a node in the graph.
type Style struct {
	Fill        string
	Stroke      string
	StrokeWidth string
	// Color applies to text color.
	Color      string
	FontWeight string
	FontSize   string
	IsHtml     bool
}

func (s Style) ToCSS() string {
	var parts []string
	if s.Fill != "" {
		parts = append(parts, "fill:"+s.Fill)
	}
	if s.Stroke != "" {
		parts = append(parts, "stroke:"+s.Stroke)
	}
	if s.StrokeWidth != "" {
		parts = append(parts, "stroke-width:"+s.StrokeWidth)
	}
	if s.Color != "" {
		parts = append(parts, "color:"+s.Color)
	}
	if s.FontWeight != "" {
		parts = append(parts, "font-weight:"+s.FontWeight)
	}
	if s.FontSize != "" {
		parts = append(parts, "font-size:"+s.FontSize)
	}
	if len(parts) == 0 {
		return ""
	}
	if s.IsHtml {
		return strings.Join(parts, ";") + ";"
	}
	return strings.Join(parts, ",")
}

// LabelBuilder helps build HTML labels for nodes in a declarative way.
type LabelBuilder struct {
	Label     string
	FontSize  int
	FontColor string
	Bold      bool
	SubLines  []string
}

func (l LabelBuilder) ToHTML() string {
	var styleParts []string
	if l.FontSize > 0 {
		styleParts = append(styleParts, fmt.Sprintf("font-size:%dpx", l.FontSize))
	}
	if l.FontColor != "" {
		styleParts = append(styleParts, fmt.Sprintf("color:%s", l.FontColor))
	}
	styleAttr := ""
	if len(styleParts) > 0 {
		styleAttr = fmt.Sprintf(" style='%s'", strings.Join(styleParts, ";"))
	}

	main := fmt.Sprintf("<span%s>%s</span>", styleAttr, l.Label)
	if l.Bold {
		main = "<b>" + main + "</b>"
	}

	var sub string
	if len(l.SubLines) > 0 {
		sub = "<br/>" + strings.Join(l.SubLines, "<br/>")
	}
	return main + sub
}

// Subline creates a subline for a node label with the given text and style.
func Subline(style Style, format string, args ...any) string {
	content := fmt.Sprintf(format, args...)
	css := style.ToCSS()
	if css != "" {
		return fmt.Sprintf("<span style='%s'>%s</span>", css, content)
	}
	return fmt.Sprintf("<span>%s</span>", content)
}

// RenderTD renders the graph in Mermaid TD (top-down) format.
func (g *Graph) RenderTD() string {
	var b strings.Builder
	b.WriteString("graph TD\n")

	// Group nodes by type
	var depsConfigs, callers, runnables, apps []Node
	for _, n := range g.Nodes {
		switch n.Type {
		case NodeDependency, NodeConfig:
			depsConfigs = append(depsConfigs, n)
		case NodeCaller, NodeInitializer:
			callers = append(callers, n)
		case NodeRunnable:
			runnables = append(runnables, n)
		case NodeApp:
			apps = append(apps, n)
		}
	}

	// Render subgraphs for each layer (without names and without subgraph style)
	if len(depsConfigs) > 0 {
		for _, n := range depsConfigs {
			id := sanitizeID(n.ID)
			fmt.Fprintf(&b, "	%s[\"%s\"]\n", id, n.Label)
		}
	}
	if len(callers) > 0 {
		for _, n := range callers {
			id := sanitizeID(n.ID)
			fmt.Fprintf(&b, "	%s[\"%s\"]\n", id, n.Label)
		}
	}
	if len(runnables) > 0 {
		for _, n := range runnables {
			id := sanitizeID(n.ID)
			fmt.Fprintf(&b, "	%s[\"%s\"]\n", id, n.Label)
		}
	}
	if len(apps) > 0 {
		for _, n := range apps {
			id := sanitizeID(n.ID)
			fmt.Fprintf(&b, "	%s[\"%s\"]\n", id, n.Label)
		}
	}

	// Render edges (sorted for determinism)
	edges := make([]Edge, len(g.Edges))
	copy(edges, g.Edges)
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From == edges[j].From {
			return edges[i].To < edges[j].To
		}
		return edges[i].From < edges[j].From
	})
	for _, e := range edges {
		from := sanitizeID(e.From)
		to := sanitizeID(e.To)
		if e.Arrow != "" {
			fmt.Fprintf(&b, "    %s %s %s\n", from, e.Arrow, to)
		} else {
			fmt.Fprintf(&b, "    %s --> %s\n", from, to)
		}
	}

	// Render styles
	for _, n := range g.Nodes {
		id := sanitizeID(n.ID)
		if n.Style.ToCSS() != "" {
			fmt.Fprintf(&b, "    style %s %s\n", id, n.Style.ToCSS())
		}
		if n.Class != "" {
			fmt.Fprintf(&b, "    class %s %s;\n", id, n.Class)
		}
	}

	return b.String()
}

// sanitizeID replaces characters in a string to make it suitable for use as an ID in Mermaid graphs.
func sanitizeID(s string) string {
	replacer := strings.NewReplacer(
		" ", "_",
		".", "_",
		"(", "_",
		")", "_",
		":", "_",
		"*", "ptr_",
		",", "_",
		"[", "_",
		"]", "_",
		"-", "_",
	)
	return replacer.Replace(s)
}
