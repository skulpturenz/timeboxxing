-- name: CreateTransitionEventDocumentEmbedding :exec
INSERT INTO transition_event_document_float32_embeddings (transition_event_document_id, embedding)
VALUES (?, ?);
