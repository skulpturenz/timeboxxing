package com.timeboxxing.app.sidecar

import com.timeboxxing.data.grpc.GrpcAmaRepository
import com.timeboxxing.data.grpc.GrpcProjectRepository
import com.timeboxxing.data.grpc.GrpcSettingsRepository
import com.timeboxxing.data.grpc.GrpcTimesheetRepository
import com.timeboxxing.data.grpc.GrpcUsageHistoryRepository
import com.timeboxxing.domain.model.UsageDay
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import java.io.File
import java.net.InetAddress
import java.net.ServerSocket
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption
import java.time.Duration
import kotlin.concurrent.thread
import kotlin.io.path.createDirectories
import kotlin.io.path.exists

class SidecarProcessManager(
    private val env: Map<String, String> = System.getenv(),
    private val sessionLog: SidecarSessionLog = SidecarSessionLog(),
    private val classLoader: ClassLoader = Thread.currentThread().contextClassLoader,
    private val systemProperty: (String) -> String? = System::getProperty,
    osName: String = System.getProperty("os.name"),
    userHome: String = System.getProperty("user.home"),
) {
    val dataDirectory: Path = resolveTimeboxxingDataDirectory(env, osName, userHome)

    suspend fun start(
        readinessDay: UsageDay,
        secrets: SidecarSecrets = SidecarSecrets(),
    ): SidecarStartResult = withContext(Dispatchers.IO) {
        val redactor = SecretRedactor(secrets.values())

        val binary = resolveSidecarBinary()
            ?: return@withContext SidecarStartResult.Failed(
                "Usage sidecar binary was not found. Set TIMEBOXXING_SIDECAR_BINARY for development.",
            )
        val port = findLoopbackPort()
        val target = "127.0.0.1:$port"
        val logTail = ProcessLogTail(redactor, sessionLog)
        val process = ProcessBuilder(binary.absolutePath)
            .redirectErrorStream(true)
            .apply {
                configureSidecarEnvironment(
                    targetEnv = environment(),
                    grpcListenAddress = target,
                    databaseDsn = sidecarDatabasePath().toString(),
                    parentEnv = env,
                    secrets = secrets,
                )
            }
            .start()
        logTail.start(process)

        val repository = GrpcUsageHistoryRepository(target)
        val amaRepository = GrpcAmaRepository(target)
        val settingsRepository = GrpcSettingsRepository(target)
        val projectRepository = GrpcProjectRepository(target)
        val timesheetRepository = GrpcTimesheetRepository(target)
        val deadlineNanos = System.nanoTime() + Duration.ofSeconds(15).toNanos()
        var lastError: Throwable? = null

        while (System.nanoTime() < deadlineNanos) {
            if (!process.isAlive) {
                repository.close()
                amaRepository.close()
                settingsRepository.close()
                projectRepository.close()
                timesheetRepository.close()
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
                        settingsRepository = settingsRepository,
                        projectRepository = projectRepository,
                        timesheetRepository = timesheetRepository,
                        process = process,
                    ),
                )
            }
            lastError = ready.exceptionOrNull()
            delay(250)
        }

        repository.close()
        amaRepository.close()
        settingsRepository.close()
        projectRepository.close()
        timesheetRepository.close()
        stopProcess(process)
        SidecarStartResult.Failed(
            "Usage sidecar did not become ready: ${redactor.redact(lastError?.message ?: "timed out")}.${logTail.messageSuffix()}",
        )
    }

    private fun resolveSidecarBinary(): File? {
        return resolveSidecarBinary(
            env = env,
            appResourcesDir = systemProperty(ComposeApplicationResourcesDirProperty),
            classLoader = classLoader,
        )
    }

    private fun sidecarDatabasePath() =
        dataDirectory.also { if (!it.exists()) it.createDirectories() }
            .resolve("timeboxxing.db")


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
    val settingsRepository: GrpcSettingsRepository,
    val projectRepository: GrpcProjectRepository,
    val timesheetRepository: GrpcTimesheetRepository,
    private val process: Process,
) : AutoCloseable {
    override fun close() {
        repository.close()
        amaRepository.close()
        settingsRepository.close()
        projectRepository.close()
        timesheetRepository.close()
        stopProcess(process)
    }
}

