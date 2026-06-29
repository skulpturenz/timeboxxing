package com.timeboxxing.app

import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.toComposeImageBitmap
import com.timeboxxing.domain.model.UsageApplicationIdentity
import com.timeboxxing.app.ui.UsageIconLoader
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.jetbrains.skia.Data
import org.jetbrains.skia.EncodedImageFormat
import org.jetbrains.skia.Surface
import org.jetbrains.skia.svg.SVGDOM
import java.awt.image.BufferedImage
import java.io.ByteArrayInputStream
import java.io.File
import java.io.IOException
import java.security.MessageDigest
import java.util.Locale
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.TimeUnit
import javax.imageio.ImageIO
import javax.swing.Icon
import javax.swing.filechooser.FileSystemView
import kotlin.concurrent.thread
import kotlin.math.max

internal class NativeUsageIconLoader(
    private val iconSource: NativeIconSource = DefaultNativeIconSource(),
    private val dispatcher: CoroutineDispatcher = Dispatchers.IO,
) : UsageIconLoader {
    private val cache = ConcurrentHashMap<String, CachedIcon>()

    override suspend fun loadIcon(identity: UsageApplicationIdentity): ImageBitmap? =
        withContext(dispatcher) {
            cache.getOrPut(cacheKey(identity)) {
                CachedIcon(iconSource.loadIcon(identity))
            }.image
        }

    private fun cacheKey(identity: UsageApplicationIdentity): String =
        listOf(identity.identifier.orEmpty(), identity.path.orEmpty(), identity.name)
            .joinToString(separator = "\u001F") { it.trim().lowercase(Locale.ROOT) }
}

internal interface NativeIconSource {
    fun loadIcon(identity: UsageApplicationIdentity): ImageBitmap?
}

internal interface SystemIconReader {
    fun readIcon(file: File): Icon?

    fun readSizedIcon(file: File, width: Int, height: Int): Icon?
}

internal class FileSystemViewSystemIconReader(
    private val fileSystemView: FileSystemView = FileSystemView.getFileSystemView(),
) : SystemIconReader {
    override fun readIcon(file: File): Icon? =
        runCatching { fileSystemView.getSystemIcon(file) }.getOrNull()

    override fun readSizedIcon(file: File, width: Int, height: Int): Icon? =
        runCatching { fileSystemView.getSystemIcon(file, width, height) }.getOrNull()
}

private data class CachedIcon(val image: ImageBitmap?)

internal class DefaultNativeIconSource(
    private val osName: String = System.getProperty("os.name").orEmpty(),
    private val homeDir: File = File(System.getProperty("user.home").orEmpty()),
    private val fileSystemView: FileSystemView = FileSystemView.getFileSystemView(),
    private val macOSJavaLauncherPath: File = File("/System/Library/CoreServices/JavaLauncher.app"),
    private val systemIconReader: SystemIconReader = FileSystemViewSystemIconReader(fileSystemView),
    private val macOSIconResolver: MacOSIconResolver = MacOSIconResolver(),
    private val windowsIconResolver: WindowsIconResolver = WindowsIconResolver(systemIconReader),
) : NativeIconSource {
    private val linuxResolver = LinuxDesktopIconResolver(homeDir)

    override fun loadIcon(identity: UsageApplicationIdentity): ImageBitmap? {
        val normalizedOsName = osName.lowercase(Locale.ROOT)
        if (normalizedOsName.contains("mac")) {
            val macOSIcon = macOSIconResolver.loadIcon(identity, macOSCandidatePaths(identity, normalizedOsName))
            if (macOSIcon != null || identity.path.isMacOSBundlePath()) {
                return macOSIcon
            }
            return iconFromPath(identity.path)
        }
        if (normalizedOsName.contains("linux")) {
            return linuxResolver.loadIcon(identity)
                ?: iconFromPath(identity.path)
        }
        if (normalizedOsName.contains("win")) {
            return windowsIconResolver.loadIcon(identity)
        }

        return iconFromPath(identity.path)
    }

    private fun iconFromPath(path: String?): ImageBitmap? {
        val file = path?.trim()?.takeIf { it.isNotEmpty() }?.let(::File) ?: return null
        return systemIcon(file)
    }

    private fun macOSCandidatePaths(
        identity: UsageApplicationIdentity,
        normalizedOsName: String,
    ): List<File> {
        if (!normalizedOsName.contains("mac")) return emptyList()
        val appName = identity.name.trim().takeIf { it.isNotEmpty() } ?: return emptyList()
        val appNameCandidates = listOf(
            File("/Applications/$appName.app"),
            File("/System/Applications/$appName.app"),
            File("/System/Library/CoreServices/Applications/$appName.app"),
            File(homeDir, "Applications/$appName.app"),
        )
        return macOSAliasCandidatePaths(identity) + appNameCandidates
    }

    private fun macOSAliasCandidatePaths(identity: UsageApplicationIdentity): List<File> {
        val tokens = identity.matchTokens().toSet()
        return when {
            "java" in tokens -> listOf(macOSJavaLauncherPath)
            else -> emptyList()
        }
    }

    private fun systemIcon(file: File): ImageBitmap? {
        if (!file.exists()) return null
        return systemIconReader.readIcon(file)?.toImageBitmapOrNull()
    }
}

