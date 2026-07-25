package timeline

import (
	"cmp"
	"container/list"
	"context"
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

		timelineGraph.addVertex(curr)
	}
	timelineGraph.buildGraph()

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

func (graph *ApplicationGraph) addVertex(curr ForegroundProcess) (bool, bool) {
	// breaks acquiring locks at the top but for reuse
	graph.mu.Lock()
	defer graph.mu.Unlock()

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

func (graph *ApplicationGraph) buildGraph() {
	// breaks acquiring locks at the top but for reuse
	graph.mu.RLock()
	defer graph.mu.RUnlock()

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
	prevProcess := graph.activeProcess

	ok, exists := graph.addVertex(curr)
	if !ok {
		return
	}

	if graph.Graph.Size() == 0 {
		graph.buildGraph()
	}

	if exists && prevProcess != nil {
		prevIdentifier := ""
		if prevProcess.IsIdle() {
			prevIdentifier = "idle"
		} else if prevProcess.IsBrowser() && !utils.IsZero(prevProcess.Enrichments.Browser.AppIdentifier) {
			prevIdentifier = *prevProcess.Enrichments.Browser.AppIdentifier
		} else {
			assert.NotNil(prevProcess.AppIdentifier)
			prevIdentifier = *prevProcess.AppIdentifier
		}
		assert.NotZero(prevIdentifier)

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
		// these edges participate in a cycle. used for cyckle counts so that we don't double count
		edgesSet := map[Edge]struct{}{}
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
				edgesSet[edgeXY] = struct{}{}
				edgesSet[edgeYX] = struct{}{}
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
		for edge := range edgesSet {
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
	graph.mu.RLock()
	defer graph.mu.RUnlock()

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

func (graph *ApplicationGraph) GetFocusScores(numMutualConnections int) map[string]float64 {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	stronglyConnectedApps := graph.getStronglyConnectedApps(numMutualConnections)

	focusScores := map[string]float64{}

	if len(graph.vertexMetaMap) == 0 {
		return focusScores
	}

	outgoingSpans := map[string][]TimeSpan{}
	for label, meta := range graph.vertexMetaMap {
		outgoingSpans[label] = append(outgoingSpans[label], meta.Intervals...)
	}

	incomingCounts := map[string]int{}
	outgoingCounts := map[string]int{}
	for edge, meta := range graph.edgeMetaMap {
		outgoingCounts[edge[0]] += meta.IncomingCount
		incomingCounts[edge[1]] += meta.IncomingCount
	}

	appCycleSpanMap := map[string][]TimeSpan{}
	appCycleIncomingCounts := map[string]int{}
	appCycleOutgoingCounts := map[string]int{}
	for label, metas := range stronglyConnectedApps {
		for _, meta := range metas {
			appCycleSpanMap[label] = append(appCycleSpanMap[label], meta.Spans...)

			for edge, count := range meta.EdgeCounts {
				if edge[0] == label {
					appCycleOutgoingCounts[label] += count
				}

				if edge[1] == label {
					appCycleIncomingCounts[label] += count
				}
			}
		}
	}

	durations := map[string][]time.Duration{}
	// collapse spans for each app which are part of a cycle
	for label, spans := range outgoingSpans {
		blocks := appCycleSpanMap[label]

		for _, span := range spans {
			if slices.ContainsFunc(blocks, span.Between) {
				continue
			}

			durations[label] = append(durations[label], span[1].Sub(span[0]))
		}

		for _, block := range blocks {
			durations[label] = append(durations[label], block[1].Sub(block[0]))
		}
	}

	topN := utils.TopN(cmp.Compare[time.Duration], 10)

	for label, d := range durations {
		incoming := incomingCounts[label] - appCycleIncomingCounts[label] + len(appCycleSpanMap[label])
		outgoing := outgoingCounts[label] - appCycleOutgoingCounts[label] + len(appCycleSpanMap[label])

		totalEdges := incoming + outgoing
		if totalEdges <= 0 {
			continue
		}

		focusScores[label] = float64(len(topN(d))) / float64(totalEdges)
	}

	return focusScores
}
