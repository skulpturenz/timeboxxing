package envs

import (
	"github.com/dogmatiq/ferrite"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
)

type GoEnv string

const (
	GoEnvProduction  GoEnv = "production"
	GoEnvDevelopment GoEnv = "development"
	GoEnvTest        GoEnv = "test"
	GoEnvLocal       GoEnv = "local"
)

// TODO(auth): Replace this placeholder DSN once the sidecar Sentry project/auth details are finalized.
const PlaceholderSidecarSentryDSN = "https://public@example.com/2"

var (
	RuntimeEnvironment = ferrite.StringAs[GoEnv]("GO_ENV", "the runtime environment used for telemetry").
				WithConstraint("must be production, development, test, or local", isSupportedGoEnv).
				WithDefault(GoEnvProduction).
				Required()
	SentryDSN = ferrite.String("SIDECAR_SENTRY_DSN", "the Sentry DSN used for sidecar error, trace, and log telemetry").
			WithDefault(PlaceholderSidecarSentryDSN).
			Required()
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
	SQLiteVectorExtensionPath = ferrite.String("SIDECAR_SQLITE_VECTOR_EXTENSION_PATH", "an optional sqlite-vector extension path override for TurboQuant semantic search").
					Optional()
	OpenRouterAPIKey = ferrite.String("SIDECAR_OPENROUTER_API_KEY", "the OpenRouter API key used for semantic search and RAG").
				WithSensitiveContent().
				Optional()
	OllamaAPIKey = ferrite.String("SIDECAR_OLLAMA_API_KEY", "the hosted Ollama bearer token used for semantic search and RAG").
			WithSensitiveContent().
			Optional()

	DatabaseKey = ferrite.String("SIDECAR_DATABASE_KEY", "the hex-encoded SQLCipher key used to encrypt the sqlite database at rest").
			WithSensitiveContent().
			Optional()
)

func init() {
	ferrite.Init()
}

func isSupportedGoEnv(value GoEnv) bool {
	switch value {
	case GoEnvProduction, GoEnvDevelopment, GoEnvTest, GoEnvLocal:
		return true
	default:
		return false
	}
}