internal class WindowsIconResolver(
    private val systemIconReader: SystemIconReader = FileSystemViewSystemIconReader(),
) {
    fun loadIcon(identity: UsageApplicationIdentity): ImageBitmap? {
        val file = identity.path
            ?.trim()
            ?.takeIf { it.isNotEmpty() }
            ?.let(::File)
            ?: return null
        if (!file.exists()) return null

        return systemIconReader.readSizedIcon(file, 256, 256)?.toImageBitmapOrNull()
            ?: systemIconReader.readIcon(file)?.toImageBitmapOrNull()
    }
}

internal class MacOSIconResolver(
    private val commandRunner: IconCommandRunner = SystemIconCommandRunner,
    private val cacheRoot: File = File(System.getProperty("java.io.tmpdir"), "timeboxxing-app-icons"),
) {
    fun loadIcon(
        identity: UsageApplicationIdentity,
        appCandidates: List<File>,
    ): ImageBitmap? {
        val identityPath = identity.path
            ?.trim()
            ?.takeIf { it.isNotEmpty() }
            ?.let(::File)
        val candidates = (listOfNotNull(identityPath) + appCandidates)
            .flatMap { listOfNotNull(it, it.appBundleAncestor()) }
            .distinctBy { it.absolutePath }

        candidates
            .filter { it.isDirectory && it.name.endsWith(".app", ignoreCase = true) }
            .firstNotNullOfOrNull(::bundleIcon)
            ?.let { return it }

        return candidates
            .filter { it.exists() }
            .firstNotNullOfOrNull(::nativeFileIcon)
    }

    private fun bundleIcon(bundle: File): ImageBitmap? {
        val iconName = bundleIconName(bundle) ?: return null
        val iconFile = bundleIconFile(bundle, iconName) ?: return null
        return convertedIcns(iconFile)
    }

    private fun bundleIconName(bundle: File): String? {
        val infoPlist = File(bundle, "Contents/Info.plist")
        if (!infoPlist.isFile) return null
        val result = commandRunner.run(
            command = listOf("/usr/libexec/PlistBuddy", "-c", "Print :CFBundleIconFile", infoPlist.absolutePath),
            timeoutSeconds = 5,
        )
        return result.stdout
            .lineSequence()
            .map { it.trim() }
            .firstOrNull { it.isNotEmpty() }
            ?.takeIf { result.exitCode == 0 }
    }

    private fun bundleIconFile(bundle: File, iconName: String): File? {
        val explicit = File(iconName)
        if (explicit.isAbsolute && explicit.isFile) {
            return explicit
        }
        val fileName = if (iconName.endsWith(".icns", ignoreCase = true)) iconName else "$iconName.icns"
        return File(bundle, "Contents/Resources/$fileName").takeIf { it.isFile }
    }

    private fun convertedIcns(iconFile: File): ImageBitmap? {
        val output = cacheFile(iconFile, "icns")
        if (!output.isFile) {
            cacheRoot.mkdirs()
            val result = commandRunner.run(
                command = listOf("/usr/bin/sips", "-s", "format", "png", iconFile.absolutePath, "--out", output.absolutePath),
                timeoutSeconds = 10,
            )
            if (result.exitCode != 0) return null
        }
        return readImageBitmap(output)
    }

    private fun nativeFileIcon(file: File): ImageBitmap? {
        if (!file.exists()) return null
        val output = cacheFile(file, "nsworkspace")
        if (!output.isFile) {
            cacheRoot.mkdirs()
            val result = commandRunner.run(
                command = listOf(
                    "/usr/bin/osascript",
                    "-l",
                    "JavaScript",
                    "-e",
                    macOSIconExportScript,
                    file.absolutePath,
                    output.absolutePath,
                ),
                timeoutSeconds = 10,
            )
            if (result.exitCode != 0) return null
        }
        return readImageBitmap(output)
    }

    private fun cacheFile(source: File, kind: String): File {
        val key = listOf(kind, source.absolutePath, source.lastModified().toString(), source.length().toString())
            .joinToString(separator = "\u001F")
        return File(cacheRoot, "${sha256(key)}.png")
    }
}

