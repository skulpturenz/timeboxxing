package masking

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/stretchr/testify/require"
)

func newTestServices(ctx context.Context, t *testing.T) *services.Services[any, any] {
	t.Helper()

	database, err := db.New(ctx, db.Options{
		SQLiteVectorExtensionPath: nil,
		DSN:                       db.NewDSN(filepath.Join(t.TempDir(), "test.db")),
	})
	require.NoError(t, err, "create test database")
	t.Cleanup(func() {
		require.NoError(t, database.Close(), "close test database")
	})

	svcs := services.New()
	db.Register(svcs, database)

	return svcs
}

func observation(name string, identifier string, domain string) models.ForegroundProcess {
	browserCategory := enumscategories.CategoryWebBrowsing

	return models.ForegroundProcess{
		AppName:       &name,
		AppIdentifier: &identifier,
		AppPath:       new("/Users/ada/Applications/" + name + ".app"),
		PID:           new(int64(4321)),
		WindowTitle:   new(name + " — " + domain),
		TitleSource:   nil,
		Timestamp:     time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC),
		Idle:          false,
		Killed:        false,
		Enrichments: models.Enrichments{
			Appmetadata: models.AppMetadata{
				FriendlyName: name,
				Description:  "",
				Category:     enumscategories.CategoryWebBrowsing,
				IconPath:     "",
				Source:       "",
			},
			Browser: models.Browser{
				Vendor:        name,
				Category:      &browserCategory,
				AppIdentifier: &identifier,
				Tab:           "",
				CdpURL:        "",
				Domain:        domain,
			},
			Location: models.Location{Latitude: nil, Longitude: nil, PublicIP: nil},
		},
	}
}

func idleObservation() models.ForegroundProcess {
	return models.ForegroundProcess{
		AppName:       nil,
		AppIdentifier: nil,
		AppPath:       nil,
		PID:           nil,
		WindowTitle:   nil,
		TitleSource:   nil,
		Timestamp:     time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC),
		Idle:          true,
		Killed:        false,
		Enrichments: models.Enrichments{
			Appmetadata: models.AppMetadata{
				FriendlyName: "",
				Description:  "",
				Category:     enumscategories.CategoryUnknown,
				IconPath:     "",
				Source:       "",
			},
			Browser: models.Browser{
				Vendor:        "",
				Category:      nil,
				AppIdentifier: nil,
				Tab:           "",
				CdpURL:        "",
				Domain:        "",
			},
			Location: models.Location{Latitude: nil, Longitude: nil, PublicIP: nil},
		},
	}
}

func generate(ctx context.Context, t *testing.T, svcs *services.Services[any, any],
	processes ...models.ForegroundProcess,
) models.MaskingTables {
	t.Helper()

	command := CommandGenerateMasks{Processes: processes}
	require.NoError(t, command.Exec(ctx, svcs), "generate masks")

	query := QueryGetMaskingTables{}
	tables, err := query.Exec(ctx, svcs)
	require.NoError(t, err, "get masking tables")

	return tables
}
