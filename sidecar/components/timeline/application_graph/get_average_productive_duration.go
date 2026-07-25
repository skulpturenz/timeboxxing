package applicationgraph

import (
	"time"

	"github.com/negrel/assert"
)

func (graph *ApplicationGraph) GetAverageProductiveDuration() time.Duration {
	graph.RWMu.RLock()
	defer graph.RWMu.RUnlock()

	spans := []TimeSpan{}

	for _, v := range graph.Graph.GetAllVertices() {
		meta := graph.vertexMetaMap[v.Label()]
		assert.NotNil(meta)

		if !meta.Category.IsProductive() {
			continue
		}

		spans = append(spans, meta.Intervals...)
	}

	if len(spans) == 0 {
		return 0 * time.Millisecond
	}

	duration := 0 * time.Millisecond
	for _, s := range spans {
		duration += s[1].Sub(s[0])
	}

	return duration / time.Duration(len(spans))
}