internal interface IconCommandRunner {
    fun run(
        command: List<String>,
        timeoutSeconds: Long = 15,
    ): IconCommandResult
}

internal data class IconCommandResult(
    val exitCode: Int,
    val stdout: String,
    val stderr: String,
)

internal object SystemIconCommandRunner : IconCommandRunner {
    override fun run(command: List<String>, timeoutSeconds: Long): IconCommandResult {
        val process = try {
            ProcessBuilder(command).start()
        } catch (error: IOException) {
            return IconCommandResult(127, "", error.message ?: "command unavailable")
        }
        val stdout = StringBuilder()
        val stderr = StringBuilder()
        val stdoutThread = thread(isDaemon = true) {
            process.inputStream.bufferedReader().useLines { lines ->
                lines.forEach { stdout.appendLine(it) }
            }
        }
        val stderrThread = thread(isDaemon = true) {
            process.errorStream.bufferedReader().useLines { lines ->
                lines.forEach { stderr.appendLine(it) }
            }
        }
        process.outputStream.close()

        val completed = process.waitFor(timeoutSeconds, TimeUnit.SECONDS)
        if (!completed) {
            process.destroyForcibly()
            return IconCommandResult(124, stdout.toString(), "command timed out")
        }
        stdoutThread.join(1_000)
        stderrThread.join(1_000)

        return IconCommandResult(process.exitValue(), stdout.toString(), stderr.toString())
    }
}

