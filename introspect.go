package symbiont

import (
	"context"
	"fmt"
	"reflect"

	"github.com/cleitonmarx/symbiont/config"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/internal/mermaid"
	"github.com/cleitonmarx/symbiont/internal/reflectx"
	"github.com/cleitonmarx/symbiont/introspection"
)

// AppIntrospection contains data about configuration keys accessed and dependency events
// during the application's lifecycle.
type AppIntrospection struct {
	Keys    []introspection.ConfigAccess
	Events  []depend.Event
	Runners []reflect.Type
}

// Introspector defines an interface for introspecting application runners, configuration and dependencies.
type Introspector interface {
	Introspect(context.Context, AppIntrospection) error
}

// Instrospect allows introspection of configuration keys used and dependency events
// during the application's lifecycle.
// The provided Introspector's Introspect method will be called after initialization
// and before starting runnables.
func (a *App) Instrospect(i Introspector) *App {
	a.introspector = i
	return a
}

// introspectSafe calls the provided Introspector's Introspect method safely,
// recovering from panics and wrapping errors with context about the introspector.
func introspectSafe(ctx context.Context, i Introspector, ai AppIntrospection) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = NewError(fmt.Errorf("panic in Introspect func: %v", r), i)
		}
	}()
	err = i.Introspect(ctx, ai)
	if err != nil {
		err = NewError(err, i.Introspect)
	}
	return err
}

const (
	emojiCodeLocation = "📍"
	emojiDep          = "🧩"
	emojiApp          = "🕷️"
	emojiCaller       = "🏗️"
	emojiConfig       = "🗝️"
	emojiService      = "📦"
	emojiSetup        = "🔧"
	emojiInjector     = "💉"
)

var (
	// node styles
	styleDepUsed     = mermaid.Style{Fill: "#e0f7fa", Stroke: "#00838f", StrokeWidth: "2px", Color: "#222222"}
	styleDepUnused   = mermaid.Style{Fill: "#fce1e1", Stroke: "#a60202", StrokeWidth: "2px", Color: "#b26a00"}
	styleConfig      = mermaid.Style{Fill: "#e8f5e9", Stroke: "#388e3c", StrokeWidth: "2px", Color: "#222222"}
	styleCaller      = mermaid.Style{Fill: "#fff3e0", Stroke: "#f57c00", StrokeWidth: "2px", Color: "#222222"}
	styleRunnable    = mermaid.Style{Fill: "#e3e0fc", Stroke: "#6c47a6", StrokeWidth: "2px", Color: "#222222"}
	styleApp         = mermaid.Style{Fill: "#0525f5", Stroke: "black", StrokeWidth: "3px", Color: "#ffffff", FontWeight: "bold"}
	styleInitializer = mermaid.Style{Fill: "#f0f0f0", Stroke: "#888888", StrokeWidth: "1px", Color: "#222222", FontWeight: "bold"}

	// sublines styles
	styleName           = mermaid.Style{Color: "#b26a00", FontSize: "12px", IsHtml: true}
	styleDepImpl        = mermaid.Style{Color: "darkgray", FontSize: "11px", IsHtml: true}
	styleDepWiring      = mermaid.Style{Color: "darkblue", FontSize: "11px", IsHtml: true}
	styleCodeLoc        = mermaid.Style{Color: "gray", FontSize: "11px", IsHtml: true}
	styleConfigProvider = mermaid.Style{FontSize: "11px", Color: "green", IsHtml: true}
	styleConfigDefault  = mermaid.Style{Color: "green", FontSize: "11px", IsHtml: true}
)

