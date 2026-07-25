package applicationgraph

import (
	"strings"
	"time"
)

type EntrySuggestion struct {
	AppIdentifiers []string
	Start          time.Time
	End            time.Time
}

func (graph *ApplicationGraph) GetEntrySuggestions(start time.Time, numMutualConnections int) []EntrySuggestion {
	graph.RWMu.RLock()
	defer graph.RWMu.RUnlock()

	stronglyConnectedApps := graph.GetStronglyConnectedApps(numMutualConnections)

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
