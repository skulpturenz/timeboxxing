package applicationgraph

func (graph *ApplicationGraph) GetEdgeMeta(fromLabel string, toLabel string) (*applicationGraphEdgeMeta, bool) {
	graph.RWMu.RLock()
	defer graph.RWMu.RUnlock()

	meta, ok := graph.edgeMetaMap[Edge{fromLabel, toLabel}]

	return meta, ok
}
