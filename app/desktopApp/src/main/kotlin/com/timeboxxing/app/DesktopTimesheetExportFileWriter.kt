package com.timeboxxing.app

import com.timeboxxing.app.presentation.TimesheetExportFileWriter
import com.timeboxxing.domain.model.TimesheetExport
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withContext
import java.awt.GraphicsEnvironment
import java.io.File
import javax.swing.JFileChooser
import javax.swing.SwingUtilities
import kotlin.coroutines.resume

internal class DesktopTimesheetExportFileWriter : TimesheetExportFileWriter {
    override suspend fun save(export: TimesheetExport): String? {
        val destination = chooseDestination(export) ?: return null
        withContext(Dispatchers.IO) {
            destination.writeBytes(export.content)
        }
        return destination.absolutePath
    }

    private suspend fun chooseDestination(export: TimesheetExport): File? {
        if (GraphicsEnvironment.isHeadless()) {
            throw IllegalStateException("File chooser is unavailable.")
        }
        return suspendCancellableCoroutine { continuation ->
            SwingUtilities.invokeLater {
                val chooser = JFileChooser().apply {
                    selectedFile = File(export.fileName)
                    dialogTitle = "Export timesheet"
                }
                val result = chooser.showSaveDialog(null)
                val file = if (result == JFileChooser.APPROVE_OPTION) chooser.selectedFile else null
                if (continuation.isActive) {
                    continuation.resume(file)
                }
            }
        }
    }
}
