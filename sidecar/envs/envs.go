package envs

import (
	"github.com/dogmatiq/ferrite"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
)

var (
	GrpcListenAddress = ferrite.NetworkAddress("SIDECAR_GRPC_LISTEN_ADDRESS", "the host and port that the gRPC server listens on").
				WithDefault("0.0.0.0:50051").
				Required()
	DatabaseEngine = ferrite.StringAs[db.Engine]("SIDECAR_DATABASE_ENGINE", "the database engine used by the sidecar").
			WithConstraint("must be sqlite", func(v db.Engine) bool {
			return v == db.EngineSqlite
		}).
		WithDefault(db.EngineSqlite).
		Required()
	DatabaseDSN = ferrite.String("SIDECAR_DATABASE_DSN", "the database data source name").
			WithDefault("test.db").
			Required()
)

func init() {
	ferrite.Init()
}
