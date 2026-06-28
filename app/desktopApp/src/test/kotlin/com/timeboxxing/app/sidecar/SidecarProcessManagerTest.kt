package com.timeboxxing.app.sidecar

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class SidecarProcessManagerTest {
    @Test
    fun childEnvironmentRemovesOpenRouterKeyAndCopiesOptionalSemanticConfig() {
        val targetEnv = mutableMapOf(OpenRouterApiKeyEnvVar to "parent-secret")
        val parentEnv = mapOf(
            OpenRouterApiKeyEnvVar to "parent-secret",
            "SIDECAR_OPENROUTER_BASE_URL" to "https://openrouter.test/api",
            "SIDECAR_EMBEDDING_MODEL" to "embedding-model",
            "SIDECAR_EMBEDDING_DIMENSION" to "768",
            "SIDECAR_RAG_MODEL" to "rag-model",
        )

        configureSidecarEnvironment(
            targetEnv = targetEnv,
            grpcListenAddress = "127.0.0.1:12345",
            databaseDsn = "/tmp/timeboxxing.db",
            parentEnv = parentEnv,
        )

        assertEquals("127.0.0.1:12345", targetEnv["SIDECAR_GRPC_LISTEN_ADDRESS"])
        assertEquals("sqlite", targetEnv["SIDECAR_DATABASE_ENGINE"])
        assertEquals("/tmp/timeboxxing.db", targetEnv["SIDECAR_DATABASE_DSN"])
        assertFalse(OpenRouterApiKeyEnvVar in targetEnv)
        assertEquals("https://openrouter.test/api", targetEnv["SIDECAR_OPENROUTER_BASE_URL"])
        assertEquals("embedding-model", targetEnv["SIDECAR_EMBEDDING_MODEL"])
        assertEquals("768", targetEnv["SIDECAR_EMBEDDING_DIMENSION"])
        assertEquals("rag-model", targetEnv["SIDECAR_RAG_MODEL"])
    }

    @Test
    fun startupLogRedactionRemovesSecretValues() {
        val redactor = SecretRedactor(listOf("literal-secret"))
        val message = "SIDECAR_OPENROUTER_API_KEY=literal-secret failed for sk-or-v1-fakeplaceholder"

        val redacted = redactor.redact(message)

        assertFalse(redacted.contains("literal-secret"))
        assertFalse(redacted.contains("sk-or-v1-fakeplaceholder"))
        assertTrue(redacted.contains("[REDACTED]"))
    }
}