internal class LinuxDesktopIconResolver(
    homeDir: File = File(System.getProperty("user.home").orEmpty()),
    private val desktopEntryRoots: List<File> = linuxDesktopEntryRoots(homeDir),
    private val iconThemeRoots: List<File> = linuxIconThemeRoots(homeDir),
    private val pixmapRoots: List<File> = listOf(File("/usr/share/pixmaps"), File("/usr/local/share/pixmaps")),
) {
    fun loadIcon(identity: UsageApplicationIdentity): ImageBitmap? {
        val entry = desktopEntries().firstOrNull { it.matches(identity) } ?: return null
        return entry.iconName?.let(::loadIconFile)
    }

    private fun desktopEntries(): Sequence<DesktopEntry> =
        desktopEntryRoots
            .asSequence()
            .filter { it.isDirectory }
            .flatMap { root ->
                root.walkTopDown()
                    .filter { it.isFile && it.extension.equals("desktop", ignoreCase = true) }
            }
            .mapNotNull(::parseDesktopEntry)

    private fun loadIconFile(iconName: String): ImageBitmap? {
        val trimmed = iconName.trim()
        if (trimmed.isEmpty()) return null

        val explicit = File(trimmed)
        if (explicit.isAbsolute) {
            readIconImage(explicit)?.let { return it }
        }

        return themedIconCandidates(trimmed)
            .firstNotNullOfOrNull(::readIconImage)
    }

    private fun themedIconCandidates(iconName: String): Sequence<File> = sequence {
        val extension = File(iconName).extension.lowercase(Locale.ROOT)
        val nameWithoutExtension = when (extension) {
            "png", "svg" -> iconName.dropLast(extension.length + 1)
            else -> iconName
        }
        val fileNames = if (extension.isNotEmpty()) {
            listOf(iconName)
        } else {
            listOf(iconName, "$nameWithoutExtension.png", "$nameWithoutExtension.svg")
        }.distinct()
        val sizes = listOf("1024x1024", "512x512", "256x256", "128x128", "96x96", "64x64", "48x48", "32x32", "24x24", "16x16")
        val scalableDirs = listOf("scalable/apps", "symbolic/apps")

        for (pixmapRoot in pixmapRoots) {
            for (fileName in fileNames) {
                yield(File(pixmapRoot, fileName))
            }
        }

        for (root in iconThemeRoots.filter { it.isDirectory }) {
            root.listFiles()
                ?.filter { it.isDirectory }
                ?.forEach { theme ->
                    for (size in sizes) {
                        for (fileName in fileNames) {
                            yield(File(theme, "$size/apps/$fileName"))
                        }
                    }
                    for (dir in scalableDirs) {
                        for (fileName in fileNames) {
                            yield(File(theme, "$dir/$fileName"))
                        }
                    }
                }
        }
    }

    private fun parseDesktopEntry(file: File): DesktopEntry? {
        var name: String? = null
        var exec: String? = null
        var icon: String? = null
        var startupWMClass: String? = null

        runCatching {
            file.useLines { lines ->
                lines.forEach { rawLine ->
                    val line = rawLine.trim()
                    when {
                        line.startsWith("Name=") && name == null -> name = line.substringAfter('=').trim()
                        line.startsWith("Exec=") && exec == null -> exec = line.substringAfter('=').trim()
                        line.startsWith("Icon=") && icon == null -> icon = line.substringAfter('=').trim()
                        line.startsWith("StartupWMClass=") && startupWMClass == null -> startupWMClass = line.substringAfter('=').trim()
                    }
                }
            }
        }.getOrElse { return null }

        return DesktopEntry(
            id = file.nameWithoutExtension,
            name = name,
            startupWMClass = startupWMClass,
            exec = exec,
            iconName = icon,
        )
    }
}

private data class DesktopEntry(
    val id: String,
    val name: String?,
    val startupWMClass: String?,
    val exec: String?,
    val iconName: String?,
) {
    fun matches(identity: UsageApplicationIdentity): Boolean {
        val identityTokens = identity.matchTokens()
        val entryTokens = (
            listOfNotNull(
                id.normalizedToken(),
                name?.normalizedToken(),
                startupWMClass?.normalizedToken(),
            ) + exec.orEmpty().execCommandTokens()
        ).filter { it.isNotEmpty() }.distinct()

        return identityTokens.any { identityToken ->
            entryTokens.any { entryToken ->
                entryToken.matchesToken(identityToken)
            }
        }
    }
}

private fun UsageApplicationIdentity.matchTokens(): List<String> =
    listOfNotNull(
        name.normalizedToken(),
        identifier?.normalizedToken(),
        path?.let { File(it).nameWithoutExtension.normalizedToken() },
    ).filter { it.isNotEmpty() }

private fun String.execCommandTokens(): List<String> {
    val command = replace(Regex("%[a-zA-Z]"), "").trim()
    if (command.isEmpty()) return emptyList()
    return unwrapExecCommand(shellWords(command))
        .flatMap { token ->
            listOf(
                token.normalizedToken(),
                File(token).nameWithoutExtension.normalizedToken(),
                token.substringAfterLast('.').normalizedToken(),
            )
        }
        .filter { it.isNotEmpty() }
        .distinct()
}

