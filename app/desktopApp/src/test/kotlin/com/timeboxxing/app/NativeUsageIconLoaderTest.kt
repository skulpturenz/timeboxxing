package com.timeboxxing.app

import androidx.compose.ui.graphics.ImageBitmap
import com.timeboxxing.app.model.UsageApplicationIdentity
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.runBlocking
import java.awt.image.BufferedImage
import java.io.File
import java.nio.file.Files
import javax.imageio.ImageIO
import javax.swing.Icon
import javax.swing.ImageIcon
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

class NativeUsageIconLoaderTest {
    @Test
    fun cachesMissingIconsByApplicationIdentity() = runBlocking {
        val source = RecordingNativeIconSource()
        val loader = NativeUsageIconLoader(
            iconSource = source,
            dispatcher = Dispatchers.Unconfined,
        )
        val identity = UsageApplicationIdentity(
            name = "Google Chrome",
            identifier = "com.google.Chrome",
            path = "/Applications/Google Chrome.app",
        )

        loader.loadIcon(identity)
        loader.loadIcon(identity.copy(name = "google chrome"))

        assertEquals(1, source.calls)
    }

    @Test
    fun macOSResolverReadsAppBundleIconBeforeSwingFallback() {
        val tempDir = newTempDir()
        val appPath = tempDir.resolve("System Settings.app")
        val resourcesDir = appPath.resolve("Contents/Resources").apply { mkdirs() }
        appPath.resolve("Contents/Info.plist").writeText("not parsed by fake runner")
        resourcesDir.resolve("SystemSettings.icns").writeBytes(byteArrayOf(1, 2, 3))
        val commandRunner = RecordingIconCommandRunner { command ->
            when (command.first()) {
                "/usr/libexec/PlistBuddy" -> IconCommandResult(0, "SystemSettings\n", "")
                "/usr/bin/sips" -> {
                    writeTestPng(File(command.last()))
                    IconCommandResult(0, "", "")
                }
                else -> IconCommandResult(1, "", "unexpected")
            }
        }
        val systemIconReader = RecordingSystemIconReader()
        val source = DefaultNativeIconSource(
            osName = "Mac OS X",
            homeDir = tempDir,
            macOSIconResolver = MacOSIconResolver(
                commandRunner = commandRunner,
                cacheRoot = tempDir.resolve("cache"),
            ),
            systemIconReader = systemIconReader,
        )

        val icon = source.loadIcon(
            UsageApplicationIdentity(
                name = "Example",
                identifier = "com.example.Example",
                path = appPath.absolutePath,
            ),
        )

        assertTrue(icon != null)
        assertEquals(emptyList(), systemIconReader.calls)
        assertEquals("/usr/libexec/PlistBuddy", commandRunner.commands[0].first())
        assertEquals("/usr/bin/sips", commandRunner.commands[1].first())
    }

    @Test
    fun macOSResolverUsesNativeExecutableIconBeforeSwingFallback() {
        val tempDir = newTempDir()
        val executablePath = tempDir.resolve("helper").apply {
            writeText("#!/bin/sh\n")
            setExecutable(true)
        }
        val commandRunner = RecordingIconCommandRunner { command ->
            if (command.first() == "/usr/bin/osascript") {
                writeTestPng(File(command.last()))
                IconCommandResult(0, "ok\n", "")
            } else {
                IconCommandResult(1, "", "unexpected")
            }
        }
        val systemIconReader = RecordingSystemIconReader()
        val source = DefaultNativeIconSource(
            osName = "Mac OS X",
            homeDir = tempDir,
            macOSIconResolver = MacOSIconResolver(
                commandRunner = commandRunner,
                cacheRoot = tempDir.resolve("cache"),
            ),
            systemIconReader = systemIconReader,
        )

        val icon = source.loadIcon(
            UsageApplicationIdentity(
                name = "helper",
                identifier = "helper",
                path = executablePath.absolutePath,
            ),
        )

        assertTrue(icon != null)
        assertEquals(emptyList(), systemIconReader.calls)
        assertEquals("/usr/bin/osascript", commandRunner.commands.single().first())
    }

    @Test
    fun macOSResolverFallsBackWhenNativeExecutableExportIsTransparent() {
        val tempDir = newTempDir()
        val executablePath = tempDir.resolve("helper").apply {
            writeText("#!/bin/sh\n")
            setExecutable(true)
        }
        val commandRunner = RecordingIconCommandRunner { command ->
            if (command.first() == "/usr/bin/osascript") {
                writeTransparentPng(File(command.last()))
                IconCommandResult(0, "ok\n", "")
            } else {
                IconCommandResult(1, "", "unexpected")
            }
        }
        val systemIconReader = RecordingSystemIconReader(unsizedIcon = testIcon())
        val source = DefaultNativeIconSource(
            osName = "Mac OS X",
            homeDir = tempDir,
            macOSIconResolver = MacOSIconResolver(
                commandRunner = commandRunner,
                cacheRoot = tempDir.resolve("cache"),
            ),
            systemIconReader = systemIconReader,
        )

        val icon = source.loadIcon(
            UsageApplicationIdentity(
                name = "helper",
                identifier = "helper",
                path = executablePath.absolutePath,
            ),
        )

        assertTrue(icon != null)
        assertEquals(listOf("unsized"), systemIconReader.calls)
    }

