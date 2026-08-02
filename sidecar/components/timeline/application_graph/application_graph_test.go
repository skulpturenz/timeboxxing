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

// Zero values, declared rather than written as literals so the fixtures below stay readable.
var (
	appMetadataZero models.AppMetadata
	browserZero     models.Browser
	locationZero    models.Location
	enrichmentsZero models.Enrichments
)

func at(seconds int) time.Time {
	return base.Add(time.Duration(seconds) * time.Second)
}

// Distinct apps must use distinct PIDs: ForegroundProcess.IsEqual keys on PID.
func appProc(identifier string, pid int64, cat enumscategories.Category, at time.Time) ForegroundProcess {
	meta := models.AppMetadata{
		FriendlyName: "",
		Description:  "",
		Category:     cat,
		IconPath:     "",
		Source:       "",
	}

	return ForegroundProcess{
		AppName:       new(identifier),
		AppIdentifier: &identifier,
		AppPath:       new("/" + identifier),
		PID:           &pid,
		WindowTitle:   nil,
		TitleSource:   nil,
		Timestamp:     at,
		Idle:          false,
		Killed:        false,
		Enrichments:   models.Enrichments{Appmetadata: meta, Browser: browserZero, Location: locationZero},
	}
}

// Labelled "idle". IsIdle asserts every app field is nil and the timestamp is non-zero.
func idleProc(at time.Time) ForegroundProcess {
	return ForegroundProcess{
		AppName:       nil,
		AppIdentifier: nil,
		AppPath:       nil,
		PID:           nil,
		WindowTitle:   nil,
		TitleSource:   nil,
		Timestamp:     at,
		Idle:          true,
		Killed:        false,
		Enrichments:   enrichmentsZero,
	}
}

// Labelled by tabID rather than the chrome bundle id, and categorised by Browser.Category.
func browserProc(tabID string, cat enumscategories.Category, pid int64, at time.Time) ForegroundProcess {
	tab := models.Browser{
		Vendor:        "chrome",
		Category:      &cat,
		AppIdentifier: &tabID,
		Tab:           "",
		CdpURL:        "",
		Domain:        "",
	}

	return ForegroundProcess{
		AppName:       nil,
		AppIdentifier: new("com.google.Chrome"),
		AppPath:       nil,
		PID:           &pid,
		WindowTitle:   nil,
		TitleSource:   nil,
		Timestamp:     at,
		Idle:          false,
		Killed:        false,
		Enrichments:   models.Enrichments{Appmetadata: appMetadataZero, Browser: tab, Location: locationZero},
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
	return From(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40)),
	))
}
