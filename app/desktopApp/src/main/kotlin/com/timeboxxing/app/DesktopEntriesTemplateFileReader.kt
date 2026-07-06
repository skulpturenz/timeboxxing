package com.timeboxxing.app

import com.timeboxxing.app.presentation.EntriesTemplateFileReader
import com.timeboxxing.app.presentation.TemplateFile
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

internal class DesktopEntriesTemplateFileReader : EntriesTemplateFileReader {
    override suspend fun open(): TemplateFile? {
        val file = NativeFileDialogs.chooseFileToOpen(title = "Upload Handlebars template") ?: return null
        val contents = withContext(Dispatchers.IO) { file.readText() }
        return TemplateFile(name = file.name, contents = contents)
    }
}
