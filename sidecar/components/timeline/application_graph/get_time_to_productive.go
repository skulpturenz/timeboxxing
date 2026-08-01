package applicationgraph

import (
	"time"

	"github.com/negrel/assert"
)

func (graph *ApplicationGraph) GetTimeToProductive() time.Duration {
	graph.RWMu.RLock()
	defer graph.RWMu.RUnlock()

	durationsToProductive := []time.Duration{}

	for _, v := range graph.Graph.GetAllVertices() {
		vertexMeta := graph.vertexMetaMap[v.Label()]
		assert.NotNil(vertexMeta)

		if vertexMeta.Category.IsProductive() {
			continue
		}

		edges := graph.Graph.EdgesOf(v)
		if len(edges) == 0 {
			continue
		}

		for _, e := range edges {
			to := e.Destination()
			assert.NotNil(to)

			toMeta := graph.vertexMetaMap[to.Label()]
			assert.NotNil(toMeta)

			if !toMeta.Category.IsProductive() {
				continue
			}

			edge := graph.edgeMetaMap[Edge{e.Source().Label(), e.Destination().Label()}]

			durationsToProductive = append(durationsToProductive, edge.IncomingDuration)
		}
	}

	if len(durationsToProductive) == 0 {
		return 0 * time.Millisecond // always productive!
	}

	total := 0 * time.Millisecond
	for _, t := range durationsToProductive {
		total += t
	}

	return total / time.Duration(len(durationsToProductive))
}
