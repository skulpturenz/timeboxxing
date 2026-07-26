package timeline

import (
	"github.com/negrel/assert"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

func mapGetUnenrichedForegroundProcessesRowForegroundProcess(row readqueries.GetUnenrichedForegroundProcessesRow) ForegroundProcess {
	titleSource, err := ParseTitleSource(utils.Coalesce(row.TitleSource, ""))
	assert.NoError(err)

	browserCategory, _ := enumscategories.Parse(row.BrowserCategory)

	result := ForegroundProcess{
		AppName:       utils.ZeroNil(row.ApplicationName),
		AppIdentifier: row.ApplicationIdentifier,
		PID:           row.Pid,
		WindowTitle:   row.WindowTitle,
		TitleSource:   utils.ZeroNil(titleSource),
		Timestamp:     row.CreatedAtUtc,
		Idle:          row.Idle,
		Killed:        row.Killed,
		Enrichments: Enrichments{
			Browser: Browser{
				Vendor:   utils.Coalesce(row.BrowserVendor, ""),
				Tab:      utils.Coalesce(row.Tab, ""),
				Category: utils.ZeroNil(browserCategory),
			},
			Location: Location{
				Latitude:  row.Latitude,
				Longitude: row.Longitude,
				PublicIP:  row.PublicIp,
			},
		},
	}

	return result
}

func mapGetUnindexedForegroundProcessesRowForegroundProcess(row readqueries.GetUnindexedForegroundProcessesRow) ForegroundProcess {
	titleSource, err := ParseTitleSource(utils.Coalesce(row.TitleSource, ""))
	assert.NoError(err)

	browserCategory, _ := enumscategories.Parse(row.BrowserCategory)

	result := ForegroundProcess{
		AppName:       utils.ZeroNil(row.ApplicationName),
		AppIdentifier: row.ApplicationIdentifier,
		PID:           row.Pid,
		WindowTitle:   row.WindowTitle,
		TitleSource:   utils.ZeroNil(titleSource),
		Timestamp:     row.CreatedAtUtc,
		Idle:          row.Idle,
		Killed:        row.Killed,
		Enrichments: Enrichments{
			Browser: Browser{
				Vendor:   utils.Coalesce(row.BrowserVendor, ""),
				Tab:      utils.Coalesce(row.Tab, ""),
				Category: utils.ZeroNil(browserCategory),
			},
			Location: Location{
				Latitude:  row.Latitude,
				Longitude: row.Longitude,
				PublicIP:  row.PublicIp,
			},
		},
	}

	return result
}
