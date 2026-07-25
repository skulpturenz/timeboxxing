package timeline

import (
	"cmp"
	"container/list"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hmdsefi/gograph"
	"github.com/hmdsefi/gograph/connectivity"
	"github.com/negrel/assert"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type Edge = [2]string // [from, to]

type ApplicationGraph struct {
	Graph         gograph.Graph[string]
	mu            sync.RWMutex
	vertexMetaMap map[string]*applicationGraphVertexMeta
	edgeMetaMap   map[Edge]*applicationGraphEdgeMeta
	activeProcess *ForegroundProcess
}

type TimeSpan = utils.TimeSpan

type applicationGraphVertexMeta struct {
	Category  enumscategories.Category
	Duration  time.Duration // total usage time
	Intervals []TimeSpan    // span from curr start -> curr end
	Count     int           // number of times app was used
}

type applicationGraphEdgeMeta struct {
	IncomingDuration time.Duration // time spent on A before switching to B
	IncomingCount    int           // number of A->B
	spans            []TimeSpan
}

func GraphFrom(timeline *list.List) *ApplicationGraph {
	g := gograph.New[string](gograph.Directed())

	vertexMetaMap := map[string]*applicationGraphVertexMeta{}
	edgeMetaMap := map[Edge]*applicationGraphEdgeMeta{}

	timelineGraph := ApplicationGraph{
		Graph:         g,
		vertexMetaMap: vertexMetaMap,
		edgeMetaMap:   edgeMetaMap,
	}

	for e := timeline.Front(); e != nil; e = e.Next() {
		curr, ok := e.Value.(ForegroundProcess)
		assert.True(ok)

		timelineGraph.upsertForegroundProcess(curr)
	}

	return &timelineGraph
}

func GraphChan(ctx context.Context, ch <-chan ForegroundProcess) *ApplicationGraph {
	g := gograph.New[string](gograph.Directed())

	vertexMetaMap := map[string]*applicationGraphVertexMeta{}
	edgeMetaMap := map[Edge]*applicationGraphEdgeMeta{}

	timelineGraph := ApplicationGraph{
		Graph:         g,
		vertexMetaMap: vertexMetaMap,
		edgeMetaMap:   edgeMetaMap,
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case curr := <-ch:
				timelineGraph.upsertForegroundProcess(curr)
			}
		}
	}()

	return &timelineGraph
}

func (graph *ApplicationGraph) upsertForegroundProcess(curr ForegroundProcess) {
	graph.mu.Lock()
	defer graph.mu.Unlock()

	prevProcess := graph.activeProcess

	if prevProcess != nil && curr.IsEqual(*prevProcess) {
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

		prevMeta := graph.vertexMetaMap[prevIdentifier]
		assert.NotNil(prevMeta)

		prevMeta.Duration += curr.Timestamp.Sub(prevProcess.Timestamp)
		prevMeta.Intervals = append(prevMeta.Intervals, TimeSpan{prevProcess.Timestamp, curr.Timestamp})
	}

	vertices := map[string]*gograph.Vertex[string]{}
	for v, meta := range graph.vertexMetaMap {
		if vertexExists { // if the vertex already exists, adding it again returns nil instead of updating the weights
			existing := graph.Graph.GetVertexByID(v)
			assert.NotNil(existing)

			graph.Graph.RemoveVertices(existing)
		}

		vertex := graph.Graph.AddVertexByLabel(v, gograph.WithVertexWeight(float64(meta.Duration.Milliseconds())))
		assert.NotNil(vertex)

		vertices[v] = vertex
	}

	for edge, meta := range graph.edgeMetaMap {
		from := vertices[edge[0]]
		assert.NotNil(from)

		to := vertices[edge[1]]
		assert.NotNil(to)

		// pairs well: spends a lot of time on `from` before switching `to`
		_, err := graph.Graph.AddEdge(from, to, gograph.WithEdgeWeight(float64(meta.IncomingDuration.Milliseconds())))
		assert.Condition(func() bool {
			return vertexExists || err == nil
		})
	}

	graph.activeProcess = &curr
}

func (graph *ApplicationGraph) GetVertexMeta(label string) (*applicationGraphVertexMeta, bool) {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	meta, ok := graph.vertexMetaMap[label]

	return meta, ok
}

func (graph *ApplicationGraph) GetEdgeMeta(fromLabel string, toLabel string) (*applicationGraphEdgeMeta, bool) {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	meta, ok := graph.edgeMetaMap[Edge{fromLabel, toLabel}]

	return meta, ok
}

type StronglyConnectedEdgesMeta struct {
	AppIdentifiers []string
	Spans          []TimeSpan
	EdgeCounts     map[Edge]int
}

