package com.timeboxxing.app.sidecar

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class SidecarProcessManagerTest {
    @Test
    fun childEnvironmentCopiesOnlySemanticSecrets() {
        val targetEnv = mutableMapOf(OpenRouterApiKeyEnvVar to "parent-secret")
        val parentEnv = mapOf(
            OpenRouterApiKeyEnvVar to "parent-secret",
            OllamaApiKeyEnvVar to "parent-ollama-secret",
        )

        configureSidecarEnvironment(
            targetEnv = targetEnv,
            grpcListenAddress = "127.0.0.1:12345",
            databaseDsn = "/tmp/timeboxxing.db",
            parentEnv = parentEnv,
            secrets = SidecarSecrets(openRouterApiKey = "keychain-openrouter-secret"),
        )

        assertEquals("127.0.0.1:12345", targetEnv["SIDECAR_GRPC_LISTEN_ADDRESS"])
        assertEquals("sqlite", targetEnv["SIDECAR_DATABASE_ENGINE"])
        assertEquals("/tmp/timeboxxing.db", targetEnv["SIDECAR_DATABASE_DSN"])
        assertEquals("keychain-openrouter-secret", targetEnv[OpenRouterApiKeyEnvVar])
        assertEquals("parent-ollama-secret", targetEnv[OllamaApiKeyEnvVar])
        assertFalse("SIDECAR_OPENROUTER_BASE_URL" in targetEnv)
        assertFalse("SIDECAR_EMBEDDING_MODEL" in targetEnv)
    }

    @Test
    fun startupLogRedactionRemovesSecretValues() {
        val redactor = SecretRedactor(listOf("literal-secret", "ollama-secret"))
        val message = "SIDECAR_OPENROUTER_API_KEY=literal-secret SIDECAR_OLLAMA_API_KEY=ollama-secret failed for sk-or-v1-fakeplaceholder"

        val redacted = redactor.redact(message)

        assertFalse(redacted.contains("literal-secret"))
        assertFalse(redacted.contains("ollama-secret"))
        assertFalse(redacted.contains("sk-or-v1-fakeplaceholder"))
        assertTrue(redacted.contains("[REDACTED]"))
    }
}