    @Test
    fun macOSResolverUsesJavaLauncherBundleForJavaIdentity() {
        val tempDir = newTempDir()
        val javaLauncherPath = tempDir.resolve("JavaLauncher.app")
        val resourcesDir = javaLauncherPath.resolve("Contents/Resources").apply { mkdirs() }
        javaLauncherPath.resolve("Contents/Info.plist").writeText("not parsed by fake runner")
        resourcesDir.resolve("JavaLauncher.icns").writeBytes(byteArrayOf(1, 2, 3))
        val javaExecutablePath = tempDir.resolve("java").apply {
            writeText("#!/bin/sh\n")
            setExecutable(true)
        }
        val commandRunner = RecordingIconCommandRunner { command ->
            when (command.first()) {
                "/usr/libexec/PlistBuddy" -> IconCommandResult(0, "JavaLauncher.icns\n", "")
                "/usr/bin/sips" -> {
                    writeTestPng(File(command.last()))
                    IconCommandResult(0, "", "")
                }
                else -> IconCommandResult(1, "", "unexpected")
            }
        }
        val systemIconReader = RecordingSystemIconReader()
        val source = DefaultNativeIconSource(
            osName = "Mac OS X",
            homeDir = tempDir,
            macOSJavaLauncherPath = javaLauncherPath,
            macOSIconResolver = MacOSIconResolver(
                commandRunner = commandRunner,
                cacheRoot = tempDir.resolve("cache"),
            ),
            systemIconReader = systemIconReader,
        )

        val icon = source.loadIcon(
            UsageApplicationIdentity(
                name = "java",
                identifier = "java",
                path = javaExecutablePath.absolutePath,
            ),
        )

        assertTrue(icon != null)
        assertEquals(emptyList(), systemIconReader.calls)
        assertEquals(listOf("/usr/libexec/PlistBuddy", "/usr/bin/sips"), commandRunner.commands.map { it.first() })
    }

    @Test
    fun missingMacOSBundleIconFallsBackToNoIcon() {
        val tempDir = newTempDir()
        val appPath = tempDir.resolve("Unknown.app").apply { mkdirs() }
        val commandRunner = RecordingIconCommandRunner {
            IconCommandResult(1, "", "missing")
        }
        val systemIconReader = RecordingSystemIconReader()
        val source = DefaultNativeIconSource(
            osName = "Mac OS X",
            homeDir = tempDir,
            macOSIconResolver = MacOSIconResolver(
                commandRunner = commandRunner,
                cacheRoot = tempDir.resolve("cache"),
            ),
            systemIconReader = systemIconReader,
        )

        val icon = source.loadIcon(
            UsageApplicationIdentity(
                name = "Unknown",
                identifier = "com.example.Unknown",
                path = appPath.absolutePath,
            ),
        )

        assertNull(icon)
        assertEquals(emptyList(), systemIconReader.calls)
    }

    @Test
    fun windowsResolverRequestsSizedSystemIconBeforeUnsizedFallback() {
        val tempDir = newTempDir()
        val executablePath = tempDir.resolve("notepad.exe").apply { writeText("binary") }
        val systemIconReader = RecordingSystemIconReader(sizedIcon = testIcon())
        val source = DefaultNativeIconSource(
            osName = "Windows 11",
            homeDir = tempDir,
            systemIconReader = systemIconReader,
        )

        val icon = source.loadIcon(
            UsageApplicationIdentity(
                name = "Notepad",
                identifier = "notepad",
                path = executablePath.absolutePath,
            ),
        )

        assertTrue(icon != null)
        assertEquals(listOf("sized:256x256"), systemIconReader.calls)
    }

    @Test
    fun windowsResolverReturnsNullWhenSystemIconsAreTransparent() {
        val tempDir = newTempDir()
        val executablePath = tempDir.resolve("helper.exe").apply { writeText("binary") }
        val systemIconReader = RecordingSystemIconReader(
            sizedIcon = testIcon(transparent = true),
            unsizedIcon = testIcon(transparent = true),
        )
        val source = DefaultNativeIconSource(
            osName = "Windows 11",
            homeDir = tempDir,
            systemIconReader = systemIconReader,
        )

        val icon = source.loadIcon(
            UsageApplicationIdentity(
                name = "Helper",
                identifier = "helper",
                path = executablePath.absolutePath,
            ),
        )

        assertNull(icon)
        assertEquals(listOf("sized:256x256", "unsized"), systemIconReader.calls)
    }

