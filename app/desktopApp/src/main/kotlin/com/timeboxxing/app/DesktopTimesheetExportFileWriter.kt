package com.timeboxxing.app

import com.timeboxxing.app.presentation.TimesheetExportFileWriter
import com.timeboxxing.domain.model.TimesheetExport
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

internal class DesktopTimesheetExportFileWriter : TimesheetExportFileWriter {
    override suspend fun save(export: TimesheetExport): String? {
        val destination = NativeFileDialogs.chooseSaveDestination(
            title = "Export",
            suggestedFileName = export.fileName,
        ) ?: return null
        withContext(Dispatchers.IO) {
            destination.writeBytes(export.content)
        }
        return destination.absolutePath
    }
}
