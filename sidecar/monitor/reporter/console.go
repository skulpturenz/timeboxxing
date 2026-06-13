package reporter

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/session"
)

const (
	colWidth  = 50
	lineWidth = 70
)

// Console reads Transitions from the manager and writes human-readable output.
type Console struct {
	out     io.Writer
	verbose bool
	started time.Time
}

// New creates a Console reporter writing to out.
func New(out io.Writer, verbose bool) *Console {
	return &Console{out: out, verbose: verbose, started: time.Now()}
}

// Run drains the transitions channel, printing each transition.
// Returns when the channel is closed (manager shutdown).
func (c *Console) Run(transitions <-chan session.Transition) {
	for t := range transitions {
		c.printTransition(t)
	}
}

func (c *Console) printTransition(t session.Transition) {
	ts := time.Now().Format("15:04:05")

	switch {
	case t.From == nil:
		// First session.
		fmt.Fprintf(c.out, "%s  START  → %s\n", ts, t.To.Key.DisplayName())

	case t.To == nil:
		// Shutdown: print duration of the last session.
		fmt.Fprintf(c.out, "%s  %s  ← END  [%s]\n",
			ts, fmtDuration(t.From.Duration), t.From.Key.DisplayName())

	default:
		fromName := t.From.Key.DisplayName()
		toName := t.To.Key.DisplayName()
		dur := fmtDuration(t.From.Duration)
		reason := string(t.Reason)

		// Show CDP URL if available.
		if t.To.Key.CDPURL != "" {
			toName = fmt.Sprintf("%s  (%s)", toName, t.To.Key.CDPURL)
		}

		fmt.Fprintf(c.out, "%s  %s  %-*s → %s  [%s]\n",
			ts, dur, colWidth, fromName, toName, reason)
	}
}

// PrintSummary prints the ranked summary table to out using the provided stats and history.
func PrintSummary(out io.Writer, stats map[session.AppKey]time.Duration, history []*session.Session, started time.Time) {
	if len(stats) == 0 {
		fmt.Fprintln(out, "\nNo sessions recorded.")
		return
	}

	// Total tracked wall-clock time.
	var total time.Duration
	for _, d := range stats {
		total += d
	}

	end := time.Now()
	wall := end.Sub(started)

	fmt.Fprintln(out)
	fmt.Fprintln(out, strings.Repeat("═", lineWidth))
	fmt.Fprintf(out, "SESSION SUMMARY   %s - %s   wall: %s   tracked: %s\n",
		started.Format("15:04:05"), end.Format("15:04:05"),
		fmtDuration(wall), fmtDuration(total))
	fmt.Fprintln(out, strings.Repeat("═", lineWidth))

	// Sort entries by duration descending.
	type entry struct {
		key session.AppKey
		dur time.Duration
	}
	entries := make([]entry, 0, len(stats))
	for k, d := range stats {
		entries = append(entries, entry{k, d})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].dur > entries[j].dur
	})

	fmt.Fprintf(out, "%-4s  %-10s  %-6s  %s\n", "Rank", "Duration", "Pct", "Application / Tab")
	fmt.Fprintln(out, strings.Repeat("─", lineWidth))

	for i, e := range entries {
		pct := 0.0
		if total > 0 {
			pct = float64(e.dur) / float64(total) * 100
		}
		fmt.Fprintf(out, "  %2d  %-10s  %5.1f%%  %s\n",
			i+1, fmtDuration(e.dur), pct, e.key.DisplayName())
	}

	fmt.Fprintln(out, strings.Repeat("─", lineWidth))
	fmt.Fprintf(out, "       %-10s  100.0%%  TOTAL\n", fmtDuration(total))

	// Application aggregate (collapsing tabs).
	appTotals := make(map[string]time.Duration)
	for _, e := range entries {
		appTotals[e.key.AppName] += e.dur
	}
	if len(appTotals) > 1 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "BY APPLICATION")
		fmt.Fprintln(out, strings.Repeat("─", lineWidth))

		type appEntry struct {
			name string
			dur  time.Duration
		}
		apps := make([]appEntry, 0, len(appTotals))
		for n, d := range appTotals {
			apps = append(apps, appEntry{n, d})
		}
		sort.Slice(apps, func(i, j int) bool { return apps[i].dur > apps[j].dur })

		for _, a := range apps {
			pct := 0.0
			if total > 0 {
				pct = float64(a.dur) / float64(total) * 100
			}
			name := a.name
			if name == "" {
				name = "[Idle]"
			}
			fmt.Fprintf(out, "  %-*s  %s  %5.1f%%\n",
				colWidth, name, fmtDuration(a.dur), pct)
		}
		fmt.Fprintln(out, strings.Repeat("─", lineWidth))
	}
}

// fmtDuration formats a duration as H:MM:SS.
func fmtDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}
