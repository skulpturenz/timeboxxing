package com.timeboxxing.app

import java.io.File
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
}

internal class DesktopSecretStore(
    private val commandRunner: SecretCommandRunner = ProcessSecretCommandRunner,
) : SecretStore {
    override suspend fun read(key: SecretKey): String =
        when {
            isMacOs() -> readMacSecret(key)
            isWindows() -> readWindowsSecret(key)
            else -> readLinuxSecret(key)
        }

    override suspend fun write(key: SecretKey, value: String) {
        if (value.isBlank()) {
            delete(key)
            return
        }

        when {
            isMacOs() -> writeMacSecret(key, value)
            isWindows() -> writeWindowsSecret(key, value)
            else -> writeLinuxSecret(key, value)
        }
    }

    override suspend fun delete(key: SecretKey) {
        when {
            isMacOs() -> deleteMacSecret(key)
            isWindows() -> deleteWindowsSecret(key)
            else -> deleteLinuxSecret(key)
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
        val script = """
            if (Test-Path -LiteralPath ${'$'}args[0]) {
              ${'$'}secure = Get-Content -Raw -LiteralPath ${'$'}args[0] | ConvertTo-SecureString
              ${'$'}bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR(${'$'}secure)
              try { [Runtime.InteropServices.Marshal]::PtrToStringBSTR(${'$'}bstr) }
              finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR(${'$'}bstr) }
            }
        """.trimIndent()
        return commandRunner.run(listOf("powershell", "-NoProfile", "-Command", script, file.absolutePath))
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
              Set-Content -NoNewline -LiteralPath ${'$'}args[0]
        """.trimIndent()
        commandRunner.run(listOf("powershell", "-NoProfile", "-Command", script, file.toFile().absolutePath), stdin = value)
    }

    private fun deleteWindowsSecret(key: SecretKey) {
        windowsSecretFile(key).toFile().delete()
    }

    private fun windowsSecretFile(key: SecretKey) =
        File(System.getProperty("user.home")).toPath()
            .resolve(".timeboxxing")
            .resolve("secrets")
            .resolve("${key.service}.dpapi")
}

internal data class SecretCommandResult(
    val exitCode: Int,
    val stdout: String,
    val stderr: String,
)

internal interface SecretCommandRunner {
    fun run(command: List<String>, stdin: String? = null): SecretCommandResult
}

internal object ProcessSecretCommandRunner : SecretCommandRunner {
    override fun run(command: List<String>, stdin: String?): SecretCommandResult {
        val process = ProcessBuilder(command)
            .redirectErrorStream(false)
            .start()
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

private fun isMacOs(): Boolean =
    System.getProperty("os.name").startsWith("Mac", ignoreCase = true)

internal fun isWindows(): Boolean =
    System.getProperty("os.name").startsWith("Windows", ignoreCase = true)
