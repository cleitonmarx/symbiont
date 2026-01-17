package mermaid

import (
	"fmt"
	"strings"

	"github.com/cleitonmarx/symbiont/introspection"
)

const (
	emojiCodeLocation   = "📍"
	emojiInterface      = "🧩"
	emojiDep            = "💉"
	emojiCaller         = "🏗️"
	emojiConfig         = "🔑"
	emojiConfigProvider = "🫴🏽"
	emojiRunnable       = "⚙️"
	emojiInitializer    = "📦"
	emojiApp            = "🚀"
)

var (
	// node styles
	styleDepUsed   = Style{Fill: "#d6fff9", Stroke: "#2ec4b6", StrokeWidth: "2px", Color: "#222222"}
	styleDepUnused = Style{Fill: "#fce1e1", Stroke: "#a60202", StrokeWidth: "2px", Color: "#b26a00"}
	styleConfig    = Style{Fill: "#f1f7d2", Stroke: "#a7c957", StrokeWidth: "2px", Color: "#222222"}
	styleCaller    = Style{Fill: "#fff3e0", Stroke: "#f57c00", StrokeWidth: "2px", Color: "#222222"}
	styleRunnable  = Style{Fill: "#f1e8ff", Stroke: "#7b2cbf", StrokeWidth: "2px", Color: "#222222"}
	// fill:#0f56c4,stroke:#68a4eb,stroke-width:6px
	styleApp         = Style{Fill: "#0f56c4", Stroke: "#68a4eb", StrokeWidth: "6px", Color: "#ffffff", FontWeight: "bold"}
	styleInitializer = Style{Fill: "#f0f0f0", Stroke: "#373636", StrokeWidth: "1px", Color: "#222222", FontWeight: "bold"}

	// sublines styles
	styleName           = Style{Color: "#b26a00", FontSize: "12px", IsHtml: true}
	styleDepImpl        = Style{Color: "darkgray", FontSize: "11px", IsHtml: true}
	styleDepWiring      = Style{Color: "darkblue", FontSize: "11px", IsHtml: true}
	styleCodeLoc        = Style{Color: "gray", FontSize: "11px", IsHtml: true}
	styleConfigProvider = Style{FontSize: "11px", Color: "green", IsHtml: true}
	styleConfigDefault  = Style{Color: "green", FontSize: "11px", IsHtml: true}
	styleTypeName       = Style{Color: "green", FontSize: "11px", IsHtml: true}
)

