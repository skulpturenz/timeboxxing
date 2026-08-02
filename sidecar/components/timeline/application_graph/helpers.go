package applicationgraph

import (
	"github.com/hmdsefi/gograph"
	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

const idleLabel = "idle"

func vertexLabel(process ForegroundProcess) string {
	switch {
	case process.IsIdle():
		return idleLabel
	case process.IsBrowser() && !utils.IsZero(process.Enrichments.Browser.AppIdentifier):
		return *process.Enrichments.Browser.AppIdentifier
	default:
		assert.NotNil(process.AppIdentifier)
		return *process.AppIdentifier
	}
}

func (strategy ConnectionStrategy) AddVertexMeta(
	graph *BaseGraph[*VertexMeta, *EdgeMeta],
	curr ForegroundProcess,
) (bool, bool) {
	prevProcess := graph.activeProcess

	if prevProcess != nil && curr.IsEqual(*prevProcess) {
		return false, true
	}

	appIdentifier := vertexLabel(curr)
	assert.NotZero(appIdentifier)

	vertexMeta, vertexExists := graph.vertexMetaMap[appIdentifier]
	if vertexMeta == nil {
		if curr.IsBrowser() && !utils.IsZero(curr.Enrichments.Browser.Category) {
			vertexMeta = &VertexMeta{
				Category:  *curr.Enrichments.Browser.Category,
				Duration:  0,
				Intervals: nil,
				Count:     0,
			}
		} else {
			vertexMeta = &VertexMeta{
				Category:  curr.Enrichments.Appmetadata.Category,
				Duration:  0,
				Intervals: nil,
				Count:     0,
			}
		}
		graph.vertexMetaMap[appIdentifier] = vertexMeta
	}
	assert.NotNil(graph.vertexMetaMap[appIdentifier])
	vertexMeta.Count++

	if prevProcess != nil {
		prevIdentifier := vertexLabel(*prevProcess)
		assert.NotEqual(appIdentifier, prevIdentifier)

		edge := Edge{prevIdentifier, appIdentifier}
		edgeMeta := graph.edgeMetaMap[edge]
		if edgeMeta == nil {
			edgeMeta = &EdgeMeta{
				IncomingDuration: 0,
				IncomingCount:    0,
				spans:            nil,
			}
			graph.edgeMetaMap[edge] = edgeMeta
		}
		assert.NotNil(graph.edgeMetaMap[edge])

		edgeMeta.IncomingDuration += curr.Timestamp.Sub(prevProcess.Timestamp)
		edgeMeta.IncomingCount++
		edgeMeta.spans = append(edgeMeta.spans, TimeSpan{prevProcess.Timestamp, curr.Timestamp})

		prevMeta := graph.vertexMetaMap[prevIdentifier]
		assert.NotNil(prevMeta)

		prevMeta.Duration += curr.Timestamp.Sub(prevProcess.Timestamp)
		prevMeta.Intervals = append(prevMeta.Intervals, TimeSpan{prevProcess.Timestamp, curr.Timestamp})
	}

	graph.activeProcess = &curr

	return true, vertexExists
}

func (strategy ConnectionStrategy) Build(
	graph *BaseGraph[*VertexMeta, *EdgeMeta],
) {
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

	strategy := ConnectionStrategy{}
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
	appIdentifier := vertexLabel(curr)
	assert.NotZero(appIdentifier)

	prevIdentifier := vertexLabel(prev)
	assert.NotZero(prevIdentifier)

	if graph.Graph.GetVertexByID(appIdentifier) == nil {
		currMeta := graph.vertexMetaMap[appIdentifier]
		assert.NotNil(currMeta)

		currVertex := graph.Graph.AddVertexByLabel(
			appIdentifier,
			gograph.WithVertexWeight(float64(currMeta.Duration.Milliseconds())),
		)
		assert.NotNil(currVertex)
	} // no else, the duration on a vertex is only updated when the edge is outgoing

	meta := graph.vertexMetaMap[prevIdentifier]
	assert.NotNil(meta)

	// we need to drop vertex and rebuild it to update weights
	vertex := graph.Graph.GetVertexByID(prevIdentifier)
	assert.NotNil(vertex)

	graph.Graph.RemoveEdges(graph.Graph.EdgesOf(vertex)...)
	graph.Graph.RemoveVertices(vertex)

	vertex = graph.Graph.AddVertexByLabel(
		prevIdentifier,
		gograph.WithVertexWeight(float64(meta.Duration.Milliseconds())),
	)
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
		_, err := graph.Graph.AddEdge(
			from,
			to,
			gograph.WithEdgeWeight(float64(edgeMeta.IncomingDuration.Milliseconds())),
		)
		assert.Nil(err)
	}
}
