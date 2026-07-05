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

sealed interface TimeboxxingSidecarStatus {
    data object Starting : TimeboxxingSidecarStatus
    data object Ready : TimeboxxingSidecarStatus
    data class Failed(val message: String) : TimeboxxingSidecarStatus
}

interface TimeboxxingRuntime {
    val usageDays: List<UsageDay>
    val initialNotice: String?
    val initialAppearanceMode: AppearanceMode
    val initialUpdateChannel: UpdateChannel
    val diagnosticsEnabled: Boolean
    val dataDirectory: String
    val appearanceMode: StateFlow<AppearanceMode>
    val repositories: StateFlow<TimeboxxingRepositories>
    val sidecarStatus: StateFlow<TimeboxxingSidecarStatus>
    val diagnosticsLogs: StateFlow<List<DiagnosticsLogLine>>
    val timesheetExportFileWriter: TimesheetExportFileWriter
    val appUpdater: AppUpdater

    suspend fun setAppearanceMode(mode: AppearanceMode)

    suspend fun setUpdateChannel(channel: UpdateChannel)

    suspend fun restartSidecar()
}

class StaticTimeboxxingRuntime(
    private val data: com.timeboxxing.domain.model.TimeboxxingMockData = mockTimeboxxingData(),
    override val initialAppearanceMode: AppearanceMode = AppearanceMode.System,
    override val initialUpdateChannel: UpdateChannel = UpdateChannel.Stable,
    override val diagnosticsEnabled: Boolean = false,
    override val dataDirectory: String = "",
) : TimeboxxingRuntime {
    override val usageDays: List<UsageDay> = data.usageDays
    override val initialNotice: String? = null
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
    override val appUpdater: AppUpdater = StaticAppUpdater()

    override suspend fun setAppearanceMode(mode: AppearanceMode) {
        appearanceMode.value = mode
    }

    override suspend fun setUpdateChannel(channel: UpdateChannel) = Unit

    override suspend fun restartSidecar() = Unit
}