func (graph *ApplicationGraph) getStronglyConnectedApps(numMutualConnections int) map[string][]StronglyConnectedEdgesMeta {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	stronglyConnectedApps := map[string][]StronglyConnectedEdgesMeta{}

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
		edges := []Edge{}
		for i, x := range vertices {
			for j, y := range vertices {
				if i == j {
					continue
				}

				edgeXY := Edge{x.Label(), y.Label()}
				edgeYX := Edge{y.Label(), x.Label()}

				edgeXYMeta := graph.edgeMetaMap[edgeXY]
				edgeYXMeta := graph.edgeMetaMap[edgeYX]
				if edgeXYMeta == nil || edgeYXMeta == nil {
					continue
				}

				if edgeXYMeta.IncomingCount < numMutualConnections || edgeYXMeta.IncomingCount < numMutualConnections {
					continue
				}

				label := x.Label()
				assert.NotZero(label)
				labels[label] = struct{}{}
				edges = append(edges, edgeXY, edgeYX)
			}
		}

		for label := range labels {
			meta := graph.vertexMetaMap[label]
			assert.NotNil(meta)

			intervals[label] = append(intervals[label], meta.Intervals...)
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

		// sorted so the identifiers, and any key derived from them, are stable across runs
		appIdentifiers := slices.Sorted(maps.Keys(labels))

		edgeCounts := map[Edge]int{}
		for _, edge := range edges {
			meta := graph.edgeMetaMap[edge]
			assert.NotNil(meta)

			for _, s := range meta.spans {
				for _, c := range consecutive {
					if s.Between(c) {
						edgeCounts[edge] += 1
					}
				}
			}
		}

		for a := range labels {
			meta := StronglyConnectedEdgesMeta{
				AppIdentifiers: appIdentifiers,
				Spans:          consecutive,
				EdgeCounts:     edgeCounts,
			}

			stronglyConnectedApps[a] = append(stronglyConnectedApps[a], meta)
		}
	}

	return stronglyConnectedApps
}

type EntrySuggestion struct {
	AppIdentifiers []string
	Start          time.Time
	End            time.Time
}

func (graph *ApplicationGraph) GetEntrySuggestions(start time.Time, numMutualConnections int) []EntrySuggestion {
	stronglyConnectedApps := graph.getStronglyConnectedApps(numMutualConnections)

	entries := []EntrySuggestion{}
	set := map[string]struct{}{}
	for _, v := range stronglyConnectedApps {
		for _, m := range v {
			k := strings.Join(m.AppIdentifiers, ",")
			if _, ok := set[k]; ok {
				continue
			}

			for _, s := range m.Spans {
				if s[0].Before(start) {
					continue
				}

				suggestion := EntrySuggestion{
					AppIdentifiers: m.AppIdentifiers,
					Start:          s[0],
					End:            s[1],
				}

				entries = append(entries, suggestion)
			}

			set[k] = struct{}{}
		}
	}

	return entries
}

func (graph *ApplicationGraph) GetTimeToProductive() time.Duration {
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

func (graph *ApplicationGraph) GetAverageProductiveDuration() time.Duration {
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

func (graph *ApplicationGraph) GetAverageUnproductiveDuration() time.Duration {
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

func (graph *ApplicationGraph) GetFocusScores() int {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	spans := map[*gograph.Vertex[string]][]TimeSpan{}
	incomingEdgeCounts := map[*gograph.Vertex[string]]int{}
	outgoingSpans := map[*gograph.Vertex[string]][]TimeSpan{}
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
			}
		}
	}

	stronglyConnectedApps := graph.getStronglyConnectedApps(3)
	seenEdgesSet := map[string]struct{}{}
	cycles := map[string][]TimeSpan{}
	cycleCounts := map[string]int{}
	for identifier, v := range stronglyConnectedApps {
		for _, meta := range v {
			for edge, count := range meta.EdgeCounts {
				keyAB := fmt.Sprintf("%v,%v", edge[0], edge[1])
				if _, ok := seenEdgesSet[keyAB]; ok {
					continue
				}

				keyBA := fmt.Sprintf("%v,%v", edge[1], edge[0])
				if _, ok := seenEdgesSet[keyBA]; ok {
					continue
				}

				cycles[identifier] = append(cycles[identifier], meta.Spans...)
				cycleCounts[identifier] += count
				// cycle so n(A->B) = n(B->A)
				seenEdgesSet[keyAB] = struct{}{}
				seenEdgesSet[keyBA] = struct{}{}
			}
		}
	}

	focusScores := map[*gograph.Vertex[string]]float64{}
	topN := utils.TopN(cmp.Compare[time.Duration], 10)

	outgoingDurations := map[*gograph.Vertex[string]][]time.Duration{}
	for v, d := range outgoingSpans {
		cs := cycles[v.Label()]

		for _, span := range d {
			for _, c := range cs {
				if span.Between(c) {
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
		incomingEdges := incomingEdgeCounts[v] - cycleCounts[v.Label()] + len(cycles[v.Label()])
		outgoingEdges := len(outgoingDurations[v]) - cycleCounts[v.Label()] + len(cycles[v.Label()])

		focusScores[v] = float64(nSessions) / (float64(incomingEdges + outgoingEdges))
	}

	return 0
}
