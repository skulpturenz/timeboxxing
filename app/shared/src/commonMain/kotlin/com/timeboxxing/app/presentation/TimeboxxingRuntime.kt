package com.timeboxxing.app.presentation

import com.timeboxxing.data.mock.mockTimeboxxingData
import com.timeboxxing.data.repository.StaticAmaRepository
import com.timeboxxing.data.repository.StaticProjectRepository
import com.timeboxxing.data.repository.StaticSettingsRepository
import com.timeboxxing.data.repository.StaticTimesheetRepository
import com.timeboxxing.data.repository.StaticUsageHistoryRepository
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.domain.model.DiagnosticsLogLine
import com.timeboxxing.domain.model.TimesheetExport
import com.timeboxxing.domain.model.UpdateChannel
import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.repository.AmaRepository
import com.timeboxxing.domain.repository.ProjectRepository
import com.timeboxxing.domain.repository.SettingsRepository
import com.timeboxxing.domain.repository.TimesheetRepository
import com.timeboxxing.domain.repository.UsageHistoryRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow

data class TimeboxxingRepositories(
    val usageHistoryRepository: UsageHistoryRepository,
    val amaRepository: AmaRepository,
    val settingsRepository: SettingsRepository,
    val projectRepository: ProjectRepository,
    val timesheetRepository: TimesheetRepository,
)

interface TimesheetExportFileWriter {
    suspend fun save(export: TimesheetExport): String?
}

class StaticTimesheetExportFileWriter : TimesheetExportFileWriter {
    val savedExports = mutableListOf<TimesheetExport>()

    override suspend fun save(export: TimesheetExport): String {
        savedExports += export
        return export.fileName
    }
}

/** A Handlebars template the user uploaded from disk. */
data class TemplateFile(val name: String, val contents: String)

/** Phases the PDF renderer reports so the UI can explain slow first-run browser setup. */
enum class PdfRenderStage {
    /** The headless browser is being downloaded (one-time, first use). Can take a while. */
    DownloadingBrowser,
    /** The browser is ready and the page is being rendered to PDF. */
    Rendering,
}

/**
 * Renders the entries export model (as JSON) through a Handlebars template into PDF bytes. The
 * real implementation lives on the desktop (Handlebars + a bundled headless Chromium).
 */
interface EntriesPdfRenderer {
    /**
     * Renders [modelJson] through [template] (or the built-in default when null). [onProgress]
     * reports the current phase and, for [PdfRenderStage.DownloadingBrowser], a 0..1 download
     * fraction (or null when the fraction is not yet known) so the UI can show a determinate
     * "downloading browser" bar on first use.
     */
    suspend fun render(
        modelJson: String,
        template: String?,
        onProgress: (stage: PdfRenderStage, fraction: Float?) -> Unit,
    ): ByteArray

    /** The bundled default template, used for the "download default template" action. */
    fun defaultTemplate(): String
}

/** Opens a native file picker so the user can upload a custom Handlebars template. */
interface EntriesTemplateFileReader {
    suspend fun open(): TemplateFile?
}

class StaticEntriesPdfRenderer : EntriesPdfRenderer {
    override suspend fun render(
        modelJson: String,
        template: String?,
        onProgress: (stage: PdfRenderStage, fraction: Float?) -> Unit,
    ): ByteArray {
        onProgress(PdfRenderStage.Rendering, null)
        return (template ?: defaultTemplate()).encodeToByteArray()
    }

    override fun defaultTemplate(): String = "PDF export is unavailable in preview."
}

class StaticEntriesTemplateFileReader : EntriesTemplateFileReader {
    override suspend fun open(): TemplateFile? = null
}

sealed interface TimeboxxingSidecarStatus {
    data object Starting : TimeboxxingSidecarStatus
    data object Ready : TimeboxxingSidecarStatus
    data class Failed(val message: String) : TimeboxxingSidecarStatus
}

interface TimeboxxingRuntime {
    val usageDays: List<UsageDay>
    val initialNotice: String?
    /**
     * A message describing a previously-attempted in-app update that failed to install (e.g. a
     * Windows app-image swap that did not apply), surfaced once on the next launch. Null when the
     * last update either succeeded or was never attempted.
     */
    val initialUpdateInstallFailure: String?
    val initialAppearanceMode: AppearanceMode
    val initialUpdateChannel: UpdateChannel
    val initialNotifyUpdatesOnStartup: Boolean
    /** True when this build is a stable (master) release; false for prerelease/dev/CI builds. */
    val isStableBuild: Boolean
    val diagnosticsEnabled: Boolean
    val dataDirectory: String
    val appearanceMode: StateFlow<AppearanceMode>
    val repositories: StateFlow<TimeboxxingRepositories>
    val sidecarStatus: StateFlow<TimeboxxingSidecarStatus>
    val diagnosticsLogs: StateFlow<List<DiagnosticsLogLine>>
    val timesheetExportFileWriter: TimesheetExportFileWriter
    val entriesPdfRenderer: EntriesPdfRenderer
    val entriesTemplateFileReader: EntriesTemplateFileReader
    val appUpdater: AppUpdater

    suspend fun setAppearanceMode(mode: AppearanceMode)

    suspend fun setUpdateChannel(channel: UpdateChannel)

    suspend fun setNotifyUpdatesOnStartup(enabled: Boolean)

    suspend fun restartSidecar()
}

class StaticTimeboxxingRuntime(
    private val data: com.timeboxxing.domain.model.TimeboxxingMockData = mockTimeboxxingData(),
    override val initialAppearanceMode: AppearanceMode = AppearanceMode.System,
    override val initialUpdateChannel: UpdateChannel = UpdateChannel.Stable,
    override val initialNotifyUpdatesOnStartup: Boolean = true,
    override val isStableBuild: Boolean = true,
    override val diagnosticsEnabled: Boolean = false,
    override val dataDirectory: String = "",
) : TimeboxxingRuntime {
    override val usageDays: List<UsageDay> = data.usageDays
    override val initialNotice: String? = null
    override val initialUpdateInstallFailure: String? = null
    override val appearanceMode = MutableStateFlow(initialAppearanceMode)
    override val repositories = MutableStateFlow(
        TimeboxxingRepositories(
            usageHistoryRepository = StaticUsageHistoryRepository(data.usageEvents),
            amaRepository = StaticAmaRepository(),
            settingsRepository = StaticSettingsRepository(),
            projectRepository = StaticProjectRepository(data.projects),
            timesheetRepository = StaticTimesheetRepository(data.initialEntries),
        ),
    )
    override val sidecarStatus = MutableStateFlow<TimeboxxingSidecarStatus>(TimeboxxingSidecarStatus.Ready)
    override val diagnosticsLogs = MutableStateFlow(emptyList<DiagnosticsLogLine>())
    override val timesheetExportFileWriter = StaticTimesheetExportFileWriter()
    override val entriesPdfRenderer: EntriesPdfRenderer = StaticEntriesPdfRenderer()
    override val entriesTemplateFileReader: EntriesTemplateFileReader = StaticEntriesTemplateFileReader()
    override val appUpdater: AppUpdater = StaticAppUpdater()

    override suspend fun setAppearanceMode(mode: AppearanceMode) {
        appearanceMode.value = mode
    }

    override suspend fun setUpdateChannel(channel: UpdateChannel) = Unit

    override suspend fun setNotifyUpdatesOnStartup(enabled: Boolean) = Unit

    override suspend fun restartSidecar() = Unit
}
