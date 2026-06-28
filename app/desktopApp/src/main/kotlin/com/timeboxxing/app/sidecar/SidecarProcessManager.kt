package com.timeboxxing.app.sidecar

import com.timeboxxing.app.data.GrpcAmaRepository
import com.timeboxxing.app.data.GrpcUsageHistoryRepository
import com.timeboxxing.app.model.UsageDay
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import java.io.File
import java.net.InetAddress
import java.net.ServerSocket
import java.nio.file.Files
import java.nio.file.StandardCopyOption
import java.time.Duration
import kotlin.concurrent.thread
import kotlin.io.path.createDirectories
import kotlin.io.path.exists

class SidecarProcessManager(
    private val env: Map<String, String> = System.getenv(),
) {
    suspend fun start(readinessDay: UsageDay): SidecarStartResult = withContext(Dispatchers.IO) {
        val redactor = SecretRedactor(emptyList())

        val binary = resolveSidecarBinary()
            ?: return@withContext SidecarStartResult.Failed(
                "Usage sidecar binary was not found. Set TIMEBOXXING_SIDECAR_BINARY for development.",
            )
        val port = findLoopbackPort()
        val target = "127.0.0.1:$port"
        val logTail = ProcessLogTail(redactor)
        val process = ProcessBuilder(binary.absolutePath)
            .redirectErrorStream(true)
            .apply {
                configureSidecarEnvironment(
                    targetEnv = environment(),
                    grpcListenAddress = target,
                    databaseDsn = sidecarDatabasePath().toString(),
                    parentEnv = env,
                )
            }
            .start()
        logTail.start(process)

        val repository = GrpcUsageHistoryRepository(target)
        val amaRepository = GrpcAmaRepository(target)
        val deadlineNanos = System.nanoTime() + Duration.ofSeconds(15).toNanos()
        var lastError: Throwable? = null

        while (System.nanoTime() < deadlineNanos) {
            if (!process.isAlive) {
                repository.close()
                amaRepository.close()
                return@withContext SidecarStartResult.Failed(
                    "Usage sidecar exited before it became ready.${logTail.messageSuffix()}",
                )
            }

            val ready = runCatching { repository.getUsageEvents(readinessDay) }
            if (ready.isSuccess) {
                return@withContext SidecarStartResult.Started(
                    SidecarConnection(
                        repository = repository,
                        amaRepository = amaRepository,
                        process = process,
                    ),
                )
            }
            lastError = ready.exceptionOrNull()
            delay(250)
        }

        repository.close()
        amaRepository.close()
        stopProcess(process)
        SidecarStartResult.Failed(
            "Usage sidecar did not become ready: ${redactor.redact(lastError?.message ?: "timed out")}.${logTail.messageSuffix()}",
        )
    }

    private fun resolveSidecarBinary(): File? {
        env["TIMEBOXXING_SIDECAR_BINARY"]
            ?.trim()
            ?.takeIf { it.isNotEmpty() }
            ?.let { return File(it) }

        val resourceName = "sidecar/$sidecarExecutableName"
        val classLoader = Thread.currentThread().contextClassLoader
        val resource = classLoader.getResource(resourceName) ?: return null
        if (resource.protocol == "file") {
            return File(resource.toURI()).apply { setExecutable(true) }
        }

        val suffix = if (sidecarExecutableName.endsWith(".exe")) ".exe" else ""
        val tempFile = Files.createTempFile("timeboxxing-sidecar-", suffix).toFile()
        classLoader.getResourceAsStream(resourceName)?.use { input ->
            Files.copy(input, tempFile.toPath(), StandardCopyOption.REPLACE_EXISTING)
        } ?: return null
        tempFile.setExecutable(true)
        tempFile.deleteOnExit()
        return tempFile
    }

    private fun sidecarDatabasePath() =
        appDataDirectory().resolve("timeboxxing.db")

    private fun appDataDirectory() =
        File(System.getProperty("user.home")).toPath()
            .resolve(".timeboxxing")
            .also { if (!it.exists()) it.createDirectories() }

    private fun findLoopbackPort(): Int =
        ServerSocket(0, 0, InetAddress.getByName("127.0.0.1")).use { it.localPort }
}