data class SidecarSecrets(
    val openRouterApiKey: String = "",
    val ollamaApiKey: String = "",
) {
    fun values(): List<String> = listOf(openRouterApiKey, ollamaApiKey)
}

internal class ProcessLogTail(
    private val redactor: SecretRedactor,
    private val sessionLog: SidecarSessionLog,
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
    internal fun append(line: String) {
        val redacted = redactor.redact(line)
        lines.addLast(redacted)
        while (lines.size > maxLines) {
            lines.removeFirst()
        }
        sessionLog.append(redacted)
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
    sqliteVectorExtensionPath: String? = null,
    parentEnv: Map<String, String>,
    secrets: SidecarSecrets = SidecarSecrets(),
) {
    targetEnv["SIDECAR_GRPC_LISTEN_ADDRESS"] = grpcListenAddress
    targetEnv["SIDECAR_DATABASE_ENGINE"] = "sqlite"
    targetEnv["SIDECAR_DATABASE_DSN"] = databaseDsn
    targetEnv.remove(SQLiteVectorExtensionPathEnvVar)
    targetEnv.remove(OpenRouterApiKeyEnvVar)
    targetEnv.remove(OllamaApiKeyEnvVar)
    val vectorExtensionPath = sqliteVectorExtensionPath.orEmpty().ifBlank {
        parentEnv[SQLiteVectorExtensionPathEnvVar].orEmpty()
    }
    if (vectorExtensionPath.isNotBlank()) {
        targetEnv[SQLiteVectorExtensionPathEnvVar] = vectorExtensionPath
    }
    val openRouterApiKey = secrets.openRouterApiKey.ifBlank { parentEnv[OpenRouterApiKeyEnvVar].orEmpty() }
    if (openRouterApiKey.isNotBlank()) {
        targetEnv[OpenRouterApiKeyEnvVar] = openRouterApiKey
    }
    val ollamaApiKey = secrets.ollamaApiKey.ifBlank { parentEnv[OllamaApiKeyEnvVar].orEmpty() }
    if (ollamaApiKey.isNotBlank()) {
        targetEnv[OllamaApiKeyEnvVar] = ollamaApiKey
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
        redacted = semanticSecretEnvPattern.replace(redacted) { match ->
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

private val sqliteVectorLibrarySuffix: String =
    when {
        System.getProperty("os.name").lowercase().contains("windows") -> ".dll"
        System.getProperty("os.name").lowercase().contains("mac") -> ".dylib"
        else -> ".so"
    }

private val sqliteVectorResourcePath: String? = run {
    val os = System.getProperty("os.name").lowercase()
    val arch = System.getProperty("os.arch").lowercase()
    val platform = when {
        os.contains("mac") -> "darwin"
        os.contains("windows") -> "windows"
        os.contains("linux") -> "linux"
        else -> null
    }
    val normalizedArch = when {
        arch == "aarch64" || arch == "arm64" -> "arm64"
        arch == "x86_64" || arch == "amd64" -> "amd64"
        else -> null
    }
    if (platform == null || normalizedArch == null) {
        null
    } else {
        "$platform-$normalizedArch/vector$sqliteVectorLibrarySuffix"
    }
}

internal const val SQLiteVectorExtensionPathEnvVar = "SIDECAR_SQLITE_VECTOR_EXTENSION_PATH"
internal const val OpenRouterApiKeyEnvVar = "SIDECAR_OPENROUTER_API_KEY"
internal const val OllamaApiKeyEnvVar = "SIDECAR_OLLAMA_API_KEY"
internal const val TimeboxxingSidecarBinaryEnvVar = "TIMEBOXXING_SIDECAR_BINARY"
private const val ComposeApplicationResourcesDirProperty = "compose.application.resources.dir"

private val openRouterKeyPattern = Regex("""sk-or-v1-[A-Za-z0-9_-]+""")
private val semanticSecretEnvPattern = Regex("""((?:SIDECAR_OPENROUTER_API_KEY|SIDECAR_OLLAMA_API_KEY)\s*=\s*)\S+""")

internal fun resolveTimeboxxingDataDirectory(
    env: Map<String, String>,
    osName: String,
    userHome: String,
): Path {
    val home = userHome.ifBlank { "." }
    val normalizedOs = osName.lowercase()
    return when {
        normalizedOs.contains("windows") -> {
            val base = env["LOCALAPPDATA"]
                ?.trim()
                ?.takeIf { it.isNotEmpty() }
                ?.let { File(it).toPath() }
                ?: File(home).toPath().resolve("AppData").resolve("Local")
            base.resolve("Timeboxxing")
        }

        normalizedOs.contains("mac") || normalizedOs.contains("darwin") -> {
            File(home).toPath()
                .resolve("Library")
                .resolve("Application Support")
                .resolve("Timeboxxing")
        }

        else -> {
            val base = env["XDG_DATA_HOME"]
                ?.trim()
                ?.takeIf { it.isNotEmpty() }
                ?.let { File(it).toPath() }
                ?: File(home).toPath().resolve(".local").resolve("share")
            base.resolve("timeboxxing")
        }
    }
}

internal fun resolveSidecarBinary(
    env: Map<String, String>,
    appResourcesDir: String?,
    classLoader: ClassLoader,
    executableName: String = sidecarExecutableName,
): File? =
    resolveInstalledSidecarBinary(appResourcesDir, executableName)
        ?: resolveConfiguredSidecarBinary(env)
        ?: resolveClasspathSidecarBinary(classLoader, executableName)

internal fun resolveInstalledSidecarBinary(
    appResourcesDir: String?,
    executableName: String = sidecarExecutableName,
): File? {
    val resourcesDir = appResourcesDir
        ?.trim()
        ?.takeIf { it.isNotEmpty() }
        ?: return null
    val candidate = File(resourcesDir)
        .resolve("sidecar")
        .resolve(executableName)
    return candidate
        .takeIf { it.isFile }
        ?.apply { setExecutable(true) }
}

private fun resolveConfiguredSidecarBinary(
    env: Map<String, String>,
): File? =
    env[TimeboxxingSidecarBinaryEnvVar]
        ?.trim()
        ?.takeIf { it.isNotEmpty() }
        ?.let { File(it) }

private fun resolveClasspathSidecarBinary(
    classLoader: ClassLoader,
    executableName: String,
): File? {
    val resourceName = "sidecar/$executableName"
    val resource = classLoader.getResource(resourceName) ?: return null
    if (resource.protocol == "file") {
        return File(resource.toURI()).apply { setExecutable(true) }
    }

    val tempDir = Files.createTempDirectory("timeboxxing-sidecar-").toFile()
    tempDir.deleteOnExit()
    val tempFile = tempDir.resolve(executableName)
    classLoader.getResourceAsStream(resourceName)?.use { input ->
        Files.copy(input, tempFile.toPath(), StandardCopyOption.REPLACE_EXISTING)
    } ?: return null
    copyBundledSQLiteVectorExtension(classLoader, tempDir)
    tempFile.setExecutable(true)
    tempFile.deleteOnExit()
    return tempFile
}

private fun copyBundledSQLiteVectorExtension(classLoader: ClassLoader, sidecarDir: File) {
    val resourcePath = sqliteVectorResourcePath ?: return
    val resourceName = "sidecar/sqlite-vector/$resourcePath"
    val target = sidecarDir.toPath()
        .resolve("sqlite-vector")
        .resolve(resourcePath)
    target.parent.createDirectories()
    classLoader.getResourceAsStream(resourceName)?.use { input ->
        Files.copy(input, target, StandardCopyOption.REPLACE_EXISTING)
    } ?: return
    target.toFile().deleteOnExit()
}
