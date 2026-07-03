package com.timeboxxing.app

import kotlinx.coroutines.runBlocking
import java.nio.file.Files
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue

class DesktopSecretStoreTest {
    @Test
    fun windowsWritePassesPathViaEnvAndValueViaStdin() = runBlocking {
        val runner = RecordingSecretCommandRunner()
        val dir = Files.createTempDirectory("secret-store-test")
        val store = DesktopSecretStore(runner, SecretPlatform.Windows, dir)

        store.write(SecretKey.OpenRouter, "or-key")

        val call = runner.calls.single()
        // No trailing path argument — powershell -Command does not bind trailing args to $args.
        assertEquals(listOf("powershell", "-NoProfile", "-Command"), call.command.dropLast(1))
        val script = call.command.last()
        assertTrue("\$env:$SECRET_FILE_ENV" in script, "script should read the path from the env var")
        assertTrue("\$args" !in script, "script must not depend on \$args")
        assertEquals("or-key", call.stdin)
        assertEquals(
            dir.resolve("com.timeboxxing.app.openrouter.dpapi").toFile().absolutePath,
            call.env[SECRET_FILE_ENV],
        )
    }

    @Test
    fun windowsWriteThrowsOnNonZeroExit() = runBlocking {
        val runner = RecordingSecretCommandRunner(
            result = SecretCommandResult(exitCode = 1, stdout = "", stderr = "boom"),
        )
        val dir = Files.createTempDirectory("secret-store-test")
        val store = DesktopSecretStore(runner, SecretPlatform.Windows, dir)

        val error = assertFailsWith<IllegalStateException> {
            runBlocking { store.write(SecretKey.OpenRouter, "or-key") }
        }
        assertTrue("boom" in (error.message ?: ""))
    }

    @Test
    fun windowsReadPassesPathViaEnvAndTrimsOutput() = runBlocking {
        val runner = RecordingSecretCommandRunner(
            result = SecretCommandResult(exitCode = 0, stdout = "or-key\r\n", stderr = ""),
        )
        val dir = Files.createTempDirectory("secret-store-test")
        val file = dir.resolve("com.timeboxxing.app.openrouter.dpapi")
        Files.writeString(file, "cipher") // read() short-circuits unless the file exists
        val store = DesktopSecretStore(runner, SecretPlatform.Windows, dir)

        val value = store.read(SecretKey.OpenRouter)

        assertEquals("or-key", value)
        assertEquals(file.toFile().absolutePath, runner.calls.single().env[SECRET_FILE_ENV])
    }

    @Test
    fun windowsReadReturnsEmptyWhenFileMissing() = runBlocking {
        val runner = RecordingSecretCommandRunner()
        val dir = Files.createTempDirectory("secret-store-test")
        val store = DesktopSecretStore(runner, SecretPlatform.Windows, dir)

        assertEquals("", store.read(SecretKey.OpenRouter))
        assertTrue(runner.calls.isEmpty(), "no process should launch when the secret file is absent")
    }

    @Test
    fun blankValueDeletesInsteadOfWriting() = runBlocking {
        val runner = RecordingSecretCommandRunner()
        val dir = Files.createTempDirectory("secret-store-test")
        val file = dir.resolve("com.timeboxxing.app.openrouter.dpapi")
        Files.writeString(file, "cipher")
        val store = DesktopSecretStore(runner, SecretPlatform.Windows, dir)

        store.write(SecretKey.OpenRouter, "   ")

        assertTrue(runner.calls.isEmpty(), "delete on Windows removes the file without a process")
        assertTrue(!file.toFile().exists(), "secret file should be deleted")
    }
}

private const val SECRET_FILE_ENV = "TIMEBOXXING_SECRET_FILE"

private data class RecordedCall(
    val command: List<String>,
    val stdin: String?,
    val env: Map<String, String>,
)

private class RecordingSecretCommandRunner(
    private val result: SecretCommandResult = SecretCommandResult(exitCode = 0, stdout = "", stderr = ""),
) : SecretCommandRunner {
    val calls = mutableListOf<RecordedCall>()

    override fun run(command: List<String>, stdin: String?, env: Map<String, String>): SecretCommandResult {
        calls += RecordedCall(command, stdin, env)
        return result
    }
}
