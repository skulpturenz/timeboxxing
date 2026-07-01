package com.timeboxxing.app.sidecar

import java.time.Clock
import java.time.Instant
import java.time.ZoneOffset
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
            SQLiteVectorExtensionPathEnvVar to "/tmp/vector.dylib",
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
        assertEquals("/tmp/vector.dylib", targetEnv[SQLiteVectorExtensionPathEnvVar])
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

    @Test
    fun capturedSidecarLinesAreRedactedBeforeSessionStorage() {
        val sessionLog = SidecarSessionLog(clock = fixedClock())
        val tail = ProcessLogTail(
            redactor = SecretRedactor(listOf("literal-secret")),
            sessionLog = sessionLog,
        )

        tail.append("startup failed with literal-secret and sk-or-v1-placeholder")

        val line = sessionLog.snapshot().single()
        assertEquals("10:15:30.123", line.timestamp)
        assertFalse(line.message.contains("literal-secret"))
        assertFalse(line.message.contains("sk-or-v1-placeholder"))
        assertTrue(line.message.contains("[REDACTED]"))
    }

    @Test
    fun sessionLogSurvivesMultipleSidecarTails() {
        val sessionLog = SidecarSessionLog(clock = fixedClock())
        val firstTail = ProcessLogTail(SecretRedactor(emptyList()), sessionLog)
        val secondTail = ProcessLogTail(SecretRedactor(emptyList()), sessionLog)

        firstTail.append("first sidecar boot")
        secondTail.append("second sidecar boot")

        assertEquals(
            listOf("first sidecar boot", "second sidecar boot"),
            sessionLog.snapshot().map { it.message },
        )
    }

    @Test
    fun sessionLogDropsOldestLinesWhenBounded() {
        val sessionLog = SidecarSessionLog(maxLines = 2, clock = fixedClock())

        sessionLog.append("one")
        sessionLog.append("two")
        sessionLog.append("three")

        assertEquals(listOf("two", "three"), sessionLog.snapshot().map { it.message })
    }

    @Test
    fun startupFailureSuffixUsesOnlyCurrentProcessTail() {
        val sessionLog = SidecarSessionLog(clock = fixedClock())
        val firstTail = ProcessLogTail(SecretRedactor(emptyList()), sessionLog)
        val secondTail = ProcessLogTail(SecretRedactor(emptyList()), sessionLog)

        firstTail.append("previous process line")
        secondTail.append("current process line")

        assertTrue(firstTail.messageSuffix().contains("previous process line"))
        assertFalse(secondTail.messageSuffix().contains("previous process line"))
        assertTrue(secondTail.messageSuffix().contains("current process line"))
        assertEquals(
            listOf("previous process line", "current process line"),
            sessionLog.snapshot().map { it.message },
        )
    }

    private fun fixedClock(): Clock =
        Clock.fixed(Instant.parse("2026-06-29T10:15:30.123Z"), ZoneOffset.UTC)
}