sealed interface SidecarStartResult {
    data class Started(val connection: SidecarConnection) : SidecarStartResult
    data class Failed(
        val message: String,
        val reason: SidecarStartFailureReason = SidecarStartFailureReason.Other,
    ) : SidecarStartResult
}

enum class SidecarStartFailureReason {
    Other,
}

class SidecarConnection(
    val repository: GrpcUsageHistoryRepository,
    val amaRepository: GrpcAmaRepository,
    private val process: Process,
) : AutoCloseable {
    override fun close() {
        repository.close()
        amaRepository.close()
        stopProcess(process)
    }
}

private class ProcessLogTail(
    private val redactor: SecretRedactor,
    private val maxLines: Int = 16,
) {
    private val lines = ArrayDeque<String>()

    fun start(process: Process) {
        thread(name = "timeboxxing-sidecar-log", isDaemon = true) {
            process.inputStream.bufferedReader().useLines { sequence ->
                sequence.forEach { append(it) }
            }
        }
    }

    @Synchronized
    private fun append(line: String) {
        lines.addLast(redactor.redact(line))
        while (lines.size > maxLines) {
            lines.removeFirst()
        }
    }

    @Synchronized
    fun messageSuffix(): String =
        if (lines.isEmpty()) {
            ""
        } else {
            " Recent sidecar log: ${lines.joinToString(" | ")}"
        }
}

private fun stopProcess(process: Process) {
    if (!process.isAlive) return
    process.destroy()
    if (!process.waitFor(2, java.util.concurrent.TimeUnit.SECONDS)) {
        process.destroyForcibly()
    }
}

internal fun configureSidecarEnvironment(
    targetEnv: MutableMap<String, String>,
    grpcListenAddress: String,
    databaseDsn: String,
    parentEnv: Map<String, String>,
) {
    targetEnv["SIDECAR_GRPC_LISTEN_ADDRESS"] = grpcListenAddress
    targetEnv["SIDECAR_DATABASE_ENGINE"] = "sqlite"
    targetEnv["SIDECAR_DATABASE_DSN"] = databaseDsn
    targetEnv.remove(OpenRouterApiKeyEnvVar)
    optionalSemanticEnvKeys.forEach { key ->
        parentEnv[key]?.let { value -> targetEnv[key] = value }
    }
}

internal class SecretRedactor(
    secrets: Collection<String>,
) {
    private val literalSecrets = secrets
        .map { it.trim() }
        .filter { it.length >= 4 }
        .distinct()

    fun redact(value: String): String {
        var redacted = openRouterKeyPattern.replace(value, "[REDACTED]")
        redacted = openRouterEnvPattern.replace(redacted) { match ->
            "${match.groupValues[1]}[REDACTED]"
        }
        literalSecrets.forEach { secret ->
            redacted = redacted.replace(secret, "[REDACTED]")
        }
        return redacted
    }
}

private val sidecarExecutableName: String =
    if (System.getProperty("os.name").lowercase().contains("windows")) {
        "timeboxxing-sidecar.exe"
    } else {
        "timeboxxing-sidecar"
    }

private val optionalSemanticEnvKeys = listOf(
    "SIDECAR_OPENROUTER_BASE_URL",
    "SIDECAR_EMBEDDING_MODEL",
    "SIDECAR_EMBEDDING_DIMENSION",
    "SIDECAR_RAG_MODEL",
)

internal const val OpenRouterApiKeyEnvVar = "SIDECAR_OPENROUTER_API_KEY"

private val openRouterKeyPattern = Regex("""sk-or-v1-[A-Za-z0-9_-]+""")
private val openRouterEnvPattern = Regex("""(SIDECAR_OPENROUTER_API_KEY\s*=\s*)\S+""")