// GenerateIntrospectionGraph generates a Mermaid graph representation of the Spider introspection data.
// It includes services, dependencies, used configuration keys, and their caller functions.
//
// The 'nodeTypes' parameter specifies which parts to include in the graph:
//   - GraphDependencies: Include dependency graph
//   - GraphConfigKeys: Include configuration keys
func (s AppIntrospection) GenerateIntrospectionGraph() string {
	var edges []mermaid.Edge
	nodeMap := make(map[string]mermaid.Node)
	depHasCaller := make(map[string]bool)

	// --- Dependencies ---
	depEdges, _ := buildDependencyGraph(nodeMap, depHasCaller)
	edges = append(edges, depEdges...)
	// --- Configs ---
	configEdges, _ := buildConfigGraph(nodeMap)
	edges = append(edges, configEdges...)
	// --- App node ---
	appNodeID := "Symbiont"
	if _, exists := nodeMap[appNodeID]; !exists {
		appLabel := mermaid.LabelBuilder{
			Label:     fmt.Sprintf("Symbiont %s", emojiApp),
			FontSize:  20,
			FontColor: "white",
			Bold:      true,
		}.ToHTML()
		nodeMap[appNodeID] = mermaid.Node{
			ID:    appNodeID,
			Label: appLabel,
			Type:  mermaid.NodeApp,
			Style: styleApp,
		}
	}

	// --- Runnable ---
	buildRunnerGraph(s.Runners, nodeMap, &edges, appNodeID)
	// Remove duplicates and preserve order
	order := buildOrderedNodeIDs(nodeMap)
	// --- Set styles using declarative Style struct ---
	applyNodeStyles(nodeMap, depHasCaller)

	// --- Build Graph struct and render ---
	var nodes []mermaid.Node
	for _, id := range order {
		nodes = append(nodes, nodeMap[id])
	}
	g := mermaid.Graph{
		Nodes: nodes,
		Edges: edges,
	}
	return g.RenderTD()
}

// buildDependencyGraph constructs the dependency graph from Spider's introspection data.
func buildDependencyGraph(nodeMap map[string]mermaid.Node, depHasCaller map[string]bool) ([]mermaid.Edge, []string) {
	edges := []mermaid.Edge{}
	registeredOrder := []string{}
	for _, ev := range depend.GetEvents() {
		dependency := ev.Implementation + ev.DepName
		if ev.Action == depend.ActionRegister {
			var sublines []string
			if ev.DepName != "" {
				sublines = append(sublines, mermaid.Subline(styleName, "name: %s", ev.DepName))
			}
			if ev.DepTypeName != ev.Implementation {
				sublines = append(sublines, mermaid.Subline(styleDepImpl, "%s %s", emojiDep, ev.Implementation))
			}
			sublines = append(sublines, mermaid.Subline(styleDepWiring, "%s %s", emojiCaller, ev.Caller))
			sublines = append(sublines, mermaid.Subline(styleCodeLoc, "%s(%s:%d)", emojiCodeLocation, ev.File, ev.Line))

			label := mermaid.LabelBuilder{
				Label:    ev.DepTypeName,
				FontSize: 16,
				Bold:     true,
				SubLines: sublines,
			}.ToHTML()

			nodeMap[dependency] = mermaid.Node{
				ID:    dependency,
				Label: label,
				Type:  mermaid.NodeDependency,
			}
			registeredOrder = append(registeredOrder, dependency)
		}
		if ev.Action == depend.ActionResolve {
			toCaller := ev.Caller
			if toCaller == "" {
				//toCaller = ev.DepTypeName
				toCaller = reflectx.GetTypeName(ev.ComponentType)
			}
			edges = append(edges, mermaid.Edge{From: dependency, To: toCaller})
			depHasCaller[dependency] = true

			label := mermaid.LabelBuilder{
				Label:    toCaller,
				FontSize: 15,
				Bold:     true,
				SubLines: []string{
					mermaid.Subline(styleCodeLoc, "%s(%s:%d)", emojiCodeLocation, ev.File, ev.Line),
				},
			}.ToHTML()

			nodeMap[toCaller] = mermaid.Node{
				ID:    toCaller,
				Label: label,
				Type:  mermaid.NodeCaller,
			}

			// if the dependency is not already in the nodeMap, add it
			// this can happen if the dependency is registered but not resolved
			if _, exists := nodeMap[dependency]; !exists {
				var sublines []string
				if ev.DepName != "" {
					sublines = append(sublines, mermaid.Subline(styleName, "name: %s", ev.DepName))
				}
				if ev.DepTypeName != ev.Implementation {
					sublines = append(sublines, mermaid.Subline(styleDepImpl, "impl: %s", ev.Implementation))
				}
				label := mermaid.LabelBuilder{Label: ev.DepTypeName, FontSize: 16, Bold: true,
					SubLines: sublines,
				}.ToHTML()

				nodeMap[dependency] = mermaid.Node{
					ID:    dependency,
					Label: label,
					Type:  mermaid.NodeDependency,
				}
			}
		}
	}
	return edges, registeredOrder
}

