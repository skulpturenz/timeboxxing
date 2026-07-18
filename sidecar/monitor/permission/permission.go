// Package permission models one OS permission some part of the monitor needs. It is a leaf package
// (imports only the standard library) so both the monitor and the enrichment layer can depend on it
// without an import cycle: the enrichment stack produces the permissions it needs and monitor.New
// requests them.
package permission

import "context"

// Permission is one OS permission the monitor (tracker or an enricher) relies on. Request triggers
// the OS authorization prompt; Name/HowToGrant/Granted describe the permission for surfacing status
// to a UI later.
type Permission interface {
	Name() string
	HowToGrant() string
	Granted() bool
	Request(ctx context.Context)
}

// Status describes an OS permission's current grant state, for surfacing in a UI. It is the shared
// shape reported by both the platform tracker and the enrichment providers.
type Status struct {
	Name       string
	Granted    bool
	HowToGrant string
}
