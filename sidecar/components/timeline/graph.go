package timeline

import (
	"cmp"
	"container/list"
	"context"
	"maps"
	"math"
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
				assert.NotZero(label)
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

func (graph *TimelineGraph) GetTimeToProductive() time.Duration {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

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

			durationsToProductive = append(durationsToProductive, toMeta.Duration)
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

func (graph *TimelineGraph) GetAverageProductiveDuration() time.Duration {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

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

func (graph *TimelineGraph) GetAverageUnproductiveDuration() time.Duration {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	spans := []TimeSpan{}

	for _, v := range graph.Graph.GetAllVertices() {
		meta := graph.vertexMetaMap[v.Label()]
		assert.NotNil(meta)

		if meta.Category.IsProductive() {
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

func (graph *TimelineGraph) GetFocusScores() int {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	// TODO: not sure if this is right
	// want to find: of the time a user spends on the app, how much of that is focused (long sessions) work?
	// lots of outgoing edges = not focused
	// what if it's part of a loop (like with GetEntrySuggestions)?
	// because there'd be a lot of outgoing edges but is focused just not on one app
	// TODO: how to get a score? it would have to be relative to other apps?
	// TODO: `Count` should be equal to `len(intervals)`. so we can weight each count based on how long the interval is
	// we have a bunch of durations. find the top x% of durations, how many sessions (nth(interval)) are there in that range?
	// number of sessions / count is percentage of focused sessions. count = len(intervals)
	// relative	to y: (percentageY - percentageX) /	percentage x
	// `GetEntrySuggestions` gives us the cycles. any spans within that range is also a focused session

	spans := map[*gograph.Vertex[string]][]TimeSpan{}
	incomingEdgeCounts := map[*gograph.Vertex[string]]int{}
	outgoingSpans := map[*gograph.Vertex[string]][]TimeSpan{}
	// outgoingDurations := map[*gograph.Vertex[string]][]time.Duration{}
	for _, v := range graph.Graph.GetAllVertices() {
		meta := graph.vertexMetaMap[v.Label()]
		assert.NotNil(meta)

		spans[v] = append(spans[v], meta.Intervals...)
	}

	if len(spans) == 0 {
		// TODO
	}

	for v, s := range spans {
		// assumption: order is preserved
		// to have an interval, there needs to be an outgoing edge
		// so if the order of edges is preserved, then span[i] is how long the app was used for before switching to outgoingEdge[i].Destination()
		for i, e := range graph.Graph.EdgesOf(v) {
			if e.Destination().Label() == v.Label() { // incoming
				incomingEdgeCounts[v] += 1 // incoming edges create an interval on the source
			} else { // outgoing
				// 0         1         2         3         4         5
				// outgoing, incoming, incoming, outgoing, incoming, outgoing
				// at 3: idx = 3 - n(incomingEdgesBefore) = 3 - 2 = 1
				//   - since len(outgoingSpans[v]) = number of outgoing edges
				idx := i - incomingEdgeCounts[v]
				outgoingSpans[v] = append(outgoingSpans[v], s[idx]) // len(outgoingSpans[v]) = number of outgoing edges
				// outgoingDurations[v] = append(outgoingDurations[v], s[idx][1].Sub(s[idx][0]))
			}

			// TODO: to consider cycles, we need to know how many outgoing edges are due to the cycle
			// and we want to consider the entire cycle time as one span
			// - we need to remove incomings during the cycle too (??)
			// - think we can collapse all intervals from cycle start - cycle end into one
			//   - number of outgoing edges after removing cycles: nOutgoingEdges - nIntervals (which are collapsed)
			//   - number of incoming edges after removing cycles: ??
			//      - nIncomingEdges - number of incoming edges from apps in cycle
			//         - number of incoming edges from apps in cycle but only within the timespan of the cycle
		}
	}

	scss := connectivity.Tarjan(graph.Graph) // stongly connected nodes
	cycles := map[*gograph.Vertex[string]][]TimeSpan{}
	cycleCounts := map[*gograph.Vertex[string]]int{}
	// TODO: refactor: most of this is from `GetEntrySuggestions`
	for _, vertices := range scss {
		if len(vertices) < 2 {
			continue
		}

		intervals := map[string][]TimeSpan{}
		labels := map[string]*gograph.Vertex[string]{}
		numMutualConnections := 3 // TODO
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
				assert.NotZero(label)
				labels[label] = x

				// TODO: new
				// cycle, so incoming = outgoing
				// ncycles = abs(((xy + yx) - (xy - yx)) / 2)
				//    - if there are cycles, then abs(outgoing) = abs(incoming). outgoing = -1 * incoming
				//    - but of the total count on an edge, only some of it are cycles
				//
				//    - xy + yx = nCycles + outgoingX + incomingX
				//    - = (nCycleOutgoingXY + nRemainderOutgoingXY) + (nCycleIncomingYX + nRemainderIncomingYX)
				//    - = (nCycleOutgoingXY + nCycleIncomingYX) + (nRemainderOutgoingXY + nRemainderIncomingYX)
				//    - = (nCycleOutgoingXY - nCycleOutgoingXY) + (nRemainderOutgoingXY + nRemainderIncomingYX)
				//    - = 0 + (nRemainderOutgoingXY + nRemainderIncomingYX)
				//    - = nRemainderOutgoingXY + nRemainderIncomingYX
				//    - noting that: nCycleOutgoingXY = -1 * nCycleIncomingYX. same magnitude, opposite direction
				//
				//    - xy - yx
				//    - = (nCycleOutgoingXY + nRemainderOutgoingXY) - (nCycleIncomingYX + nRemainderIncomingYX)
				//    - = (nCycleOutgoingXY - nCycleIncomingYX) + (nRemainderOutgoingXY - nRemainderIncomingYX)
				//    - = (nCycleOutgoingXY + nCycleOutgoingXY) + (nRemainderOutgoingXY - nRemainderIncomingYX)
				//    - = (2 * nCycleOutgoingXY) + (nRemainderOutgoingXY - nRemainderIncomingYX)
				//    - noting that: nCycleOutgoingXY = -1 * nCycleIncomingYX. same magnitude, opposite direction
				//
				//    - (xy + yx) - (xy - yx)
				//    - = (nRemainderOutgoingXY + nRemainderIncomingYX) - ((2 * nCycleOutgoingXY) + (nRemainderOutgoingXY - nRemainderIncomingYX))
				//    - = (-2 * nCycleOutgoingXY) + (nRemainderOutgoingXY - nRemainderOutgoingXY) + (nRemainderIncomingYX - nRemainderIncomingYX)
				//    - = (-2 * nCycleOutgoingXY) + 0 + 0
				//    - = -2 * nCycleOutgoingXY
				//
				//    - so: abs(((xy + yx) - (xy - yx)) / 2)
				//    - = abs(-2 * nCycleOutgoingXY / 2)
				//    - = abs(-1 * nCycleOutgoingXY)
				//    - = nCycleOutgoingXY
				// idea is that xy + yx will have 2 cycle components. xy - yx will remove the cycle component
				// so if the vertex has a cycle, then subtracting the two will remove the non cycle components and leave us with the cycle component only
				// if its zero then there are no cycles
				nCycles := math.Abs((float64(edgeXY.Count+edgeYX.Count) - float64(edgeXY.Count-edgeYX.Count)) / 2.0)
				nCyclesRound := int(math.Round(nCycles))
				assert.NotZero(nCyclesRound) // TODO: still not sure
				cycleCounts[x] += nCyclesRound
				cycleCounts[y] += nCyclesRound
			}
		}

		for label := range labels {
			meta := graph.vertexMetaMap[label]
			assert.NotNil(meta)

			filtered := []TimeSpan{}
			for _, s := range meta.Intervals {
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

		for _, v := range labels {
			cycles[v] = append(cycles[v], consecutive...)
		}
	}

	focusScores := map[*gograph.Vertex[string]]float64{}
	topN := utils.TopN(cmp.Compare[time.Duration], 10)

	outgoingDurations := map[*gograph.Vertex[string]][]time.Duration{}
	for v, d := range outgoingSpans {
		cs := cycles[v]

		for _, span := range d {
			for _, c := range cs {
				if (span[0].Equal(c[0]) || span[0].After(c[0])) && (span[1].Equal(c[1]) || span[1].Before(c[1])) {
					continue
				}

				outgoingDurations[v] = append(outgoingDurations[v], span[1].Sub(span[0]))
			}
		}
	}

	for v, d := range outgoingDurations {
		ts := topN(d)
		nSessions := len(ts)

		// cycleCounts[v] gives us the total number of incoming and outgoing edges from a cycle
		// since we're collapsing cycles, we remove all of them and
		// len(cycles[v]) gives us the number of consecutive sessions
		incomingEdges := incomingEdgeCounts[v] - cycleCounts[v] + len(cycles[v])
		outgoingEdges := len(outgoingDurations[v]) - cycleCounts[v] + len(cycles[v])

		focusScores[v] = float64(nSessions) / (float64(incomingEdges + outgoingEdges))
	}

	return 0
}
