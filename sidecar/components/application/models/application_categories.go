package models

import (
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
)

// ApplicationCategories is one application's classification as the event store records it. The link
// is many-to-many, so it is a slice even though ingest writes exactly one today — consumers that
// carry a single category read the first.
//
// It wraps the slice rather than being one because it is the source of a goverter update converter,
// which only maps from a struct.
type ApplicationCategories struct {
	Categories []enumscategories.Category
}
