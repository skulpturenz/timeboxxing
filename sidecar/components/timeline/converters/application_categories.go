package converters

import (
	applicationmodels "github.com/skulpturenz/timeboxxing/sidecar/components/application/models"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
)

// The category is the only part of the app-metadata enrichment the event store persists, and it
// hangs off the application rather than the observation — so it arrives from a separate read and
// is merged onto an enrichment a row converter has already mapped.

// goverter:converter
// goverter:name ApplicationCategoriesConverter
// goverter:output:file ./application_categories.gen.go
// goverter:skipCopySameType
type applicationCategoriesConverter interface {
	// update rather than convert: every other field is left as the row converter set it.
	// goverter:update target
	// goverter:ignore FriendlyName Description IconPath Source
	// goverter:map Categories Category | firstCategory
	MergeAppMetadata(source applicationmodels.ApplicationCategories, target *models.AppMetadata)
}
