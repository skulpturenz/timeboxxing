package com.timeboxxing.app.ui.fonts

import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.platform.Font
import java.io.File
import java.net.URI
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

private const val InterFontUrl =
    "https://raw.githubusercontent.com/google/fonts/main/ofl/inter/Inter%5Bopsz,wght%5D.ttf"
private const val InterFontFileName = "Inter-opsz-wght.ttf"

fun interface InterFontDownloader {
    fun download(url: String): ByteArray
}

object UrlInterFontDownloader : InterFontDownloader {
    override fun download(url: String): ByteArray =
        URI.create(url).toURL().openStream().use { it.readBytes() }
}

object InterFontLoader {
    suspend fun loadFontFamily(
        cacheDirectory: File = defaultInterFontCacheDirectory(),
        downloader: InterFontDownloader = UrlInterFontDownloader,
    ): FontFamily? {
        val fontFile = ensureCachedInterFontFile(
            cacheDirectory = cacheDirectory,
            downloader = downloader,
        ) ?: return null

        return createInterFontFamily(fontFile)
    }
}

internal suspend fun ensureCachedInterFontFile(
    cacheDirectory: File,
    downloader: InterFontDownloader,
): File? = withContext(Dispatchers.IO) {
    val target = File(cacheDirectory, InterFontFileName)
    if (target.exists() && target.length() > 0L) {
        return@withContext target
    }

    runCatching {
        cacheDirectory.mkdirs()
        val bytes = downloader.download(InterFontUrl)
        if (bytes.isEmpty()) {
            return@runCatching null
        }

        val tmp = File(cacheDirectory, "$InterFontFileName.tmp")
        tmp.writeBytes(bytes)
        if (target.exists()) {
            target.delete()
        }
        tmp.copyTo(target, overwrite = true)
        tmp.delete()
        target
    }.getOrNull()
}

internal fun defaultInterFontCacheDirectory(
    osName: String = System.getProperty("os.name").orEmpty(),
    userHome: String = System.getProperty("user.home").orEmpty(),
    localAppData: String? = System.getenv("LOCALAPPDATA"),
    xdgCacheHome: String? = System.getenv("XDG_CACHE_HOME"),
): File {
    val normalizedOs = osName.lowercase()
    return when {
        normalizedOs.contains("mac") ->
            File(userHome, "Library/Caches/Timeboxxing/fonts")

        normalizedOs.contains("win") && !localAppData.isNullOrBlank() ->
            File(localAppData, "Timeboxxing/cache/fonts")

        !xdgCacheHome.isNullOrBlank() ->
            File(xdgCacheHome, "timeboxxing/fonts")

        else ->
            File(userHome, ".cache/timeboxxing/fonts")
    }
}

private fun createInterFontFamily(fontFile: File): FontFamily =
    FontFamily(
        Font(file = fontFile, weight = FontWeight.Normal),
        Font(file = fontFile, weight = FontWeight.Medium),
        Font(file = fontFile, weight = FontWeight.SemiBold),
        Font(file = fontFile, weight = FontWeight.Bold),
    )
