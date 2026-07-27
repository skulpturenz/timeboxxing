package applicationgraph

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/hmdsefi/gograph"
	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type Edge = [2]string // [from, to]
type ForegroundProcess = models.ForegroundProcess

type ApplicationGraph struct {
	Graph         gograph.Graph[string]
	RWMu          sync.RWMutex
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
	timelineGraph.RWMu.Lock()
	defer timelineGraph.RWMu.Unlock()

	for e := timeline.Front(); e != nil; e = e.Next() {
		curr, ok := e.Value.(ForegroundProcess)
		assert.True(ok)

		timelineGraph.addVertexMeta(curr)
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
