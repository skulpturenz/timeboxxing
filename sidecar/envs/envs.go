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
	OpenRouterAPIKey = ferrite.String("SIDECAR_OPENROUTER_API_KEY", "the OpenRouter API key used for semantic search and RAG").
				WithDefault("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").
				Required()
	OpenRouterBaseURL = ferrite.String("SIDECAR_OPENROUTER_BASE_URL", "the OpenRouter API base URL").
				WithDefault("https://openrouter.ai/api/v1").
				Required()
	EmbeddingModel = ferrite.String("SIDECAR_EMBEDDING_MODEL", "the OpenRouter embedding model used for semantic search").
			WithDefault("nvidia/llama-nemotron-embed-vl-1b-v2:free").
			Required()
	EmbeddingDimension = ferrite.Unsigned[uint]("SIDECAR_EMBEDDING_DIMENSION", "the embedding vector dimension").
				WithDefault(2048).
				Required()
	RAGModel = ferrite.String("SIDECAR_RAG_MODEL", "the OpenRouter generation model used for RAG answers").
			WithDefault("google/gemma-4-31b-it:free").
			Required()
)

func init() {
	ferrite.Init()
}