// buildConfigGraph constructs the configuration graph from Spider's introspection data.
func buildConfigGraph(nodeMap map[string]mermaid.Node) ([]mermaid.Edge, []string) {
	edges := []mermaid.Edge{}
	configOrder := []string{}
	configKeys := config.IntrospectConfigAccesses()
	for _, k := range configKeys {
		configKey := k.Key
		if _, exists := nodeMap[configKey]; !exists {
			configOrder = append(configOrder, configKey)
		}
		var sublines []string
		if k.Provider != "" {
			sublines = append(sublines, mermaid.Subline(styleConfigProvider, "%s %s", emojiConfig, k.Provider))
		}
		if k.UsedDefault {
			sublines = append(sublines, mermaid.Subline(styleConfigDefault, "default"))
		}
		label := mermaid.LabelBuilder{
			Label:    k.Key,
			FontSize: 16,
			Bold:     true,
			SubLines: sublines,
		}.ToHTML()
		nodeMap[configKey] = mermaid.Node{
			ID:    configKey,
			Label: label,
			Type:  mermaid.NodeConfig,
		}

		caller := k.Caller.Func
		if caller == "" && k.Component != "" {
			caller = k.Component
		}
		if caller == "" {
			caller = "unknown caller"
		}
		edges = append(edges, mermaid.Edge{From: configKey, To: caller})
		labelCaller := mermaid.LabelBuilder{
			Label:    caller,
			FontSize: 15,
			Bold:     true,
			SubLines: []string{
				mermaid.Subline(styleCodeLoc, "%s(%s:%d)", emojiCodeLocation, k.Caller.File, k.Caller.Line),
			},
		}.ToHTML()
		nodeMap[caller] = mermaid.Node{
			ID:    caller,
			Label: labelCaller,
			Type:  mermaid.NodeCaller,
		}
	}
	return edges, configOrder
}

// buildRunnerGraph builds runnable nodes and returns their IDs in order.
func buildRunnerGraph(runnableTypes []reflect.Type, nodeMap map[string]mermaid.Node, edges *[]mermaid.Edge, appNodeId string) {
	for _, runnable := range runnableTypes {
		runnableID := reflectx.GetTypeName(runnable)
		label := mermaid.LabelBuilder{
			Label:    runnableID,
			FontSize: 16,
			Bold:     true,
			SubLines: []string{
				mermaid.Subline(styleConfigDefault, "%s Runnable", emojiService),
			},
		}.ToHTML()
		nodeMap[runnableID] = mermaid.Node{
			ID:    runnableID,
			Label: label,
			Type:  mermaid.NodeRunnable,
			Style: styleCaller,
		}
		*edges = append(*edges, mermaid.Edge{From: runnableID, To: appNodeId})
	}
}

// applyNodeStyles applies styles to nodes based on their type and whether they have callers.
func applyNodeStyles(nodeMap map[string]mermaid.Node, depHasCaller map[string]bool) {
	for id, n := range nodeMap {
		switch n.Type {
		case mermaid.NodeDependency:
			if !depHasCaller[id] {
				n.Style = styleDepUnused
			} else {
				n.Style = styleDepUsed
			}
		case mermaid.NodeConfig:
			n.Style = styleConfig
		case mermaid.NodeCaller:
			n.Style = styleCaller
		case mermaid.NodeRunnable:
			n.Style = styleRunnable
		case mermaid.NodeApp:
			n.Style = styleApp
		case mermaid.NodeInitializer:
			n.Style = styleInitializer
		}
		nodeMap[id] = n
	}
}

// correlateServicesToSetups updates caller nodes to setups and creates edges from setups to services and services to app.
func correlateServicesToSetups(runnableOrder []string, edges *[]mermaid.Edge, appNodeID string) {
	for _, runnableID := range runnableOrder {
		// Runnables point to App (downward)
		*edges = append(*edges, mermaid.Edge{From: runnableID, To: appNodeID})
	}
}

// buildOrderedNodeIDs returns a deduplicated, ordered list of node IDs for rendering.
// The order is: deps/configs -> callers -> runnables -> app.
func buildOrderedNodeIDs(nodeMap map[string]mermaid.Node) []string {
	// --- Build node order: deps/configs -> callers -> runnables -> app ---
	var depConfigOrder, callerOrder, runnableOrderOrdered []string
	var appNode string
	for id, node := range nodeMap {
		switch node.Type {
		case mermaid.NodeDependency, mermaid.NodeConfig:
			depConfigOrder = append(depConfigOrder, id)
		case mermaid.NodeCaller, mermaid.NodeInitializer:
			callerOrder = append(callerOrder, id)
		case mermaid.NodeRunnable:
			runnableOrderOrdered = append(runnableOrderOrdered, id)
		case mermaid.NodeApp:
			appNode = id
		}
	}

	// Deduplicate while preserving order
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
