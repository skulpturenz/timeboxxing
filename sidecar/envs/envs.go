package envs

import (
	"github.com/dogmatiq/ferrite"
	enumsenv "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_env"
)

// TODO(auth): Replace this placeholder DSN once the sidecar Sentry project/auth details are finalized.
const PlaceholderSidecarSentryDSN = "https://public@example.com/2"

var (
	GO_ENV = ferrite.
		Enum("GO_ENV", "Golang environment").
		WithMembers(enumsenv.Production.String(), enumsenv.Development.String(), enumsenv.Test.String(), enumsenv.Local.String()).
		WithDefault(enumsenv.Development.String()).
		Required()
	LAUNCH_TOKEN = ferrite.String("SIDECAR_LAUNCH_SECRET", "Launch secret").
			WithSensitiveContent().
			WithDefault("").
			Required()
	SENTRY_DSN = ferrite.String("SIDECAR_SENTRY_DSN", "the Sentry DSN used for sidecar error, trace, and log telemetry").
			WithDefault(PlaceholderSidecarSentryDSN).
			Required()
	GRPC_LISTEN_ADDRESS = ferrite.NetworkAddress("SIDECAR_GRPC_LISTEN_ADDRESS", "the host and port that the gRPC server listens on").
				WithDefault("0.0.0.0:50051").
				Required()
	DB_DSN = ferrite.String("SIDECAR_DATABASE_DSN", "the database data source name").
		WithDefault("test.db").
		Required()
	SQLiteVectorExtensionPath = ferrite.String("SIDECAR_SQLITE_VECTOR_EXTENSION_PATH", "an optional sqlite-vector extension path override for TurboQuant semantic search").
					Optional()
	OPENROUTER_API_KEY = ferrite.String("SIDECAR_OPENROUTER_API_KEY", "the OpenRouter API key used for semantic search and RAG").
				WithSensitiveContent().
				Optional()
	OLLAMA_API_KEY = ferrite.String("SIDECAR_OLLAMA_API_KEY", "the hosted Ollama bearer token used for semantic search and RAG").
			WithSensitiveContent().
			Optional()

	DATABASE_KEY = ferrite.String("SIDECAR_DATABASE_KEY", "the hex-encoded SQLCipher key used to encrypt the sqlite database at rest").
			WithSensitiveContent().
			Optional()
)

func init() {
	ferrite.Init()
}