private fun unwrapExecCommand(tokens: List<String>): List<String> {
    var remaining = tokens.filter { it.isNotBlank() }
    if (remaining.isEmpty()) return emptyList()

    if (remaining.first().normalizedToken() == "env") {
        var index = 1
        while (index < remaining.size && (remaining[index].startsWith("-") || remaining[index].isEnvAssignment())) {
            index += 1
        }
        remaining = remaining.drop(index)
    }
    if (remaining.isEmpty()) return emptyList()

    val command = File(remaining.first()).nameWithoutExtension.normalizedToken()
    if (command == "flatpak" && remaining.getOrNull(1)?.normalizedToken() == "run") {
        val appId = remaining.drop(2).firstOrNull { !it.startsWith("-") } ?: return emptyList()
        return listOf(appId, appId.substringAfterLast('.'))
    }
    if (command == "snap" && remaining.getOrNull(1)?.normalizedToken() == "run") {
        val appId = remaining.drop(2).firstOrNull { !it.startsWith("-") } ?: return emptyList()
        return listOf(appId, appId.substringAfterLast('.'))
    }

    return listOf(remaining.first())
}

private fun shellWords(command: String): List<String> {
    val words = mutableListOf<String>()
    val current = StringBuilder()
    var quote: Char? = null
    var escaping = false

    command.forEach { char ->
        when {
            escaping -> {
                current.append(char)
                escaping = false
            }
            char == '\\' -> escaping = true
            quote != null && char == quote -> quote = null
            quote == null && (char == '\'' || char == '"') -> quote = char
            quote == null && char.isWhitespace() -> {
                if (current.isNotEmpty()) {
                    words += current.toString()
                    current.clear()
                }
            }
            else -> current.append(char)
        }
    }
    if (current.isNotEmpty()) {
        words += current.toString()
    }
    return words
}

private fun String.isEnvAssignment(): Boolean {
    val equalsIndex = indexOf('=')
    return equalsIndex > 0 && take(equalsIndex).all { it == '_' || it.isLetterOrDigit() }
}

private fun String.matchesToken(other: String): Boolean {
    if (this == other) return true
    return (length >= 4 && other.contains(this)) || (other.length >= 4 && contains(other))
}

private fun String.normalizedToken(): String =
    lowercase(Locale.ROOT)
        .replace(Regex("[^a-z0-9]+"), "")

private fun File.appBundleAncestor(): File? =
    generateSequence(this) { it.parentFile }
        .firstOrNull { it.isDirectory && it.name.endsWith(".app", ignoreCase = true) }

private fun String?.isMacOSBundlePath(): Boolean {
    val path = this?.trim()?.takeIf { it.isNotEmpty() } ?: return false
    val file = File(path)
    return file.name.endsWith(".app", ignoreCase = true) ||
        file.appBundleAncestor() != null ||
        path.contains(".app/", ignoreCase = true)
}

private fun readImageBitmap(file: File): ImageBitmap? {
    if (!file.isFile) return null
    return runCatching {
        ImageIO.read(file)?.toVisibleImageBitmap()
    }.getOrNull()
}

private fun readImageBitmap(bytes: ByteArray): ImageBitmap? =
    runCatching {
        ImageIO.read(ByteArrayInputStream(bytes))?.toVisibleImageBitmap()
    }.getOrNull()

private fun readIconImage(file: File): ImageBitmap? =
    when (file.extension.lowercase(Locale.ROOT)) {
        "svg" -> readSvgImageBitmap(file) ?: readImageBitmap(file)
        else -> readImageBitmap(file)
    }

