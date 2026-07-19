package timeline

import (
	"container/list"
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/hmdsefi/gograph"
	"github.com/hmdsefi/gograph/connectivity"
	"github.com/negrel/assert"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type Edge = [2]string // [from, to]

type TimelineGraph struct {
	Graph         gograph.Graph[string]
	mu            sync.RWMutex
	vertexMetaMap map[string]*VertexMeta
	edgeMetaMap   map[Edge]*EdgeMeta
}

type TimeSpan = [2]time.Time

type VertexMeta struct {
	Category  enumscategories.Category
	Duration  time.Duration
	Intervals []TimeSpan // span from curr start -> curr end
	Count     int
}

type EdgeMeta struct {
	Duration time.Duration
	Count    int
}

func GraphFrom(timeline *list.List) *TimelineGraph {
	g := gograph.New[string](gograph.Directed())

	vertexMetaMap := map[string]*VertexMeta{}
	edgeMetaMap := map[Edge]*EdgeMeta{}

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
		vertexMeta.Count += 1

		if prev != nil {
			prevProcess, ok := prev.Value.(ForegroundProcess)
			assert.True(ok)

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
			edgeMeta := edgeMetaMap[edge]
			if edgeMeta == nil {
				edgeMeta = &EdgeMeta{}
				edgeMetaMap[edge] = edgeMeta
			}
			assert.NotNil(edgeMetaMap[edge])

			edgeMeta.Duration += curr.Timestamp.Sub(prevProcess.Timestamp)
			edgeMeta.Count += 1
		}

		if next != nil {
			nextProcess, ok := next.Value.(ForegroundProcess)
			assert.True(ok)

			if !nextProcess.IsEqual(curr) {
				vertexMeta.Duration += nextProcess.Timestamp.Sub(curr.Timestamp)
				vertexMeta.Intervals = append(vertexMeta.Intervals, TimeSpan{curr.Timestamp, nextProcess.Timestamp})
			}
		}
	}

	vertices := map[string]*gograph.Vertex[string]{}
	for v, meta := range vertexMetaMap {
		vertex := g.AddVertexByLabel(v, gograph.WithVertexWeight(float64(meta.Duration.Milliseconds())))

		assert.NotNil(vertex)
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

	timelineGraph := TimelineGraph{
		Graph:         g,
		vertexMetaMap: vertexMetaMap,
		edgeMetaMap:   edgeMetaMap,
	}

	return &timelineGraph
}

func GraphChan(ctx context.Context, ch <-chan ForegroundProcess) *TimelineGraph {
	g := gograph.New[string](gograph.Directed())

	vertexMetaMap := map[string]*VertexMeta{}
	edgeMetaMap := map[Edge]*EdgeMeta{}

	timelineGraph := TimelineGraph{
		Graph:         g,
		vertexMetaMap: vertexMetaMap,
		edgeMetaMap:   edgeMetaMap,
	}

	go func() {
		var prev *ForegroundProcess
		for {
			select {
			case <-ctx.Done():
				return
			case curr := <-ch:
				func() {
					timelineGraph.mu.Lock()
					defer timelineGraph.mu.Unlock()

					if prev != nil && curr.IsEqual(*prev) {
						return
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
					vertexMeta.Count += 1

					if prev != nil {
						prevProcess := *prev

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
						edgeMeta := edgeMetaMap[edge]
						if edgeMeta == nil {
							edgeMeta = &EdgeMeta{}
							edgeMetaMap[edge] = edgeMeta
						}
						assert.NotNil(edgeMetaMap[edge])

						edgeMeta.Duration += curr.Timestamp.Sub(prevProcess.Timestamp)
						edgeMeta.Count += 1

						prevMeta := vertexMetaMap[prevIdentifier]
						assert.NotNil(prevMeta)

						prevMeta.Duration += curr.Timestamp.Sub(prev.Timestamp)
						prevMeta.Intervals = append(prevMeta.Intervals, TimeSpan{prev.Timestamp, curr.Timestamp})
					}

					for v, meta := range vertexMetaMap {
						if g.GetVertexByID(v) == nil {
							vertex := g.AddVertexByLabel(v, gograph.WithVertexWeight(float64(meta.Duration.Milliseconds())))

							assert.NotNil(vertex)
						}
					}

					for edge, meta := range edgeMetaMap {
						from := g.GetVertexByID(edge[0])
						assert.NotNil(from)

						to := g.GetVertexByID(edge[1])
						assert.NotNil(to)

						if !g.ContainsEdge(from, to) {
							_, err := g.AddEdge(from, to, gograph.WithEdgeWeight(float64(meta.Duration.Milliseconds())))
							assert.Nil(err)
						}
					}

					prev = &curr
				}()

			}
		}
	}()

	return &timelineGraph
}

func (graph *TimelineGraph) GetVertexMeta(label string) (*VertexMeta, bool) {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	meta, ok := graph.vertexMetaMap[label]

	return meta, ok
}

func (graph *TimelineGraph) GetEdgeMeta(fromLabel string, toLabel string) (*EdgeMeta, bool) {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	meta, ok := graph.edgeMetaMap[Edge{fromLabel, toLabel}]

	return meta, ok
}

type EntrySuggestion struct {
	AppIdentifiers []string
	Start          time.Time
	End            time.Time
}

func (graph *TimelineGraph) GetEntrySuggestions(start time.Time, numMutualConnections int) []EntrySuggestion {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	entries := []EntrySuggestion{}

	scss := connectivity.Tarjan(graph.Graph) // stongly connected nodes
	// easiest to explain with an example:
	// user is doing FE work and they're switching between chrome, vscode and terminal
	// in one session they switch between these 3 apps multiple times
	// want to find these blocks of time
	for _, vertices := range scss {
		if len(vertices) < 2 {
			continue
		}

		intervals := map[string][]TimeSpan{}
		labels := map[string]struct{}{}
		for i, x := range vertices {
			for j, y := range vertices {
				if i == j {
					continue
				}

				edgeXY := graph.edgeMetaMap[Edge{x.Label(), y.Label()}]
				edgeYX := graph.edgeMetaMap[Edge{y.Label(), x.Label()}]
				if edgeXY == nil || edgeYX == nil {
					continue
				}

				if edgeXY.Count < numMutualConnections || edgeYX.Count < numMutualConnections {
					continue
				}

				label := x.Label()
				labels[label] = struct{}{}
			}
		}

		for label := range labels {
			meta := graph.vertexMetaMap[label]
			assert.NotNil(meta)

			filtered := []TimeSpan{}
			for _, s := range meta.Intervals {
				if !s[0].After(start) {
					continue
				}

				filtered = append(filtered, s)
			}

			intervals[label] = append(intervals[label], filtered...)
		}

		flattened := []TimeSpan{}
		for _, v := range intervals {
			flattened = append(flattened, v...)
		}

		if len(flattened) == 0 {
			continue
		}

		slices.SortFunc(flattened, func(x TimeSpan, y TimeSpan) int {
			return x[0].Compare(y[0])
		})

		// flattened has a list of intervals sorted in ascending order of start
		// we want to find consecutive blocks of time
		// in these blocks of times, the set of apps are strongly connected
		consecutive := []TimeSpan{flattened[0]}
		for _, curr := range flattened[1:] {
			prev := &consecutive[len(consecutive)-1]

			// without the gap check everything gets merged into one consecutive time
			// because next will always be after
			if curr[0].Sub(prev[1]) <= 1*time.Minute {
				if curr[1].After(prev[1]) {
					prev[1] = curr[1]
				}
			} else {
				consecutive = append(consecutive, curr)
			}
		}

		suggestions := []EntrySuggestion{}
		for _, i := range consecutive {
			suggestions = append(suggestions, EntrySuggestion{
				AppIdentifiers: slices.Collect(maps.Keys(labels)),
				Start:          i[0],
				End:            i[1],
			})
		}

		entries = append(entries, suggestions...)
	}

	return entries
}
