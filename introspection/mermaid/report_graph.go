package mermaid

import (
	"fmt"

	"github.com/cleitonmarx/symbiont/introspection"
)

const (
	emojiCodeLocation = "📍"
	emojiDep          = "🧩"
	emojiApp          = "🕷️"
	emojiCaller       = "🏗️"
	emojiConfig       = "🗝️"
	emojiService      = "📦"
)

var (
	// node styles
	styleDepUsed     = Style{Fill: "#e0f7fa", Stroke: "#00838f", StrokeWidth: "2px", Color: "#222222"}
	styleDepUnused   = Style{Fill: "#fce1e1", Stroke: "#a60202", StrokeWidth: "2px", Color: "#b26a00"}
	styleConfig      = Style{Fill: "#e8f5e9", Stroke: "#388e3c", StrokeWidth: "2px", Color: "#222222"}
	styleCaller      = Style{Fill: "#fff3e0", Stroke: "#f57c00", StrokeWidth: "2px", Color: "#222222"}
	styleRunnable    = Style{Fill: "#e3e0fc", Stroke: "#6c47a6", StrokeWidth: "2px", Color: "#222222"}
	styleApp         = Style{Fill: "#0525f5", Stroke: "black", StrokeWidth: "3px", Color: "#ffffff", FontWeight: "bold"}
	styleInitializer = Style{Fill: "#f0f0f0", Stroke: "#888888", StrokeWidth: "1px", Color: "#222222", FontWeight: "bold"}

	// sublines styles
	styleName           = Style{Color: "#b26a00", FontSize: "12px", IsHtml: true}
	styleDepImpl        = Style{Color: "darkgray", FontSize: "11px", IsHtml: true}
	styleDepWiring      = Style{Color: "darkblue", FontSize: "11px", IsHtml: true}
	styleCodeLoc        = Style{Color: "gray", FontSize: "11px", IsHtml: true}
	styleConfigProvider = Style{FontSize: "11px", Color: "green", IsHtml: true}
	styleConfigDefault  = Style{Color: "green", FontSize: "11px", IsHtml: true}
)

// GenerateIntrospectionGraph generates a Mermaid graph representation of the introspection report.
func GenerateIntrospectionGraph(r introspection.Report) string {
	var edges []Edge
	nodeMap := make(map[string]Node)
	depHasCaller := make(map[string]bool)

	// --- Dependencies ---
	depEdges, _ := buildDependencyGraph(nodeMap, depHasCaller, r.Deps)
	edges = append(edges, depEdges...)
	// --- Configs ---
	configEdges, _ := buildConfigGraph(nodeMap, r.Configs)
	edges = append(edges, configEdges...)
	// --- App node ---
	appNodeID := "Symbiont"
	if _, exists := nodeMap[appNodeID]; !exists {
		appLabel := LabelBuilder{
			Label:     fmt.Sprintf("Symbiont %s", emojiApp),
			FontSize:  20,
			FontColor: "white",
			Bold:      true,
		}.ToHTML()
		nodeMap[appNodeID] = Node{
			ID:    appNodeID,
			Label: appLabel,
			Type:  NodeApp,
			Style: styleApp,
		}
	}

	// --- Runnable ---
	buildRunnerGraph(r.Runners, nodeMap, &edges, appNodeID)
	// Remove duplicates and preserve order
	order := buildOrderedNodeIDs(nodeMap)
	// --- Set styles using declarative Style struct ---
	applyNodeStyles(nodeMap, depHasCaller)

	// --- Build Graph struct and render ---
	var nodes []Node
	for _, id := range order {
		nodes = append(nodes, nodeMap[id])
	}
	g := Graph{
		Nodes: nodes,
		Edges: edges,
	}
	return g.RenderTD()
}

// buildDependencyGraph constructs the dependency graph from introspection data.
func buildDependencyGraph(nodeMap map[string]Node, depHasCaller map[string]bool, deps []introspection.DepEvent) ([]Edge, []string) {
	edges := []Edge{}
	registeredOrder := []string{}
	for _, ev := range deps {
		dependency := ev.Impl + ev.Name
		if ev.Kind == introspection.DepRegistered {
			var sublines []string
			if ev.Name != "" {
				sublines = append(sublines, Subline(styleName, "name: %s", ev.Name))
			}
			if ev.Type != ev.Impl {
				sublines = append(sublines, Subline(styleDepImpl, "%s %s", emojiDep, ev.Impl))
			}
			sublines = append(sublines, Subline(styleDepWiring, "%s %s", emojiCaller, ev.Caller.Func))
			sublines = append(sublines, Subline(styleCodeLoc, "%s(%s:%d)", emojiCodeLocation, ev.Caller.File, ev.Caller.Line))

			label := LabelBuilder{
				Label:    ev.Type,
				FontSize: 16,
				Bold:     true,
				SubLines: sublines,
			}.ToHTML()

			nodeMap[dependency] = Node{
				ID:    dependency,
				Label: label,
				Type:  NodeDependency,
			}
			registeredOrder = append(registeredOrder, dependency)
		}
		if ev.Kind == introspection.DepResolved {
			toCaller := ev.Caller.Func
			if toCaller == "" {
				toCaller = ev.Component
			}
			if toCaller == "" {
				toCaller = ev.Type
			}
			edges = append(edges, Edge{From: dependency, To: toCaller})
			depHasCaller[dependency] = true

			label := LabelBuilder{
				Label:    toCaller,
				FontSize: 15,
				Bold:     true,
				SubLines: []string{
					Subline(styleCodeLoc, "%s(%s:%d)", emojiCodeLocation, ev.Caller.File, ev.Caller.Line),
				},
			}.ToHTML()

			nodeMap[toCaller] = Node{
				ID:    toCaller,
				Label: label,
				Type:  NodeCaller,
			}

			if _, exists := nodeMap[dependency]; !exists {
				var sublines []string
				if ev.Name != "" {
					sublines = append(sublines, Subline(styleName, "name: %s", ev.Name))
				}
				if ev.Type != ev.Impl {
					sublines = append(sublines, Subline(styleDepImpl, "impl: %s", ev.Impl))
				}
				label := LabelBuilder{Label: ev.Type, FontSize: 16, Bold: true,
					SubLines: sublines,
				}.ToHTML()

				nodeMap[dependency] = Node{
					ID:    dependency,
					Label: label,
					Type:  NodeDependency,
				}
			}
		}
	}
	return edges, registeredOrder
}

