package applicationgraph

func (graph *ApplicationGraph) GetEdgeMeta(fromLabel string, toLabel string) (*EdgeMeta, bool) {
	graph.RWMu.RLock()
	defer graph.RWMu.RUnlock()

	meta, ok := graph.edgeMetaMap[Edge{fromLabel, toLabel}]

	return meta, ok
}
