package timeline

import (
	"container/list"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type QueryExportCSV struct {
	Timeline list.List
}

func (q QueryExportCSV) Exec(ctx context.Context, _ *services.Services[any, any]) (string, error) {
	fps := []models.ForegroundProcess{}
	for v, i := q.Timeline.Front(), 0; v != nil; v, i = v.Next(), i+1 {
		c, ok := v.Value.(models.ForegroundProcess)
		assert.True(ok)
		if !ok {
			return "", fmt.Errorf("command export csv: item %v is not valid", i)
		}

		fps = append(fps, c)
	}

	var csvBuilder strings.Builder
	csvBuilder.Grow(len(fps) + 1)

	headers := []string{
		"appIdentifier",
		"appPath",
		"pid",
		"windowTitle",
		"titleSource",
		"timestamp",
		"idle",
		"killed",
	}
	fmt.Fprintf(&csvBuilder, "%v\n", strings.Join(headers, ","))

	for _, fp := range fps {
		// the minimum we need to derive all other state
		// enrichment data is not guaranteed to be static
		row := []string{
			utils.Coalesce(fp.AppIdentifier, ""),
			utils.Coalesce(fp.AppPath, ""),
			strconv.FormatInt(utils.Coalesce(fp.PID, 0), 10),
			utils.Coalesce(fp.WindowTitle, ""),
			fmt.Sprintf("%v", utils.Coalesce(fp.TitleSource, models.TitleSourceUnknown)),
			fp.Timestamp.Format(time.RFC3339),
			strconv.FormatBool(fp.Idle),
			strconv.FormatBool(fp.Killed),
		}

		fmt.Fprintf(&csvBuilder, "%v\n", strings.Join(row, ","))
	}

	return csvBuilder.String(), nil
}