// buildConfigGraph constructs the configuration graph from introspection data.
func buildConfigGraph(nodeMap map[string]Node, configs []introspection.ConfigAccess) ([]Edge, []string) {
	edges := []Edge{}
	configOrder := []string{}
	for _, k := range configs {
		configKey := k.Key
		if _, exists := nodeMap[configKey]; !exists {
			configOrder = append(configOrder, configKey)
		}
		var sublines []string
		if k.Provider != "" {
			sublines = append(sublines, Subline(styleConfigProvider, "%s %s", emojiConfig, k.Provider))
		}
		if k.UsedDefault {
			sublines = append(sublines, Subline(styleConfigDefault, "default"))
		}
		label := LabelBuilder{
			Label:    k.Key,
			FontSize: 16,
			Bold:     true,
			SubLines: sublines,
		}.ToHTML()
		nodeMap[configKey] = Node{
			ID:    configKey,
			Label: label,
			Type:  NodeConfig,
		}

		caller := k.Caller.Func
		if caller == "" && k.Component != "" {
			caller = k.Component
		}
		if caller == "" {
			caller = "unknown caller"
		}
		edges = append(edges, Edge{From: configKey, To: caller})
		labelCaller := LabelBuilder{
			Label:    caller,
			FontSize: 15,
			Bold:     true,
			SubLines: []string{
				Subline(styleCodeLoc, "%s(%s:%d)", emojiCodeLocation, k.Caller.File, k.Caller.Line),
			},
		}.ToHTML()
		nodeMap[caller] = Node{
			ID:    caller,
			Label: labelCaller,
			Type:  NodeCaller,
		}
	}
	return edges, configOrder
}

// buildRunnerGraph builds runnable nodes and returns their IDs in order.
func buildRunnerGraph(runnerInfos []introspection.RunnerInfo, nodeMap map[string]Node, edges *[]Edge, appNodeId string) {
	for _, runnableInfo := range runnerInfos {
		runnableID := runnableInfo.Type
		label := LabelBuilder{
			Label:    runnableID,
			FontSize: 16,
			Bold:     true,
			SubLines: []string{
				Subline(styleConfigDefault, "%s Runnable", emojiService),
			},
		}.ToHTML()
		nodeMap[runnableID] = Node{
			ID:    runnableID,
			Label: label,
			Type:  NodeRunnable,
			Style: styleCaller,
		}
		*edges = append(*edges, Edge{From: runnableID, To: appNodeId})
	}
}

// applyNodeStyles applies styles to nodes based on their type and whether they have callers.
func applyNodeStyles(nodeMap map[string]Node, depHasCaller map[string]bool) {
	for id, n := range nodeMap {
		switch n.Type {
		case NodeDependency:
			if !depHasCaller[id] {
				n.Style = styleDepUnused
			} else {
				n.Style = styleDepUsed
			}
		case NodeConfig:
			n.Style = styleConfig
		case NodeCaller:
			n.Style = styleCaller
		case NodeRunnable:
			n.Style = styleRunnable
		case NodeApp:
			n.Style = styleApp
		case NodeInitializer:
			n.Style = styleInitializer
		}
		nodeMap[id] = n
	}
}

// buildOrderedNodeIDs returns a deduplicated, ordered list of node IDs for rendering.
// The order is: deps/configs -> callers -> runnables -> app.
func buildOrderedNodeIDs(nodeMap map[string]Node) []string {
	var depConfigOrder, callerOrder, runnableOrderOrdered []string
	var appNode string
	for id, node := range nodeMap {
		switch node.Type {
		case NodeDependency, NodeConfig:
			depConfigOrder = append(depConfigOrder, id)
		case NodeCaller, NodeInitializer:
			callerOrder = append(callerOrder, id)
		case NodeRunnable:
			runnableOrderOrdered = append(runnableOrderOrdered, id)
		case NodeApp:
			appNode = id
		}
	}

	seen := make(map[string]struct{})
	order := []string{}
	for _, id := range depConfigOrder {
		if _, ok := seen[id]; !ok {
			order = append(order, id)
			seen[id] = struct{}{}
		}
	}
	for _, id := range callerOrder {
		if _, ok := seen[id]; !ok {
			order = append(order, id)
			seen[id] = struct{}{}
		}
	}
	for _, id := range runnableOrderOrdered {
		if _, ok := seen[id]; !ok {
			order = append(order, id)
			seen[id] = struct{}{}
		}
	}
	if appNode != "" {
		order = append(order, appNode)
	}
	return order
}
