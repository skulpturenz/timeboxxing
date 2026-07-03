package com.timeboxxing.app

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class DesktopSentryTest {
    @Test
    fun configUsesPlaceholderDsnWhenAppDsnIsMissing() {
        val config = desktopSentryConfig(env = emptyMap(), javaEnv = JavaEnv.Local)

        assertEquals(PlaceholderDesktopSentryDsn, config.dsn)
    }

    @Test
    fun configUsesTrimmedAppDsnOverride() {
        val config = desktopSentryConfig(
            env = mapOf(TimeboxxingAppSentryDsnEnvVar to " https://public@example.com/123 "),
            javaEnv = JavaEnv.Local,
        )

        assertEquals("https://public@example.com/123", config.dsn)
    }

    @Test
    fun configUsesBuildDerivedJavaEnvForEnvironment() {
        val config = desktopSentryConfig(
            env = emptyMap(),
            javaEnv = JavaEnv.Test,
        )

        assertEquals("test", config.environment)
    }

    @Test
    fun configIgnoresRuntimeEnvironmentForEnvironment() {
        val config = desktopSentryConfig(
            env = mapOf(JavaEnvVar to "production", GoEnvVar to "development"),
            javaEnv = JavaEnv.Local,
        )

        assertEquals("local", config.environment)
    }

    @Test
    fun configKeepsPrivacyAndTelemetryDefaults() {
        val config = desktopSentryConfig(env = emptyMap(), javaEnv = JavaEnv.Local)

        assertFalse(config.sendDefaultPii)
        assertEquals(0.2, config.tracesSampleRate)
        assertTrue(config.logsEnabled)
        assertEquals(mapOf("process" to "desktop-app"), config.tags)
    }
}
