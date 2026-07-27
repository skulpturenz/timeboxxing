package applicationgraph

func (graph *ApplicationGraph) GetVertexMeta(label string) (*applicationGraphVertexMeta, bool) {
	graph.RWMu.RLock()
	defer graph.RWMu.RUnlock()

	meta, ok := graph.vertexMetaMap[label]

	return meta, ok
}
