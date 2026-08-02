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

type (
	Edge              = [2]string // [from, to]
	ForegroundProcess = models.ForegroundProcess
)

type BaseGraph[V, E any] struct {
	RWMu          sync.RWMutex
	Graph         gograph.Graph[string]
	activeProcess *ForegroundProcess
	vertexMetaMap map[string]V
	edgeMetaMap   map[Edge]E
}

type ApplicationGraph struct {
	*BaseGraph[*VertexMeta, *EdgeMeta]
}

type TimeSpan = utils.TimeSpan

type VertexMeta struct {
	Category  enumscategories.Category
	Duration  time.Duration // total usage time
	Intervals []TimeSpan    // span from curr start -> curr end
	Count     int           // number of times app was used
}

type EdgeMeta struct {
	IncomingDuration time.Duration // time spent on A before switching to B
	IncomingCount    int           // number of A->B
	spans            []TimeSpan
}

type GraphConnectionStrategy[V, E any] interface {
	AddVertexMeta(graph *BaseGraph[V, E], curr ForegroundProcess) (bool, bool)
	Build(graph *BaseGraph[V, E])
}

type ConnectionStrategy struct{}

var _ GraphConnectionStrategy[*VertexMeta, *EdgeMeta] = ConnectionStrategy{}

func NewBaseGraph[V, E any]() *BaseGraph[V, E] {
	return &BaseGraph[V, E]{
		RWMu:          sync.RWMutex{},
		Graph:         nil,
		activeProcess: nil,
		vertexMetaMap: map[string]V{},
		edgeMetaMap:   map[Edge]E{},
	}
}

func GraphFrom[V, E any](strategy GraphConnectionStrategy[V, E], timeline *list.List) *BaseGraph[V, E] {
	timelineGraph := NewBaseGraph[V, E]()

	timelineGraph.RWMu.Lock()
	defer timelineGraph.RWMu.Unlock()

	for e := timeline.Front(); e != nil; e = e.Next() {
		curr, ok := e.Value.(ForegroundProcess)
		assert.True(ok)

		strategy.AddVertexMeta(timelineGraph, curr)
	}
	strategy.Build(timelineGraph)

	return timelineGraph
}

func From(timeline *list.List) *ApplicationGraph {
	return &ApplicationGraph{
		BaseGraph: GraphFrom(ConnectionStrategy{}, timeline),
	}
}

func Chan(ctx context.Context, ch <-chan ForegroundProcess) *ApplicationGraph {
	strategy := ConnectionStrategy{}

	timelineGraph := &ApplicationGraph{
		BaseGraph: NewBaseGraph[*VertexMeta, *EdgeMeta](),
	}
	strategy.Build(timelineGraph.BaseGraph)

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

	return timelineGraph
}
