package applicationgraph

import (
	"maps"
	"slices"
	"time"

	"github.com/hmdsefi/gograph/connectivity"
	"github.com/negrel/assert"
)

type StronglyConnectedEdgesMeta struct {
	AppIdentifiers []string
	Spans          []TimeSpan
	EdgeCounts     map[Edge]int
}

const minCycleVertices = 2

func (graph *ApplicationGraph) GetStronglyConnectedApps(
	numMutualConnections int,
) map[string][]StronglyConnectedEdgesMeta {
	stronglyConnectedApps := map[string][]StronglyConnectedEdgesMeta{}

	scss := connectivity.Tarjan(graph.Graph) // stongly connected nodes
	// easiest to explain with an example:
	// user is doing FE work and they're switching between chrome, vscode and terminal
	// in one session they switch between these 3 apps multiple times
	// want to find these blocks of time
	for _, vertices := range scss {
		if len(vertices) < minCycleVertices {
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
						edgeCounts[edge]++
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
