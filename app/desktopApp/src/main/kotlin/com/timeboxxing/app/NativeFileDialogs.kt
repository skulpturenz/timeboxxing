package com.timeboxxing.app

import kotlinx.coroutines.suspendCancellableCoroutine
import java.awt.FileDialog
import java.awt.Frame
import java.awt.GraphicsEnvironment
import java.io.File
import javax.swing.SwingUtilities
import kotlin.coroutines.resume

/**
 * Native OS file pickers backed by [java.awt.FileDialog] (the platform-native dialog on macOS and
 * Windows), rather than Swing's cross-platform [javax.swing.JFileChooser].
 */
internal object NativeFileDialogs {
    suspend fun chooseSaveDestination(title: String, suggestedFileName: String): File? =
        showDialog(title, FileDialog.SAVE) { file = suggestedFileName }

    suspend fun chooseFileToOpen(title: String): File? =
        showDialog(title, FileDialog.LOAD) { }

    private suspend fun showDialog(
        title: String,
        mode: Int,
        configure: FileDialog.() -> Unit,
    ): File? {
        if (GraphicsEnvironment.isHeadless()) {
            throw IllegalStateException("File dialog is unavailable.")
        }
        return suspendCancellableCoroutine { continuation ->
            SwingUtilities.invokeLater {
                val dialog = FileDialog(null as Frame?, title, mode).apply(configure)
                dialog.isVisible = true // blocks on the EDT until the native dialog is dismissed
                val directory = dialog.directory
                val name = dialog.file
                val file = if (directory != null && name != null) File(directory, name) else null
                if (continuation.isActive) {
                    continuation.resume(file)
                }
            }
        }
    }
}
