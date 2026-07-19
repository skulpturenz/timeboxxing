package timeline

import (
	"container/list"
	"context"
	"time"

	"github.com/hmdsefi/gograph"
	"github.com/negrel/assert"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type TimelineGraph struct {
	Graph         gograph.Graph[string]
	vertexMetaMap map[string]*VertexMeta
	edgeMetaMap   map[[2]string]*EdgeMeta
}

type VertexMeta struct {
	Category enumscategories.Category
	Duration time.Duration
}

type EdgeMeta struct {
	Duration time.Duration
}

func GraphFrom(timeline *list.List) TimelineGraph {
	g := gograph.New[string](gograph.Directed())

	vertexMetaMap := map[string]*VertexMeta{}
	edgeMetaMap := map[[2]string]*EdgeMeta{}

	for e := timeline.Front(); e != nil; e = e.Next() {
		prev := e.Prev()
		next := e.Next()

		curr, ok := e.Value.(ForegroundProcess)
		assert.True(ok)

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

		vertexMeta := vertexMetaMap[appIdentifier]
		if vertexMeta == nil {
			if curr.IsBrowser() && !utils.IsZero(curr.Enrichments.Browser.Category) {
				vertexMeta = &VertexMeta{
					Category: *curr.Enrichments.Browser.Category,
				}
			} else {
				vertexMeta = &VertexMeta{
					Category: curr.Enrichments.Appmetadata.Category,
				}
			}
			vertexMetaMap[appIdentifier] = vertexMeta
		}
		assert.NotNil(vertexMetaMap[appIdentifier])

		if prev != nil {
			prevProcess, ok := prev.Value.(ForegroundProcess)
			assert.True(ok)

			prevIdentifier := ""
			if prevProcess.IsIdle() {
				prevIdentifier = "idle"
			} else if prevProcess.IsBrowser() && !utils.IsZero(curr.Enrichments.Browser.AppIdentifier) {
				prevIdentifier = *curr.Enrichments.Browser.AppIdentifier
			} else {
				assert.NotNil(prevProcess.Idle)
				prevIdentifier = *prevProcess.AppIdentifier
			}
			assert.NotEqual(appIdentifier, prevIdentifier)

			edge := [2]string{prevIdentifier, appIdentifier}
			edgeMeta := edgeMetaMap[edge]
			if edgeMeta == nil {
				edgeMeta = &EdgeMeta{}
				edgeMetaMap[edge] = edgeMeta
			}
			assert.NotNil(edgeMetaMap[edge])

			edgeMeta.Duration += curr.Timestamp.Sub(prevProcess.Timestamp)
		}

		if next != nil {
			nextProcess, ok := next.Value.(ForegroundProcess)
			assert.True(ok)

			vertexMeta.Duration += nextProcess.Timestamp.Sub(curr.Timestamp)
		}
	}

	vertices := map[string]*gograph.Vertex[string]{}
	for v, meta := range vertexMetaMap {
		vertex := g.AddVertexByLabel(v, gograph.WithVertexWeight(float64(meta.Duration.Milliseconds())))

		vertices[v] = vertex
	}

	for edge, meta := range edgeMetaMap {
		from := vertices[edge[0]]
		assert.NotNil(from)

		to := vertices[edge[1]]
		assert.NotNil(to)

		_, err := g.AddEdge(from, to, gograph.WithEdgeWeight(float64(meta.Duration.Milliseconds())))
		assert.Nil(err)
	}

	return TimelineGraph{
		Graph:         g,
		vertexMetaMap: vertexMetaMap,
		edgeMetaMap:   edgeMetaMap,
	}
}

func GraphChan(ctx context.Context, ch <-chan ForegroundProcess) TimelineGraph {
	g := gograph.New[string](gograph.Directed())

	vertexMetaMap := map[string]*VertexMeta{}
	edgeMetaMap := map[[2]string]*EdgeMeta{}

	go func() {
		var prev *ForegroundProcess
		for {
			select {
			case <-ctx.Done():
				return
			case curr := <-ch:
				if prev != nil && curr.IsEqual(*prev) {
					continue
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

				vertexMeta := vertexMetaMap[appIdentifier]
				if vertexMeta == nil {
					if curr.IsBrowser() && !utils.IsZero(curr.Enrichments.Browser.Category) {
						vertexMeta = &VertexMeta{
							Category: *curr.Enrichments.Browser.Category,
						}
					} else {
						vertexMeta = &VertexMeta{
							Category: curr.Enrichments.Appmetadata.Category,
						}
					}
					vertexMetaMap[appIdentifier] = vertexMeta
				}
				assert.NotNil(vertexMetaMap[appIdentifier])

				if prev != nil {
					prevProcess := *prev

					prevIdentifier := ""
					if prevProcess.IsIdle() {
						prevIdentifier = "idle"
					} else if prevProcess.IsBrowser() && !utils.IsZero(curr.Enrichments.Browser.AppIdentifier) {
						prevIdentifier = *curr.Enrichments.Browser.AppIdentifier
					} else {
						assert.NotNil(prevProcess.Idle)
						prevIdentifier = *prevProcess.AppIdentifier
					}
					assert.NotEqual(appIdentifier, prevIdentifier)

					edge := [2]string{prevIdentifier, appIdentifier}
					edgeMeta := edgeMetaMap[edge]
					if edgeMeta == nil {
						edgeMeta = &EdgeMeta{}
						edgeMetaMap[edge] = edgeMeta
					}
					assert.NotNil(edgeMetaMap[edge])

					edgeMeta.Duration += curr.Timestamp.Sub(prevProcess.Timestamp)

					prevMeta := vertexMetaMap[prevIdentifier]
					assert.NotNil(prevMeta)

					prevMeta.Duration += curr.Timestamp.Sub(prev.Timestamp)
				}

				prev = &curr
			}
		}
	}()

	return TimelineGraph{
		Graph:         g,
		vertexMetaMap: vertexMetaMap,
		edgeMetaMap:   edgeMetaMap,
	}
}

func (graph TimelineGraph) GetVertexMeta(label string) (*VertexMeta, bool) {
	meta, ok := graph.vertexMetaMap[label]

	return meta, ok
}

func (graph TimelineGraph) GetEdgeMeta(fromLabel string, toLabel string) (*EdgeMeta, bool) {
	meta, ok := graph.edgeMetaMap[[2]string{fromLabel, toLabel}]

	return meta, ok
}
