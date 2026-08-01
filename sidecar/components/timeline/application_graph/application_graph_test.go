package applicationgraph

import (
	"container/list"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
)

// base is monotonic-clock-free so interval assertions are deterministic.
var base = time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

func at(seconds int) time.Time {
	return base.Add(time.Duration(seconds) * time.Second)
}

// Distinct apps must use distinct PIDs: ForegroundProcess.IsEqual keys on PID.
func appProc(identifier string, pid int64, cat enumscategories.Category, at time.Time) ForegroundProcess {
	return ForegroundProcess{
		AppName:       new(identifier),
		AppIdentifier: &identifier,
		AppPath:       new("/" + identifier),
		PID:           &pid,
		Timestamp:     at,
		Enrichments:   models.Enrichments{Appmetadata: models.AppMetadata{Category: cat}},
	}
}

// Labelled "idle". IsIdle asserts every app field is nil and the timestamp is non-zero.
func idleProc(at time.Time) ForegroundProcess {
	return ForegroundProcess{Idle: true, Timestamp: at}
}

// Labelled by tabID rather than the chrome bundle id, and categorised by Browser.Category.
func browserProc(tabID string, cat enumscategories.Category, pid int64, at time.Time) ForegroundProcess {
	return ForegroundProcess{
		AppIdentifier: new("com.google.Chrome"),
		PID:           &pid,
		Timestamp:     at,
		Enrichments: models.Enrichments{
			Browser: models.Browser{Vendor: "chrome", AppIdentifier: &tabID, Category: &cat},
		},
	}
}

func listOf(ps ...ForegroundProcess) *list.List {
	l := list.New()
	for _, p := range ps {
		l.PushBack(p)
	}
	return l
}

// vscode<->terminal switched twice each way: both edges clear a threshold of 2 and the pair forms
// a strongly connected component.
func mutualCluster(t *testing.T) *ApplicationGraph {
	t.Helper()
	return GraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40)),
	))
}
