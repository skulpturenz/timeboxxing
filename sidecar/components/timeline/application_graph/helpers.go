package applicationgraph

import (
	"github.com/hmdsefi/gograph"
	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

func (strategy ApplicationGraphConnectionStrategy) AddVertexMeta(graph *BaseGraph[*applicationGraphVertexMeta, *applicationGraphEdgeMeta], curr ForegroundProcess) (bool, bool) {
	prevProcess := graph.activeProcess

	if prevProcess != nil && curr.IsEqual(*prevProcess) {
		return false, true
	}

	appIdentifier := ""
	if curr.IsIdle() {
		appIdentifier = "idle"
	} else if curr.IsBrowser() && !utils.IsZero(curr.Enrichments.Browser.AppIdentifier) {
		appIdentifier = *curr.Enrichments.Browser.AppIdentifier
	} else {
		assert.NotNil(curr.AppIdentifier)
		appIdentifier = *curr.AppIdentifier
	}
	assert.NotZero(appIdentifier)

	vertexMeta, vertexExists := graph.vertexMetaMap[appIdentifier]
	if vertexMeta == nil {
		if curr.IsBrowser() && !utils.IsZero(curr.Enrichments.Browser.Category) {
			vertexMeta = &applicationGraphVertexMeta{
				Category: *curr.Enrichments.Browser.Category,
			}
		} else {
			vertexMeta = &applicationGraphVertexMeta{
				Category: curr.Enrichments.Appmetadata.Category,
			}
		}
		graph.vertexMetaMap[appIdentifier] = vertexMeta
	}
	assert.NotNil(graph.vertexMetaMap[appIdentifier])
	vertexMeta.Count += 1

	if prevProcess != nil {
		prevIdentifier := ""
		if prevProcess.IsIdle() {
			prevIdentifier = "idle"
		} else if prevProcess.IsBrowser() && !utils.IsZero(prevProcess.Enrichments.Browser.AppIdentifier) {
			prevIdentifier = *prevProcess.Enrichments.Browser.AppIdentifier
		} else {
			assert.NotNil(prevProcess.AppIdentifier)
			prevIdentifier = *prevProcess.AppIdentifier
		}
		assert.NotEqual(appIdentifier, prevIdentifier)

		edge := Edge{prevIdentifier, appIdentifier}
		edgeMeta := graph.edgeMetaMap[edge]
		if edgeMeta == nil {
			edgeMeta = &applicationGraphEdgeMeta{}
			graph.edgeMetaMap[edge] = edgeMeta
		}
		assert.NotNil(graph.edgeMetaMap[edge])

		edgeMeta.IncomingDuration += curr.Timestamp.Sub(prevProcess.Timestamp)
		edgeMeta.IncomingCount += 1
		edgeMeta.spans = append(edgeMeta.spans, TimeSpan{prevProcess.Timestamp, curr.Timestamp})

		prevMeta := graph.vertexMetaMap[prevIdentifier]
		assert.NotNil(prevMeta)

		prevMeta.Duration += curr.Timestamp.Sub(prevProcess.Timestamp)
		prevMeta.Intervals = append(prevMeta.Intervals, TimeSpan{prevProcess.Timestamp, curr.Timestamp})
	}

	graph.activeProcess = &curr

	return true, vertexExists
}

func (strategy ApplicationGraphConnectionStrategy) Build(graph *BaseGraph[*applicationGraphVertexMeta, *applicationGraphEdgeMeta]) {
	graph.Graph = gograph.New[string](gograph.Directed())

	for v, m := range graph.vertexMetaMap {
		vertex := graph.Graph.AddVertexByLabel(v, gograph.WithVertexWeight(float64(m.Duration.Milliseconds())))
		assert.NotNil(vertex)
	}

	for e, m := range graph.edgeMetaMap {
		from := graph.Graph.GetVertexByID(e[0])
		assert.NotNil(from)

		to := graph.Graph.GetVertexByID(e[1])
		assert.NotNil(to)

		// pairs well: spends a lot of time on `from` before switching `to`
		_, err := graph.Graph.AddEdge(from, to, gograph.WithEdgeWeight(float64(m.IncomingDuration.Milliseconds())))
		assert.Nil(err)
	}
}

func (graph *ApplicationGraph) upsertForegroundProcess(curr ForegroundProcess) {
	graph.RWMu.Lock()
	defer graph.RWMu.Unlock()

	strategy := ApplicationGraphConnectionStrategy{}
	prevProcess := graph.activeProcess

	ok, _ := strategy.AddVertexMeta(graph.BaseGraph, curr)
	if !ok {
		return
	}

	if graph.Graph.Order() == 0 {
		strategy.Build(graph.BaseGraph)
	} else if prevProcess != nil {
		graph.addVertexAndEdge(*prevProcess, curr)
	}
}

func (graph *ApplicationGraph) addVertexAndEdge(prev ForegroundProcess, curr ForegroundProcess) {
	appIdentifier := ""
	if curr.IsIdle() {
		appIdentifier = "idle"
	} else if curr.IsBrowser() && !utils.IsZero(curr.Enrichments.Browser.AppIdentifier) {
		appIdentifier = *curr.Enrichments.Browser.AppIdentifier
	} else {
		assert.NotNil(curr.AppIdentifier)
		appIdentifier = *curr.AppIdentifier
	}
	assert.NotZero(appIdentifier)

	prevIdentifier := ""
	if prev.IsIdle() {
		prevIdentifier = "idle"
	} else if prev.IsBrowser() && !utils.IsZero(prev.Enrichments.Browser.AppIdentifier) {
		prevIdentifier = *prev.Enrichments.Browser.AppIdentifier
	} else {
		assert.NotNil(prev.AppIdentifier)
		prevIdentifier = *prev.AppIdentifier
	}
	assert.NotZero(prevIdentifier)

	if graph.Graph.GetVertexByID(appIdentifier) == nil {
		currMeta := graph.vertexMetaMap[appIdentifier]
		assert.NotNil(currMeta)

		currVertex := graph.Graph.AddVertexByLabel(appIdentifier, gograph.WithVertexWeight(float64(currMeta.Duration.Milliseconds())))
		assert.NotNil(currVertex)
	} // no else, the duration on a vertex is only updated when the edge is outgoing

	meta := graph.vertexMetaMap[prevIdentifier]
	assert.NotNil(meta)

	// we need to drop vertex and rebuild it to update weights
	vertex := graph.Graph.GetVertexByID(prevIdentifier)
	assert.NotNil(vertex)

	graph.Graph.RemoveEdges(graph.Graph.EdgesOf(vertex)...)
	graph.Graph.RemoveVertices(vertex)

	vertex = graph.Graph.AddVertexByLabel(prevIdentifier, gograph.WithVertexWeight(float64(meta.Duration.Milliseconds())))
	assert.NotNil(vertex)

	for edge, edgeMeta := range graph.edgeMetaMap {
		if edge[0] != prevIdentifier && edge[1] != prevIdentifier {
			continue
		}

		from := graph.Graph.GetVertexByID(edge[0])
		assert.NotNil(from)

		to := graph.Graph.GetVertexByID(edge[1])
		assert.NotNil(to)

		// pairs well: spends a lot of time on `from` before switching `to`
		_, err := graph.Graph.AddEdge(from, to, gograph.WithEdgeWeight(float64(edgeMeta.IncomingDuration.Milliseconds())))
		assert.Nil(err)
	}
}
