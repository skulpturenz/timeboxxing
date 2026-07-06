package com.timeboxxing.app

import com.github.jknack.handlebars.Handlebars
import com.github.jknack.handlebars.Helper
import com.timeboxxing.app.presentation.EntriesPdfRenderer
import com.timeboxxing.app.presentation.PdfRenderStage
import com.microsoft.playwright.Playwright
import com.microsoft.playwright.impl.driver.Driver
import com.microsoft.playwright.options.LoadState
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.longOrNull
import java.io.File

/**
 * Renders the entries export model through a Handlebars template and prints the resulting HTML to a
 * PDF via a headless Chromium (Playwright). The template author uses TailwindCSS utility classes;
 * `{{{tailwindRuntime}}}` inlines the vendored Tailwind browser runtime so styling works offline.
 *
 * Only Chromium is downloaded (Playwright's auto-installer would otherwise pull Firefox + WebKit
 * too), and only once, into [browsersDir] — a stable, app-controlled location. That first-run
 * download is reported via [onProgress] with a determinate fraction so the UI can show a real
 * progress bar.
 */
internal class DesktopEntriesPdfRenderer(
    private val browsersDir: File,
) : EntriesPdfRenderer {
    // Lazily initialized so constructing this renderer at app startup does no work and cannot throw
    // (e.g. from a missing bundled resource); any such failure surfaces only when a PDF is exported.
    private val handlebars: Handlebars by lazy {
        Handlebars().apply {
            registerHelper("tailwindRuntime", Helper<Any?> { _, _ -> Handlebars.SafeString(tailwindScriptTag) })
        }
    }

    override suspend fun render(
        modelJson: String,
        template: String?,
        onProgress: (stage: PdfRenderStage, fraction: Float?) -> Unit,
    ): ByteArray {
        val source = template?.takeIf { it.isNotBlank() } ?: defaultTemplate()
        val context = Json.parseToJsonElement(modelJson).toJavaValue()
        val html = handlebars.compileInline(source).apply(context)
        return withContext(Dispatchers.IO) { htmlToPdf(html, onProgress) }
    }

    override fun defaultTemplate(): String = defaultTemplateSource

    private fun htmlToPdf(html: String, onProgress: (PdfRenderStage, Float?) -> Unit): ByteArray {
        val browsersEnv = mapOf("PLAYWRIGHT_BROWSERS_PATH" to browsersDir.absolutePath)
        try {
            if (!isChromiumInstalled()) {
                installChromium(browsersEnv, onProgress)
            }
            // Chromium is present now, so tell Playwright not to auto-install the full browser set.
            val createOptions = Playwright.CreateOptions().setEnv(
                browsersEnv + ("PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD" to "1"),
            )
            Playwright.create(createOptions).use { playwright ->
                playwright.chromium().launch().use { browser ->
                    onProgress(PdfRenderStage.Rendering, null)
                    val page = browser.newPage()
                    page.setContent(html)
                    page.waitForLoadState(LoadState.NETWORKIDLE)
                    // Give the Tailwind runtime a moment to observe the DOM and inject styles.
                    page.waitForTimeout(300.0)
                    return page.pdf(
                        com.microsoft.playwright.Page.PdfOptions()
                            .setFormat("A4")
                            .setPrintBackground(true),
                    )
                }
            }
        } catch (error: Exception) {
            throw IllegalStateException(
                "Could not render the PDF. The one-time browser download may have failed — check your connection and try again. (${error.message})",
                error,
            )
        }
    }

    /**
     * Installs only Chromium (via the Playwright driver CLI) into [browsersDir], parsing the CLI's
     * download output to report a determinate progress fraction. Runs the driver as a subprocess so
     * it never calls System.exit on our JVM.
     */
    private fun installChromium(browsersEnv: Map<String, String>, onProgress: (PdfRenderStage, Float?) -> Unit) {
        onProgress(PdfRenderStage.DownloadingBrowser, null) // indeterminate until the first % arrives
        val driver = Driver.ensureDriverInstalled(browsersEnv, false)
        val process = driver.createProcessBuilder().apply {
            command().addAll(listOf("install", "chromium"))
            environment()["PLAYWRIGHT_BROWSERS_PATH"] = browsersDir.absolutePath
            redirectErrorStream(true)
        }.start()

        // `install chromium` downloads a few components (Chromium, its headless shell, ffmpeg), each
        // reporting its own 0->100%. Fold them into one monotonic bar so it doesn't sawtooth.
        var componentIndex = 0
        process.inputStream.bufferedReader().use { reader ->
            var line = reader.readLine()
            while (line != null) {
                if (downloadingHeaderPattern.containsMatchIn(line) && !line.contains('%')) {
                    componentIndex++
                }
                percentPattern.find(line)?.groupValues?.get(1)?.toIntOrNull()?.let { pct ->
                    if (componentIndex > 0) {
                        val overall = ((componentIndex - 1 + pct.coerceIn(0, 100) / 100f) / ExpectedDownloadComponents)
                            .coerceIn(0f, 1f)
                        onProgress(PdfRenderStage.DownloadingBrowser, overall)
                    }
                }
                line = reader.readLine()
            }
        }
        val exitCode = process.waitFor()
        if (exitCode != 0) {
            throw IllegalStateException("Browser download failed (exit $exitCode).")
        }
    }

    private fun isChromiumInstalled(): Boolean =
        browsersDir.listFiles()?.any { it.isDirectory && it.name.startsWith("chromium-") } == true

    private companion object {
        val defaultTemplateSource: String by lazy { readResource("export/default-template.hbs") }
        val tailwindScriptTag: String by lazy { "<script>${readResource("export/tailwind-runtime.js")}</script>" }
        val percentPattern = Regex("""(\d{1,3})\s*%""")
        val downloadingHeaderPattern = Regex("""(?i)\bDownloading\b""")

        // `playwright install chromium` fetches Chromium, its headless shell, and ffmpeg. Used only to
        // fold their separate progress reports into one bar; extra/fewer components degrade gracefully.
        const val ExpectedDownloadComponents = 3f

        fun readResource(path: String): String {
            val stream = DesktopEntriesPdfRenderer::class.java.classLoader.getResourceAsStream(path)
                ?: throw IllegalStateException("Bundled resource missing: $path")
            return stream.bufferedReader().use { it.readText() }
        }
    }
}

private fun JsonElement.toJavaValue(): Any? = when (this) {
    is JsonNull -> null
    is JsonObject -> entries.associate { (key, value) -> key to value.toJavaValue() }
    is JsonArray -> map { it.toJavaValue() }
    is JsonPrimitive -> when {
        isString -> content
        booleanOrNull != null -> booleanOrNull
        longOrNull != null -> longOrNull
        doubleOrNull != null -> doubleOrNull
        else -> content
    }
}
