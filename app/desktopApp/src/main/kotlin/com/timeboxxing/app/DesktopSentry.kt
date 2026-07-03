package com.timeboxxing.app

import io.sentry.ITransaction
import io.sentry.Sentry

internal object DesktopSentry {
    private const val FlushTimeoutMillis = 2_000L

    fun init(
        env: Map<String, String> = System.getenv(),
        javaEnv: JavaEnv,
    ) {
        val config = desktopSentryConfig(env, javaEnv)
        runCatching {
            Sentry.init { options ->
                options.setDsn(config.dsn)
                options.setEnvironment(config.environment)
                options.setRelease(config.release)
                options.setSendDefaultPii(config.sendDefaultPii)
                options.setTracesSampleRate(config.tracesSampleRate)
                options.getLogs().setEnabled(config.logsEnabled)
            }
            config.tags.forEach { (key, value) ->
                Sentry.setTag(key, value)
            }
        }.onFailure { error ->
            System.err.println("Sentry initialization failed: ${error.message}")
        }
    }

    fun captureException(throwable: Throwable) {
        runCatching {
            if (Sentry.isEnabled()) {
                Sentry.captureException(throwable)
            }
        }
    }

    fun startTransaction(name: String, operation: String): DesktopSentryTransaction {
        val transaction = runCatching {
            if (Sentry.isEnabled()) {
                Sentry.startTransaction(name, operation)
            } else {
                null
            }
        }.getOrNull()
        return DesktopSentryTransaction(transaction)
    }

    fun close() {
        runCatching {
            if (Sentry.isEnabled()) {
                Sentry.flush(FlushTimeoutMillis)
            }
            Sentry.close()
        }
    }
}

internal class DesktopSentryTransaction(
    private val transaction: ITransaction?,
) : AutoCloseable {
    override fun close() {
        runCatching {
            transaction?.finish()
        }
    }
}

internal data class DesktopSentryConfig(
    val dsn: String,
    val release: String,
    val environment: String,
    val sendDefaultPii: Boolean,
    val tracesSampleRate: Double,
    val logsEnabled: Boolean,
    val tags: Map<String, String>,
)

internal const val TimeboxxingAppSentryDsnEnvVar = "TIMEBOXXING_APP_SENTRY_DSN"

// TODO(auth): Replace this placeholder DSN once the desktop Sentry project/auth details are finalized.
internal const val PlaceholderDesktopSentryDsn = "https://public@example.com/1"

internal fun desktopSentryConfig(
    env: Map<String, String>,
    javaEnv: JavaEnv,
): DesktopSentryConfig =
    DesktopSentryConfig(
        dsn = env.trimmedValue(TimeboxxingAppSentryDsnEnvVar)
            ?: PlaceholderDesktopSentryDsn,
        release = env.trimmedValue("SENTRY_RELEASE") ?: "timeboxxing@1.0.0",
        environment = javaEnv.value,
        sendDefaultPii = false,
        tracesSampleRate = 0.2,
        logsEnabled = true,
        tags = mapOf("process" to "desktop-app"),
    )

private fun Map<String, String>.trimmedValue(key: String): String? =
    this[key]
        ?.trim()
        ?.takeIf { it.isNotEmpty() }
