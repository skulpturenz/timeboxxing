package com.timeboxxing.app

import com.timeboxxing.app.presentation.AppUpdater
import com.timeboxxing.app.presentation.AvailableUpdate
import com.timeboxxing.app.presentation.UpdateCheckResult
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.IOException
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.nio.file.Files
import java.nio.file.Path
import java.time.Duration
import kotlin.io.path.absolutePathString
import kotlin.system.exitProcess

/**
 * Desktop [AppUpdater] backed by GitHub Releases. It checks the repository's latest stable release,
 * downloads the matching jpackage installer (`.dmg`/`.exe`), applies it in place via a detached
 * helper process that waits for this app to exit, then relaunches.
 *
 * Update checks are disabled for local/dev builds (see [updatesSupported]) so a developer build does
 * not spuriously report the newest published release as an available update.
 */
internal class DesktopAppUpdater(
    override val currentVersion: String,
    private val javaEnv: JavaEnv,
    private val onBeforeExit: () -> Unit = {},
) : AppUpdater {

    private val httpClient: HttpClient by lazy {
        HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(15))
            .followRedirects(HttpClient.Redirect.NORMAL)
            .build()
    }

    override suspend fun check(): UpdateCheckResult = withContext(Dispatchers.IO) {
        if (!updatesSupported()) {
            return@withContext UpdateCheckResult.Unsupported
        }

        val body = fetchLatestReleaseJson()
        val latestTag = TAG_REGEX.find(body)?.groupValues?.get(1)
            ?: throw IOException("Could not read the latest release version.")
        val latestVersion = latestTag.removePrefix("v")
        val assetUrl = assetUrlRegex()?.find(body)?.groupValues?.get(1)

        if (assetUrl == null || compareVersions(latestVersion, currentVersion) <= 0) {
            UpdateCheckResult.UpToDate
        } else {
            UpdateCheckResult.Available(
                AvailableUpdate(version = latestVersion, downloadUrl = assetUrl, notes = null),
            )
        }
    }

    override suspend fun downloadAndInstall(update: AvailableUpdate, onProgress: (Float) -> Unit) {
        val installer = withContext(Dispatchers.IO) {
            downloadInstaller(update.downloadUrl, onProgress)
        }
        // Signals the install/relaunch phase to the UI. downloadInstaller keeps progress < 1f.
        onProgress(1f)
        withContext(Dispatchers.IO) {
            launchInstallerAndExit(installer)
        }
    }

    private fun updatesSupported(): Boolean {
        if (javaEnv == JavaEnv.Local) return false
        val version = currentVersion.trim()
        if (version.isEmpty() || version == DevSentinelVersion) return false
        return currentPlatform() != Platform.Unsupported
    }

    private fun fetchLatestReleaseJson(): String {
        val request = HttpRequest.newBuilder()
            .uri(URI.create("$ReleasesApiBase/latest"))
            .header("Accept", "application/vnd.github+json")
            .header("User-Agent", UserAgent)
            .timeout(Duration.ofSeconds(20))
            .GET()
            .build()
        val response = httpClient.send(request, HttpResponse.BodyHandlers.ofString())
        if (response.statusCode() !in 200..299) {
            throw IOException("Update check failed (HTTP ${response.statusCode()}).")
        }
        return response.body()
    }

    private fun downloadInstaller(url: String, onProgress: (Float) -> Unit): Path {
        val request = HttpRequest.newBuilder()
            .uri(URI.create(url))
            .header("User-Agent", UserAgent)
            .timeout(Duration.ofMinutes(10))
            .GET()
            .build()
        val response = httpClient.send(request, HttpResponse.BodyHandlers.ofInputStream())
        if (response.statusCode() !in 200..299) {
            response.body().close()
            throw IOException("Update download failed (HTTP ${response.statusCode()}).")
        }

        val fileName = url.substringAfterLast('/').ifBlank { "timeboxxing-update" }
        val target = Files.createTempDirectory("timeboxxing-update").resolve(fileName)
        val totalBytes = response.headers().firstValueAsLong("content-length").orElse(-1L)

        response.body().use { input ->
            Files.newOutputStream(target).use { output ->
                val buffer = ByteArray(1 shl 16)
                var downloaded = 0L
                while (true) {
                    val read = input.read(buffer)
                    if (read < 0) break
                    output.write(buffer, 0, read)
                    downloaded += read
                    if (totalBytes > 0) {
                        // Cap at 0.99 so the explicit onProgress(1f) marks the install/relaunch phase.
                        onProgress((downloaded.toFloat() / totalBytes).coerceIn(0f, 0.99f))
                    }
                }
            }
        }
        return target
    }

    private fun launchInstallerAndExit(installer: Path): Nothing {
        when (currentPlatform()) {
            Platform.MacOs -> installMacOs(installer)
            Platform.Windows -> installWindows(installer)
            Platform.Unsupported -> throw IOException("Updates are not supported on this platform.")
        }
        onBeforeExit()
        exitProcess(0)
    }

    private fun installMacOs(dmg: Path) {
        val targetApp = currentMacAppBundle()
        val pid = ProcessHandle.current().pid().toString()
        val script = writeTempScript(
            name = "timeboxxing-update.sh",
            content = MacUpdateScript,
        )
        spawnDetached(
            "/bin/bash",
            script.absolutePathString(),
            pid,
            dmg.absolutePathString(),
            targetApp.absolutePathString(),
        )
    }

    private fun installWindows(installer: Path) {
        val pid = ProcessHandle.current().pid().toString()
        val relaunch = System.getProperty("jpackage.app-path")
            ?: throw IOException("Could not resolve the app launcher for relaunch.")
        val script = writeTempScript(
            name = "timeboxxing-update.ps1",
            content = WindowsUpdateScript,
        )
        spawnDetached(
            "powershell",
            "-NoProfile",
            "-ExecutionPolicy",
            "Bypass",
            "-File",
            script.absolutePathString(),
            "-AppPid",
            pid,
            "-Installer",
            installer.absolutePathString(),
            "-Relaunch",
            relaunch,
        )
    }

    /** Resolves the running `.app` bundle by walking up from the jpackage launcher path. */
    private fun currentMacAppBundle(): Path {
        val appPath = System.getProperty("jpackage.app-path")
        if (appPath != null) {
            var dir: Path? = Path.of(appPath)
            while (dir != null) {
                if (dir.fileName?.toString()?.endsWith(".app") == true) return dir
                dir = dir.parent
            }
        }
        return Path.of("/Applications/Timeboxxing.app")
    }

    private fun writeTempScript(name: String, content: String): Path {
        val script = Files.createTempDirectory("timeboxxing-update-script").resolve(name)
        Files.writeString(script, content)
        script.toFile().setExecutable(true, false)
        return script
    }

    private fun spawnDetached(vararg command: String) {
        ProcessBuilder(*command)
            .redirectOutput(ProcessBuilder.Redirect.DISCARD)
            .redirectError(ProcessBuilder.Redirect.DISCARD)
            .start()
    }

    private fun assetUrlRegex(): Regex? = when (currentPlatform()) {
        // Release assets are arch-qualified, e.g. Timeboxxing-<tag>-macOS-arm64.dmg.
        Platform.MacOs -> currentArchToken()?.let { arch ->
            Regex("\"browser_download_url\"\\s*:\\s*\"([^\"]*-macOS-$arch\\.dmg)\"")
        }
        // Windows ships amd64 only; an amd64 JVM under Windows-arm64 emulation reports os.arch=amd64.
        Platform.Windows -> Regex("\"browser_download_url\"\\s*:\\s*\"([^\"]*-Windows-amd64\\.exe)\"")
        Platform.Unsupported -> null
    }

    private enum class Platform { MacOs, Windows, Unsupported }

    private fun currentPlatform(): Platform {
        val os = System.getProperty("os.name").orEmpty().lowercase()
        return when {
            os.startsWith("mac") -> Platform.MacOs
            os.contains("windows") -> Platform.Windows
            else -> Platform.Unsupported
        }
    }

    /** Normalizes the JVM `os.arch` to the release asset arch token. Mirrors SidecarProcessManager. */
    private fun currentArchToken(): String? =
        when (System.getProperty("os.arch").orEmpty().lowercase()) {
            "aarch64", "arm64" -> "arm64"
            "x86_64", "amd64" -> "amd64"
            else -> null
        }

    /** Compares dotted numeric versions, ignoring any `-prerelease`/`+build` suffix. */
    private fun compareVersions(a: String, b: String): Int {
        val pa = versionParts(a)
        val pb = versionParts(b)
        val size = maxOf(pa.size, pb.size)
        for (i in 0 until size) {
            val diff = pa.getOrElse(i) { 0 }.compareTo(pb.getOrElse(i) { 0 })
            if (diff != 0) return diff
        }
        return 0
    }

    private fun versionParts(version: String): List<Int> =
        version.trim()
            .removePrefix("v")
            .substringBefore('-')
            .substringBefore('+')
            .split('.')
            .map { it.toIntOrNull() ?: 0 }

    private companion object {
        const val ReleasesApiBase = "https://api.github.com/repos/skulpturenz/timeboxxing/releases"
        const val UserAgent = "Timeboxxing-Updater"
        const val DevSentinelVersion = "0.0.0"
        val TAG_REGEX = Regex("\"tag_name\"\\s*:\\s*\"([^\"]+)\"")

        // Waits for this app to exit, replaces the installed .app bundle from the mounted DMG, then
        // relaunches. Falls back to an admin prompt only when the bundle location is not writable
        // (e.g. /Applications without write access); a per-user install replaces silently.
        val MacUpdateScript = """
            #!/bin/bash
            APP_PID="${'$'}1"
            DMG="${'$'}2"
            TARGET_APP="${'$'}3"

            while kill -0 "${'$'}APP_PID" 2>/dev/null; do sleep 0.5; done

            MOUNT_DIR="${'$'}(mktemp -d)"
            hdiutil attach -nobrowse -noautoopen -mountpoint "${'$'}MOUNT_DIR" "${'$'}DMG" >/dev/null
            NEW_APP="${'$'}MOUNT_DIR/Timeboxxing.app"
            TARGET_DIR="${'$'}(dirname "${'$'}TARGET_APP")"

            if [ -w "${'$'}TARGET_DIR" ]; then
              rm -rf "${'$'}TARGET_APP"
              /usr/bin/ditto "${'$'}NEW_APP" "${'$'}TARGET_APP"
            else
              /usr/bin/osascript -e "do shell script \"rm -rf '${'$'}TARGET_APP'; /usr/bin/ditto '${'$'}NEW_APP' '${'$'}TARGET_APP'\" with administrator privileges"
            fi

            hdiutil detach "${'$'}MOUNT_DIR" >/dev/null 2>&1 || true
            /usr/bin/xattr -dr com.apple.quarantine "${'$'}TARGET_APP" 2>/dev/null || true
            open "${'$'}TARGET_APP"
        """.trimIndent() + "\n"

        // Waits for this app to exit, runs the jpackage installer, then relaunches. The install is
        // per-user (no elevation) and the stable upgradeUuid makes it an in-place upgrade. '/quiet'
        // targets a silent run; if a given installer build ignores it, the wizard shows and the
        // upgrade still applies.
        val WindowsUpdateScript = """
            param(
                [int]${'$'}AppPid,
                [string]${'$'}Installer,
                [string]${'$'}Relaunch
            )

            try { Wait-Process -Id ${'$'}AppPid -ErrorAction SilentlyContinue } catch {}
            Start-Process -FilePath ${'$'}Installer -ArgumentList '/quiet' -Wait
            Start-Process -FilePath ${'$'}Relaunch
        """.trimIndent() + "\n"
    }
}