// GenerateIntrospectionGraph generates a Mermaid graph representation of the introspection report.
func GenerateIntrospectionGraph(r introspection.Report) string {
	var edges []Edge
	nodeMap := make(map[string]Node)
	depHasCaller := make(map[string]bool)
	initializerTypes := make(map[string]struct{}, len(r.Initializers))
	for _, init := range r.Initializers {
		initializerTypes[init.Type] = struct{}{}
	}

	// --- App node ---
	appNodeID := "SymbiontApp"
	appLabel := LabelBuilder{
		Label:     fmt.Sprintf("%s Symbiont App", emojiApp),
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

	// --- Configs ---
	buildConfigGraph(nodeMap, initializerTypes, r.Configs, &edges)
	// --- Initializers ---
	buildInitializerGraph(r.Initializers, nodeMap)
	// --- Dependencies ---
	buildDependencyGraph(nodeMap, depHasCaller, initializerTypes, r.Deps, &edges)
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
func buildDependencyGraph(nodeMap map[string]Node, depHasCaller map[string]bool, initializerTypes map[string]struct{}, deps []introspection.DepEvent, edges *[]Edge) {
	for _, ev := range deps {
		dependency := ev.Impl + ev.Name
		if ev.Kind == introspection.DepRegistered {
			var sublines []string
			if ev.Name != "" {
				sublines = append(sublines, Subline(styleName, "name: %s", ev.Name))
			}
			if ev.Type != ev.Impl {
				sublines = append(sublines, Subline(styleDepImpl, "%s %s", emojiInterface, ev.Impl))
			}
			sublines = append(sublines, Subline(styleDepWiring, "%s %s", emojiCaller, ev.Caller.Func))
			sublines = append(sublines, Subline(styleCodeLoc, "%s(%s:%d)", emojiCodeLocation, ev.Caller.File, ev.Caller.Line))
			sublines = append(sublines, Subline(styleTypeName, "%s <b>Dependency</b>", emojiDep))

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

			if ev.Caller.Func != "" {
				callerID, callerType := canonicalCaller(ev.Caller.Func, initializerTypes)
				style := styleCaller
				if callerType == NodeInitializer {
					style = styleInitializer
				}
				if _, ok := nodeMap[callerID]; !ok {
					nodeMap[callerID] = Node{
						ID:    callerID,
						Label: LabelBuilder{Label: callerID, FontSize: 15, Bold: true}.ToHTML(),
						Type:  callerType,
						Style: style,
					}
				}
				*edges = append(*edges, Edge{From: callerID, To: dependency, Arrow: "--o"})
			}
		}
		if ev.Kind == introspection.DepResolved {
			toCaller, callerType := canonicalCaller(ev.Caller.Func, initializerTypes)
			if toCaller == "" {
				toCaller = ev.Component
			}
			if toCaller == "" {
				toCaller = ev.Type
			}
			*edges = append(*edges, Edge{From: dependency, To: toCaller, Arrow: "-.->"})
			depHasCaller[dependency] = true

			label := LabelBuilder{
				Label:    toCaller,
				FontSize: 15,
				Bold:     true,
				SubLines: []string{
					Subline(styleCodeLoc, "%s(%s:%d)", emojiCodeLocation, ev.Caller.File, ev.Caller.Line),
				},
			}.ToHTML()

			style := styleCaller
			if callerType == NodeInitializer {
				style = styleInitializer
			}
			nodeMap[toCaller] = Node{
				ID:    toCaller,
				Label: label,
				Type:  callerType,
				Style: style,
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
}

// buildConfigGraph constructs the configuration graph from introspection data.
func buildConfigGraph(nodeMap map[string]Node, initializerTypes map[string]struct{}, configs []introspection.ConfigAccess, edges *[]Edge) {
	for _, k := range configs {
		configKey := k.Key
		var sublines []string
		if k.Provider != "" {
			sublines = append(sublines, Subline(styleConfigProvider, "%s %s", emojiConfigProvider, k.Provider))
		}
		if k.UsedDefault {
			sublines = append(sublines, Subline(styleConfigDefault, "default"))
		}
		sublines = append(sublines, Subline(styleTypeName, "%s <b>Config</b>", emojiConfig))
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

		caller, callerType := canonicalCaller(k.Caller.Func, initializerTypes)
		if caller == "" && k.Component != "" {
			caller = k.Component
			callerType = NodeCaller
		}
		if caller == "" {
			caller = "unknown caller"
			callerType = NodeCaller
		}
		*edges = append(*edges, Edge{From: configKey, To: caller, Arrow: "-.->"})
		labelCaller := LabelBuilder{
			Label:    caller,
			FontSize: 15,
			Bold:     true,
			SubLines: []string{
				Subline(styleCodeLoc, "%s(%s:%d)", emojiCodeLocation, k.Caller.File, k.Caller.Line),
			},
		}.ToHTML()
		style := styleCaller
		if callerType == NodeInitializer {
			style = styleInitializer
		}
		nodeMap[caller] = Node{
			ID:    caller,
			Label: labelCaller,
			Type:  callerType,
			Style: style,
		}
	}
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
				Subline(styleTypeName, "%s <b>Runnable</b>", emojiRunnable),
			},
		}.ToHTML()
		nodeMap[runnableID] = Node{
			ID:    runnableID,
			Label: label,
			Type:  NodeRunnable,
			Style: styleCaller,
		}
		*edges = append(*edges, Edge{From: runnableID, To: appNodeId, Arrow: "---"})
	}
}

func buildInitializerGraph(initializers []introspection.InitializerInfo, nodeMap map[string]Node) {
	for _, init := range initializers {
		initID := init.Type
		label := LabelBuilder{
			Label:    initID,
			FontSize: 16,
			Bold:     true,
			SubLines: []string{
				Subline(styleTypeName, "%s <b>Initializer</b>", emojiInitializer),
			},
		}.ToHTML()
		nodeMap[initID] = Node{
			ID:    initID,
			Label: label,
			Type:  NodeInitializer,
			Style: styleInitializer,
		}
	}
}

// canonicalCaller resolves a caller function string to either a caller ID or an initializer ID.
func canonicalCaller(caller string, initializerTypes map[string]struct{}) (string, NodeType) {
	for initType := range initializerTypes {
		base := strings.TrimPrefix(initType, "*")
		short := base
		if idx := strings.LastIndex(base, "."); idx != -1 {
			short = base[idx+1:]
		}
		if strings.Contains(caller, initType) || strings.Contains(caller, base) || strings.Contains(caller, short) {
			return initType, NodeInitializer
		}
	}
	if caller == "" {
		return "", NodeCaller
	}
	return caller, NodeCaller
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
// The order is: deps/configs -> callers/initializers -> runnables -> app.
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
