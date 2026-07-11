package com.timeboxxing.app

import java.io.File
import java.nio.file.Path
import kotlin.io.path.createDirectories

internal interface SecretStore {
    suspend fun read(key: SecretKey): String
    suspend fun write(key: SecretKey, value: String)
    suspend fun delete(key: SecretKey)
}

internal enum class SecretKey(
    val service: String,
) {
    OpenRouter("com.timeboxxing.app.openrouter"),
    Ollama("com.timeboxxing.app.ollama"),
    DatabaseKey("com.timeboxxing.app.database-key"),
}

internal enum class SecretPlatform { MacOs, Windows, Linux }

internal fun detectSecretPlatform(): SecretPlatform = when {
    isMacOs() -> SecretPlatform.MacOs
    isWindows() -> SecretPlatform.Windows
    else -> SecretPlatform.Linux
}

internal class DesktopSecretStore(
    private val commandRunner: SecretCommandRunner = ProcessSecretCommandRunner,
    private val platform: SecretPlatform = detectSecretPlatform(),
    private val secretsDir: Path = defaultSecretsDir(),
) : SecretStore {
    override suspend fun read(key: SecretKey): String =
        when (platform) {
            SecretPlatform.MacOs -> readMacSecret(key)
            SecretPlatform.Windows -> readWindowsSecret(key)
            SecretPlatform.Linux -> readLinuxSecret(key)
        }

    override suspend fun write(key: SecretKey, value: String) {
        if (value.isBlank()) {
            delete(key)
            return
        }

        when (platform) {
            SecretPlatform.MacOs -> writeMacSecret(key, value)
            SecretPlatform.Windows -> writeWindowsSecret(key, value)
            SecretPlatform.Linux -> writeLinuxSecret(key, value)
        }
    }

    override suspend fun delete(key: SecretKey) {
        when (platform) {
            SecretPlatform.MacOs -> deleteMacSecret(key)
            SecretPlatform.Windows -> deleteWindowsSecret(key)
            SecretPlatform.Linux -> deleteLinuxSecret(key)
        }
    }

    private fun readMacSecret(key: SecretKey): String =
        commandRunner.run(
            listOf(
                "security",
                "find-generic-password",
                "-a",
                KeychainAccount,
                "-s",
                key.service,
                "-w",
            ),
        ).takeIf { it.exitCode == 0 }?.stdout.orEmpty().trim()

    private fun writeMacSecret(key: SecretKey, value: String) {
        commandRunner.run(
            listOf(
                "security",
                "add-generic-password",
                "-a",
                KeychainAccount,
                "-s",
                key.service,
                "-w",
                value,
                "-U",
            ),
        )
    }

    private fun deleteMacSecret(key: SecretKey) {
        commandRunner.run(
            listOf(
                "security",
                "delete-generic-password",
                "-a",
                KeychainAccount,
                "-s",
                key.service,
            ),
        )
    }

    private fun readLinuxSecret(key: SecretKey): String =
        commandRunner.run(
            listOf("secret-tool", "lookup", "application", "timeboxxing", "service", key.service),
        ).takeIf { it.exitCode == 0 }?.stdout.orEmpty().trim()

    private fun writeLinuxSecret(key: SecretKey, value: String) {
        commandRunner.run(
            listOf("secret-tool", "store", "--label", "Timeboxxing ${key.name}", "application", "timeboxxing", "service", key.service),
            stdin = value,
        )
    }

    private fun deleteLinuxSecret(key: SecretKey) {
        commandRunner.run(
            listOf("secret-tool", "clear", "application", "timeboxxing", "service", key.service),
        )
    }

    private fun readWindowsSecret(key: SecretKey): String {
        val file = windowsSecretFile(key).toFile()
        if (!file.exists()) return ""
        // The path is passed via an environment variable rather than a trailing argument:
        // `powershell -Command "<script>"` does NOT bind trailing args to $args, so $args[0]
        // would be $null. $env:TIMEBOXXING_SECRET_FILE is unambiguous across PowerShell 5.1/7.
        val script = """
            if (Test-Path -LiteralPath ${'$'}env:$SecretFileEnvVar) {
              ${'$'}secure = Get-Content -Raw -LiteralPath ${'$'}env:$SecretFileEnvVar | ConvertTo-SecureString
              ${'$'}bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR(${'$'}secure)
              try { [Runtime.InteropServices.Marshal]::PtrToStringBSTR(${'$'}bstr) }
              finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR(${'$'}bstr) }
            }
        """.trimIndent()
        return commandRunner.run(
            listOf("powershell", "-NoProfile", "-Command", script),
            env = mapOf(SecretFileEnvVar to file.absolutePath),
        )
            .takeIf { it.exitCode == 0 }
            ?.stdout
            .orEmpty()
            .trim()
    }

    private fun writeWindowsSecret(key: SecretKey, value: String) {
        val file = windowsSecretFile(key)
        file.parent.createDirectories()
        val script = """
            ${'$'}secret = [Console]::In.ReadToEnd()
            ConvertTo-SecureString ${'$'}secret -AsPlainText -Force |
              ConvertFrom-SecureString |
              Set-Content -NoNewline -LiteralPath ${'$'}env:$SecretFileEnvVar
        """.trimIndent()
        val result = commandRunner.run(
            listOf("powershell", "-NoProfile", "-Command", script),
            stdin = value,
            env = mapOf(SecretFileEnvVar to file.toFile().absolutePath),
        )
        // Surface real failures instead of silently reporting success — DesktopSettingsRepository
        // runs inside the ViewModel's runCatching, which renders this as the settings error banner.
        check(result.exitCode == 0) {
            "Failed to store ${key.name} secret: " +
                result.stderr.trim().ifBlank { "powershell exited ${result.exitCode}" }
        }
    }

    private fun deleteWindowsSecret(key: SecretKey) {
        windowsSecretFile(key).toFile().delete()
    }

    private fun windowsSecretFile(key: SecretKey) =
        secretsDir.resolve("${key.service}.dpapi")
}

private fun defaultSecretsDir(): Path =
    File(System.getProperty("user.home")).toPath()
        .resolve(".timeboxxing")
        .resolve("secrets")

internal data class SecretCommandResult(
    val exitCode: Int,
    val stdout: String,
    val stderr: String,
)

internal interface SecretCommandRunner {
    fun run(command: List<String>, stdin: String? = null, env: Map<String, String> = emptyMap()): SecretCommandResult
}

internal object ProcessSecretCommandRunner : SecretCommandRunner {
    override fun run(command: List<String>, stdin: String?, env: Map<String, String>): SecretCommandResult {
        val builder = ProcessBuilder(command).redirectErrorStream(false)
        if (env.isNotEmpty()) builder.environment().putAll(env)
        val process = builder.start()
        if (stdin != null) {
            process.outputStream.bufferedWriter().use { writer ->
                writer.write(stdin)
                writer.newLine()
            }
        } else {
            process.outputStream.close()
        }
        val stdout = process.inputStream.bufferedReader().readText()
        val stderr = process.errorStream.bufferedReader().readText()
        val exitCode = process.waitFor()
        return SecretCommandResult(exitCode, stdout, stderr)
    }
}

private const val KeychainAccount = "api-key"
private const val SecretFileEnvVar = "TIMEBOXXING_SECRET_FILE"

private fun isMacOs(): Boolean =
    System.getProperty("os.name").startsWith("Mac", ignoreCase = true)

internal fun isWindows(): Boolean =
    System.getProperty("os.name").startsWith("Windows", ignoreCase = true)
