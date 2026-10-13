package main

import "devdeck/internal/knowgraph"

// KnowGraphInfo reports whether the project has an AI knowledge graph
// (e.g. .code-review-graph) and returns summary stats when the DB is readable.
func (a *App) KnowGraphInfo(projectRoot string) (knowgraph.Info, error) {
	return knowgraph.Detect(projectRoot)
}

// KnowGraphSearch finds nodes by full-text or name match.
func (a *App) KnowGraphSearch(projectRoot, query string, limit int) ([]knowgraph.SearchHit, error) {
	return knowgraph.Search(projectRoot, query, limit)
}

// KnowGraphNodes returns a page of nodes, optionally filtered by kind.
func (a *App) KnowGraphNodes(projectRoot, kind string, limit, offset int) ([]knowgraph.Node, error) {
	return knowgraph.ListNodes(projectRoot, kind, limit, offset)
}

// KnowGraphNode loads one node by qualified name plus its edges.
func (a *App) KnowGraphNode(projectRoot, qualifiedName string) (knowgraph.NodeDetail, error) {
	return knowgraph.GetNode(projectRoot, qualifiedName)
}

// KnowGraphCommunities lists community clusters in the graph.
func (a *App) KnowGraphCommunities(projectRoot string) ([]knowgraph.Community, error) {
	return knowgraph.ListCommunities(projectRoot)
}

// KnowGraphFlows lists execution flows ordered by criticality.
func (a *App) KnowGraphFlows(projectRoot string, limit int) ([]knowgraph.Flow, error) {
	return knowgraph.ListFlows(projectRoot, limit)
}
