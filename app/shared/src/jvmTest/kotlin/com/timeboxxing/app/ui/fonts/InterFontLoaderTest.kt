package com.timeboxxing.app.ui.fonts

import java.io.File
import kotlin.io.path.createTempDirectory
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertTrue
import kotlinx.coroutines.runBlocking

class InterFontLoaderTest {
    @Test
    fun cacheDirectoryUsesMacCachesFolder() {
        val dir = defaultInterFontCacheDirectory(
            osName = "Mac OS X",
            userHome = "/Users/tester",
            localAppData = null,
            xdgCacheHome = null,
        )

        assertEquals(File("/Users/tester/Library/Caches/Timeboxxing/fonts"), dir)
    }

    @Test
    fun cacheDirectoryUsesXdgCacheOnLinux() {
        val dir = defaultInterFontCacheDirectory(
            osName = "Linux",
            userHome = "/home/tester",
            localAppData = null,
            xdgCacheHome = "/tmp/xdg-cache",
        )

        assertEquals(File("/tmp/xdg-cache/timeboxxing/fonts"), dir)
    }

    @Test
    fun downloadsFontWhenCacheIsEmpty() = runBlocking {
        val cacheDir = createTempDirectory("inter-font-cache").toFile()
        var requestedUrl: String? = null

        val file = ensureCachedInterFontFile(cacheDir) { url ->
            requestedUrl = url
            byteArrayOf(1, 2, 3, 4)
        }

        assertNotNull(file)
        assertTrue(file.exists())
        assertEquals(4L, file.length())
        assertTrue(requestedUrl.orEmpty().contains("raw.githubusercontent.com/google/fonts"))
    }

    @Test
    fun usesCachedFontWithoutDownloadingAgain() = runBlocking {
        val cacheDir = createTempDirectory("inter-font-cache").toFile()
        val cached = File(cacheDir, "Inter-opsz-wght.ttf")
        cached.parentFile.mkdirs()
        cached.writeBytes(byteArrayOf(9, 8, 7))

        val file = ensureCachedInterFontFile(cacheDir) {
            error("Downloader should not be called when cached font exists.")
        }

        assertEquals(cached, file)
        assertEquals(3L, cached.length())
    }
}
