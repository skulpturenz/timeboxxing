package applicationgraph

import (
	"cmp"
	"slices"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

func (graph *ApplicationGraph) GetFocusScores(numMutualConnections int) map[string]float64 {
	graph.RWMu.RLock()
	defer graph.RWMu.RUnlock()

	stronglyConnectedApps := graph.GetStronglyConnectedApps(numMutualConnections)

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