private fun readSvgImageBitmap(file: File, size: Int = 256): ImageBitmap? {
    if (!file.isFile) return null
    return runCatching {
        val data = Data.makeFromBytes(file.readBytes())
        try {
            val svg = SVGDOM(data)
            try {
                svg.setContainerSize(size.toFloat(), size.toFloat())
                val surface = Surface.makeRasterN32Premul(size, size)
                try {
                    surface.canvas.clear(0x00000000)
                    svg.render(surface.canvas)
                    val image = surface.makeImageSnapshot()
                    try {
                        val encoded = image.encodeToData(EncodedImageFormat.PNG) ?: return@runCatching null
                        try {
                            readImageBitmap(encoded.bytes)
                        } finally {
                            encoded.close()
                        }
                    } finally {
                        image.close()
                    }
                } finally {
                    surface.close()
                }
            } finally {
                svg.close()
            }
        } finally {
            data.close()
        }
    }.getOrNull()
}

private fun BufferedImage.toVisibleImageBitmap(): ImageBitmap? =
    takeIf { it.hasVisiblePixels() }?.toComposeImageBitmap()

private fun BufferedImage.hasVisiblePixels(): Boolean {
    for (y in 0 until height) {
        for (x in 0 until width) {
            val argb = getRGB(x, y)
            val alpha = (argb ushr 24) and 0xFF
            if (alpha > 8) return true
        }
    }
    return false
}

private fun sha256(value: String): String {
    val bytes = MessageDigest.getInstance("SHA-256").digest(value.toByteArray(Charsets.UTF_8))
    return bytes.joinToString(separator = "") { byte -> "%02x".format(byte) }
}

private fun Icon.toImageBitmapOrNull(): ImageBitmap? {
    val width = max(iconWidth, 1)
    val height = max(iconHeight, 1)
    val image = BufferedImage(width, height, BufferedImage.TYPE_INT_ARGB)
    val graphics = image.createGraphics()
    try {
        paintIcon(null, graphics, 0, 0)
    } finally {
        graphics.dispose()
    }
    return image.toVisibleImageBitmap()
}

private fun linuxDesktopEntryRoots(homeDir: File): List<File> =
    listOf(
        File(homeDir, ".local/share/applications"),
        File("/usr/local/share/applications"),
        File("/usr/share/applications"),
        File(homeDir, ".local/share/flatpak/exports/share/applications"),
        File("/var/lib/flatpak/exports/share/applications"),
        File("/var/lib/snapd/desktop/applications"),
    )

private fun linuxIconThemeRoots(homeDir: File): List<File> =
    listOf(
        File(homeDir, ".local/share/icons"),
        File("/usr/local/share/icons"),
        File("/usr/share/icons"),
    )

private const val macOSIconExportScript = """
function run(argv) {
  ObjC.import('AppKit');
  ObjC.import('Foundation');
  if (argv.length < 2) {
    return 'missing-arguments';
  }
  var sourcePath = argv[0];
  var outputPath = argv[1];
  var rect = $.NSMakeRect(0, 0, 256, 256);
  var image = $.NSWorkspace.sharedWorkspace.iconForFile(sourcePath);
  if (!image) {
    return 'missing-image';
  }
  image.setSize($.NSMakeSize(256, 256));
  var bitmap = $.NSBitmapImageRep.alloc.initWithBitmapDataPlanesPixelsWidePixelsHighBitsPerSampleSamplesPerPixelHasAlphaIsPlanarColorSpaceNameBitmapFormatBytesPerRowBitsPerPixel(
    null,
    256,
    256,
    8,
    4,
    true,
    false,
    $.NSDeviceRGBColorSpace,
    0,
    0,
    0
  );
  if (!bitmap) {
    return 'missing-bitmap';
  }
  var context = $.NSGraphicsContext.graphicsContextWithBitmapImageRep(bitmap);
  $.NSGraphicsContext.saveGraphicsState;
  $.NSGraphicsContext.setCurrentContext(context);
  image.drawInRectFromRectOperationFraction(rect, $.NSZeroRect, $.NSCompositingOperationSourceOver, 1.0);
  $.NSGraphicsContext.restoreGraphicsState;
  var data = bitmap.representationUsingTypeProperties($.NSBitmapImageFileTypePNG, $.NSDictionary.dictionary);
  if (!data) {
    return 'missing-data';
  }
  return data.writeToFileAtomically(outputPath, true) ? 'ok' : 'write-failed';
}
"""