    @Test
    fun linuxResolverMatchesDesktopEntryByExecutablePath() {
        val tempDir = newTempDir()
        val applicationsDir = tempDir.resolve("applications").apply { mkdirs() }
        val pixmapsDir = tempDir.resolve("pixmaps").apply { mkdirs() }
        val iconFile = pixmapsDir.resolve("example.png")
        writeTestPng(iconFile)
        applicationsDir.resolve("example.desktop").writeText(
            """
            [Desktop Entry]
            Name=Example
            Exec=/opt/example/bin/example %U
            Icon=example
            """.trimIndent(),
        )
        val resolver = LinuxDesktopIconResolver(
            desktopEntryRoots = listOf(applicationsDir),
            iconThemeRoots = emptyList(),
            pixmapRoots = listOf(pixmapsDir),
        )

        val icon = resolver.loadIcon(
            UsageApplicationIdentity(
                name = "Example",
                identifier = "example",
                path = "/opt/example/bin/example",
            ),
        )

        assertTrue(icon != null)
    }

    @Test
    fun linuxResolverMatchesStartupWMClassAndSvgThemeIcon() {
        val tempDir = newTempDir()
        val applicationsDir = tempDir.resolve("applications").apply { mkdirs() }
        val iconThemeRoot = tempDir.resolve("icons")
        val scalableDir = iconThemeRoot.resolve("hicolor/scalable/apps").apply { mkdirs() }
        writeTestSvg(scalableDir.resolve("jetbrains-idea.svg"))
        applicationsDir.resolve("random.desktop").writeText(
            """
            [Desktop Entry]
            Name=Different
            StartupWMClass=jetbrains-idea
            Exec=/opt/idea/bin/idea %F
            Icon=jetbrains-idea
            """.trimIndent(),
        )
        val resolver = LinuxDesktopIconResolver(
            desktopEntryRoots = listOf(applicationsDir),
            iconThemeRoots = listOf(iconThemeRoot),
            pixmapRoots = emptyList(),
        )

        val icon = resolver.loadIcon(
            UsageApplicationIdentity(
                name = "Idea",
                identifier = "jetbrains-idea",
                path = "/opt/idea/bin/idea",
            ),
        )

        assertTrue(icon != null)
    }

    @Test
    fun linuxResolverMatchesWrappedFlatpakExecCommandFromSnapRoot() {
        val tempDir = newTempDir()
        val snapApplicationsDir = tempDir.resolve("snap-applications").apply { mkdirs() }
        val pixmapsDir = tempDir.resolve("pixmaps").apply { mkdirs() }
        writeTestPng(pixmapsDir.resolve("wrapped.png"))
        snapApplicationsDir.resolve("random.desktop").writeText(
            """
            [Desktop Entry]
            Name=Different
            Exec=env FOO=bar flatpak run --branch=stable com.example.Wrapped %U
            Icon=wrapped
            """.trimIndent(),
        )
        val resolver = LinuxDesktopIconResolver(
            desktopEntryRoots = listOf(snapApplicationsDir),
            iconThemeRoots = emptyList(),
            pixmapRoots = listOf(pixmapsDir),
        )

        val icon = resolver.loadIcon(
            UsageApplicationIdentity(
                name = "Process",
                identifier = "wrapped",
                path = "/tmp/process",
            ),
        )

        assertTrue(icon != null)
    }
}

private class RecordingNativeIconSource : NativeIconSource {
    var calls = 0

    override fun loadIcon(identity: UsageApplicationIdentity): ImageBitmap? {
        calls += 1
        return null
    }
}

private class RecordingIconCommandRunner(
    private val handler: (List<String>) -> IconCommandResult,
) : IconCommandRunner {
    val commands = mutableListOf<List<String>>()

    override fun run(command: List<String>, timeoutSeconds: Long): IconCommandResult {
        commands += command
        return handler(command)
    }
}

private class RecordingSystemIconReader(
    private val unsizedIcon: Icon? = null,
    private val sizedIcon: Icon? = null,
) : SystemIconReader {
    val calls = mutableListOf<String>()

    override fun readIcon(file: File): Icon? {
        calls += "unsized"
        return unsizedIcon
    }

    override fun readSizedIcon(file: File, width: Int, height: Int): Icon? {
        calls += "sized:${width}x$height"
        return sizedIcon
    }
}

private fun newTempDir(): File =
    Files.createTempDirectory("timeboxxing-icon-test").toFile()

private fun testIcon(transparent: Boolean = false): Icon =
    ImageIcon(
        BufferedImage(1, 1, BufferedImage.TYPE_INT_ARGB).apply {
            setRGB(0, 0, if (transparent) 0x00000000 else 0xFF336699.toInt())
        },
    )

private fun writeTestPng(file: File) {
    val image = BufferedImage(1, 1, BufferedImage.TYPE_INT_ARGB)
    image.setRGB(0, 0, 0xFF336699.toInt())
    ImageIO.write(image, "png", file)
}

private fun writeTestSvg(file: File) {
    file.writeText(
        """
        <svg xmlns="http://www.w3.org/2000/svg" width="256" height="256" viewBox="0 0 256 256">
          <rect width="256" height="256" fill="#336699"/>
        </svg>
        """.trimIndent(),
    )
}

private fun writeTransparentPng(file: File) {
    val image = BufferedImage(1, 1, BufferedImage.TYPE_INT_ARGB)
    image.setRGB(0, 0, 0x00000000)
    ImageIO.write(image, "png", file)
}
