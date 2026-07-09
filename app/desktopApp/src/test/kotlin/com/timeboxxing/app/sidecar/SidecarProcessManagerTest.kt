package com.timeboxxing.app.sidecar

import com.timeboxxing.app.GoEnvVar
import com.timeboxxing.app.JavaEnv
import java.io.File
import java.net.URLClassLoader
import java.nio.file.Files
import java.time.Clock
import java.time.Instant
import java.time.ZoneOffset
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class SidecarProcessManagerTest {
    @Test
    fun macOsDataDirectoryUsesApplicationSupport() {
        val dir = resolveTimeboxxingDataDirectory(
            env = emptyMap(),
            osName = "Mac OS X",
            userHome = "/Users/tester",
        )

        assertEquals(
            File("/Users/tester/Library/Application Support/Timeboxxing").toPath(),
            dir,
        )
    }

    @Test
    fun windowsDataDirectoryUsesLocalAppDataWhenAvailable() {
        val dir = resolveTimeboxxingDataDirectory(
            env = mapOf("LOCALAPPDATA" to "/Users/tester/AppData/Local"),
            osName = "Windows 11",
            userHome = "/Users/tester",
        )

        assertEquals(
            File("/Users/tester/AppData/Local/Skulpture/Timeboxxing").toPath(),
            dir,
        )
    }

    @Test
    fun windowsDataDirectoryFallsBackToUserLocalAppData() {
        val dir = resolveTimeboxxingDataDirectory(
            env = emptyMap(),
            osName = "Windows 11",
            userHome = "/Users/tester",
        )

        assertEquals(
            File("/Users/tester/AppData/Local/Skulpture/Timeboxxing").toPath(),
            dir,
        )
    }

    @Test
    fun linuxDataDirectoryUsesXdgDataHomeWhenAvailable() {
        val dir = resolveTimeboxxingDataDirectory(
            env = mapOf("XDG_DATA_HOME" to "/Users/tester/.xdg-data"),
            osName = "Linux",
            userHome = "/Users/tester",
        )

        assertEquals(
            File("/Users/tester/.xdg-data/timeboxxing").toPath(),
            dir,
        )
    }

    @Test
    fun linuxDataDirectoryFallsBackToLocalShare() {
        val dir = resolveTimeboxxingDataDirectory(
            env = emptyMap(),
            osName = "Linux",
            userHome = "/Users/tester",
        )

        assertEquals(
            File("/Users/tester/.local/share/timeboxxing").toPath(),
            dir,
        )
    }

    @Test
    fun installedSidecarResourceWinsOverDevAndClasspathFallbacks() {
        val appResources = tempDir("installed-resources")
        val installedBinary = appResources.resolve("sidecar/timeboxxing-sidecar")
        installedBinary.parentFile.mkdirs()
        installedBinary.writeText("installed")
        val devBinary = tempFile("dev-sidecar")
        val classpathRoot = tempDir("classpath-resources")
        classpathRoot.resolve("sidecar/timeboxxing-sidecar").apply {
            parentFile.mkdirs()
            writeText("classpath")
        }

        URLClassLoader(arrayOf(classpathRoot.toURI().toURL()), null).use { classLoader ->
            val resolved = resolveSidecarBinary(
                env = mapOf(TimeboxxingSidecarBinaryEnvVar to devBinary.absolutePath),
                appResourcesDir = appResources.absolutePath,
                classLoader = classLoader,
                executableName = "timeboxxing-sidecar",
            )

            assertEquals(installedBinary.canonicalFile, resolved?.canonicalFile)
        }
    }

    @Test
    fun devSidecarOverrideIsUsedWhenInstalledResourceIsMissing() {
        val devBinary = tempFile("dev-sidecar")
        val classpathRoot = tempDir("classpath-resources")
        classpathRoot.resolve("sidecar/timeboxxing-sidecar").apply {
            parentFile.mkdirs()
            writeText("classpath")
        }

        URLClassLoader(arrayOf(classpathRoot.toURI().toURL()), null).use { classLoader ->
            val resolved = resolveSidecarBinary(
                env = mapOf(TimeboxxingSidecarBinaryEnvVar to devBinary.absolutePath),
                appResourcesDir = null,
                classLoader = classLoader,
                executableName = "timeboxxing-sidecar",
            )

            assertEquals(devBinary.canonicalFile, resolved?.canonicalFile)
        }
    }

    @Test
    fun classpathSidecarFallbackIsUsedWhenInstalledAndDevPathsAreMissing() {
        val classpathRoot = tempDir("classpath-resources")
        val classpathBinary = classpathRoot.resolve("sidecar/timeboxxing-sidecar")
        classpathBinary.parentFile.mkdirs()
        classpathBinary.writeText("classpath")

        URLClassLoader(arrayOf(classpathRoot.toURI().toURL()), null).use { classLoader ->
            val resolved = resolveSidecarBinary(
                env = emptyMap(),
                appResourcesDir = null,
                classLoader = classLoader,
                executableName = "timeboxxing-sidecar",
            )

            assertEquals(classpathBinary.canonicalFile, resolved?.canonicalFile)
        }
    }

    @Test
    fun childEnvironmentNeverCarriesApiKeys() {
        val targetEnv = mutableMapOf(
            GoEnvVar to "stale",
            OpenRouterApiKeyEnvVar to "inherited-openrouter-secret",
            OllamaApiKeyEnvVar to "inherited-ollama-secret",
            SidecarSentryDsnEnvVar to "stale-sentry-dsn",
            "SIDECAR_OPENROUTER_BASE_URL" to "https://example.invalid",
            "SIDECAR_SENTRY_AUTH_TOKEN" to "stale-sidecar-auth-token",
            "SENTRY_AUTH_TOKEN" to "stale-auth-token",
        )
        val parentEnv = mapOf(
            OpenRouterApiKeyEnvVar to "parent-secret",
            OllamaApiKeyEnvVar to "parent-ollama-secret",
            SQLiteVectorExtensionPathEnvVar to "/tmp/vector.dylib",
            SidecarSentryDsnEnvVar to "https://public@example.com/99",
            GoEnvVar to "LOCAL",
        )

        configureSidecarEnvironment(
            targetEnv = targetEnv,
            grpcListenAddress = "127.0.0.1:12345",
            databaseDsn = "/tmp/timeboxxing.db",
            parentEnv = parentEnv,
            javaEnv = JavaEnv.Test,
        )

        assertEquals("127.0.0.1:12345", targetEnv["SIDECAR_GRPC_LISTEN_ADDRESS"])
        assertEquals("sqlite", targetEnv["SIDECAR_DATABASE_ENGINE"])
        assertEquals("/tmp/timeboxxing.db", targetEnv["SIDECAR_DATABASE_DSN"])
        assertEquals("/tmp/vector.dylib", targetEnv[SQLiteVectorExtensionPathEnvVar])
        // Secrets are handed over stdin, never via the environment — even an inherited copy is stripped.
        assertFalse(OpenRouterApiKeyEnvVar in targetEnv)
        assertFalse(OllamaApiKeyEnvVar in targetEnv)
        assertEquals("https://public@example.com/99", targetEnv[SidecarSentryDsnEnvVar])
        assertEquals("test", targetEnv[GoEnvVar])
        assertFalse("SIDECAR_OPENROUTER_BASE_URL" in targetEnv)
        assertFalse("SIDECAR_EMBEDDING_MODEL" in targetEnv)
        assertFalse("SIDECAR_SENTRY_AUTH_TOKEN" in targetEnv)
        assertFalse("SENTRY_AUTH_TOKEN" in targetEnv)
    }

    @Test
    fun resolveSidecarSecretsPrefersKeychainThenParentEnv() {
        val parentEnv = mapOf(
            OpenRouterApiKeyEnvVar to "parent-openrouter",
            OllamaApiKeyEnvVar to "parent-ollama",
        )

        val resolved = resolveSidecarSecrets(
            secrets = SidecarSecrets(openRouterApiKey = "keychain-openrouter"),
            parentEnv = parentEnv,
        )

        // Keychain value wins for OpenRouter; Ollama falls back to the parent environment.
        assertEquals("keychain-openrouter", resolved.openRouterApiKey)
        assertEquals("parent-ollama", resolved.ollamaApiKey)
    }

    @Test
    fun buildSecretHandoffPayloadEmitsOnlyNonBlankSecrets() {
        val payload = buildSecretHandoffPayload(
            SidecarSecrets(openRouterApiKey = "or-key", ollamaApiKey = ""),
        )

        assertEquals("$OpenRouterApiKeyEnvVar=or-key\n", payload)
    }

    @Test
    fun buildSecretHandoffPayloadIsEmptyWhenNoSecrets() {
        assertEquals("", buildSecretHandoffPayload(SidecarSecrets()))
    }

    @Test
    fun buildSecretHandoffPayloadEmitsDatabaseKey() {
        val payload = buildSecretHandoffPayload(
            SidecarSecrets(openRouterApiKey = "or-key", databaseKey = "deadbeef"),
        )

        assertTrue(payload.contains("$DatabaseKeyEnvVar=deadbeef\n"))
        assertTrue(payload.contains("$OpenRouterApiKeyEnvVar=or-key\n"))
    }

    @Test
    fun childEnvironmentStripsInheritedDatabaseKey() {
        val targetEnv = mutableMapOf(DatabaseKeyEnvVar to "inherited-db-key")

        configureSidecarEnvironment(
            targetEnv = targetEnv,
            grpcListenAddress = "127.0.0.1:12345",
            databaseDsn = "/tmp/timeboxxing.db",
            parentEnv = emptyMap(),
        )

        // The database key is handed over stdin, never via the environment.
        assertFalse(DatabaseKeyEnvVar in targetEnv)
    }

    @Test
    fun resolveSidecarSecretsResolvesDatabaseKeyFromParentEnv() {
        val resolved = resolveSidecarSecrets(
            secrets = SidecarSecrets(),
            parentEnv = mapOf(DatabaseKeyEnvVar to "parent-db-key"),
        )

        assertEquals("parent-db-key", resolved.databaseKey)
    }

    @Test
    fun startupLogRedactionRemovesDatabaseKey() {
        val redactor = SecretRedactor(emptyList())
        val message = "opening $DatabaseKeyEnvVar=2dd29ca851e7b56e failed"

        val redacted = redactor.redact(message)

        assertFalse(redacted.contains("2dd29ca851e7b56e"))
        assertTrue(redacted.contains("[REDACTED]"))
    }

    @Test
    fun childEnvironmentFallsBackToPlaceholderSentryDsn() {
        val targetEnv = mutableMapOf<String, String>()

        configureSidecarEnvironment(
            targetEnv = targetEnv,
            grpcListenAddress = "127.0.0.1:12345",
            databaseDsn = "/tmp/timeboxxing.db",
            parentEnv = emptyMap(),
        )

        assertEquals(PlaceholderSidecarSentryDsn, targetEnv[SidecarSentryDsnEnvVar])
    }

    @Test
    fun childEnvironmentPropagatesInjectedJavaEnvAsGoEnv() {
        val targetEnv = mutableMapOf(GoEnvVar to "stale")

        configureSidecarEnvironment(
            targetEnv = targetEnv,
            grpcListenAddress = "127.0.0.1:12345",
            databaseDsn = "/tmp/timeboxxing.db",
            parentEnv = mapOf(GoEnvVar to "staging"),
            javaEnv = JavaEnv.Production,
        )

        assertEquals("production", targetEnv[GoEnvVar])
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
    fun startupLogRedactionRemovesSentryAuthTokens() {
        val redactor = SecretRedactor(emptyList())
        val message = "SIDECAR_SENTRY_AUTH_TOKEN=sidecar-token SENTRY_AUTH_TOKEN=release-token"

        val redacted = redactor.redact(message)

        assertFalse(redacted.contains("sidecar-token"))
        assertFalse(redacted.contains("release-token"))
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

    private fun tempDir(prefix: String): File =
        Files.createTempDirectory(prefix).toFile()

    private fun tempFile(prefix: String): File =
        Files.createTempFile(prefix, ".bin").toFile()
}
