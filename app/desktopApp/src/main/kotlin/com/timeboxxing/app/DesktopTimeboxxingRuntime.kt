package com.timeboxxing.app

import com.timeboxxing.app.presentation.TimeboxxingRepositories
import com.timeboxxing.app.presentation.TimeboxxingRuntime
import com.timeboxxing.app.presentation.TimeboxxingSidecarStatus
import com.timeboxxing.app.presentation.TimesheetExportFileWriter
import com.timeboxxing.app.sidecar.SidecarConnection
import com.timeboxxing.app.sidecar.SidecarProcessManager
import com.timeboxxing.app.sidecar.SidecarSecrets
import com.timeboxxing.app.sidecar.SidecarSessionLog
import com.timeboxxing.app.sidecar.SidecarStartResult
import com.timeboxxing.data.repository.EmptyUsageHistoryRepository
import com.timeboxxing.data.repository.UnavailableProjectRepository
import com.timeboxxing.data.repository.UnavailableAmaRepository
import com.timeboxxing.data.repository.UnavailableSettingsRepository
import com.timeboxxing.data.repository.UnavailableTimesheetRepository
import com.timeboxxing.data.repository.UnavailableUsageHistoryRepository
import com.timeboxxing.data.time.recentUsageDays
import com.timeboxxing.domain.model.AppearanceMode
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.CoroutineExceptionHandler
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

internal class DesktopTimeboxxingRuntime(
    private val secretStore: SecretStore,
    private val appearancePreferences: AppearancePreferences,
    private val sidecarManager: SidecarProcessManager,
    private val sidecarSessionLog: SidecarSessionLog,
    override val diagnosticsEnabled: Boolean,
) : TimeboxxingRuntime, AutoCloseable {
    private val scope = CoroutineScope(
        SupervisorJob() + Dispatchers.Default + CoroutineExceptionHandler { _, throwable ->
            DesktopSentry.captureException(throwable)
        },
    )
    private var sidecarConnection: SidecarConnection? = null

    override val usageDays = recentUsageDays()
    override val initialNotice: String = StartingSidecarMessage
    override val initialAppearanceMode: AppearanceMode = appearancePreferences.load()
    override val dataDirectory: String = sidecarManager.dataDirectory.toString()

    private val _appearanceMode = MutableStateFlow(initialAppearanceMode)
    override val appearanceMode: StateFlow<AppearanceMode> = _appearanceMode

    private val _repositories = MutableStateFlow(startingRepositories())
    override val repositories: StateFlow<TimeboxxingRepositories> = _repositories

    private val _sidecarStatus = MutableStateFlow<TimeboxxingSidecarStatus>(TimeboxxingSidecarStatus.Starting)
    override val sidecarStatus: StateFlow<TimeboxxingSidecarStatus> = _sidecarStatus

    override val diagnosticsLogs = sidecarSessionLog.lines
    override val timesheetExportFileWriter: TimesheetExportFileWriter = DesktopTimesheetExportFileWriter()

    fun start() {
        scope.launch {
            restartSidecar()
        }
    }

    override suspend fun setAppearanceMode(mode: AppearanceMode) {
        if (_appearanceMode.value == mode) return
        _appearanceMode.value = mode
        withContext(Dispatchers.IO) {
            appearancePreferences.save(mode)
        }
    }

    override suspend fun restartSidecar() {
        val transaction = DesktopSentry.startTransaction("sidecar.start", "task")
        try {
            _sidecarStatus.value = TimeboxxingSidecarStatus.Starting
            closeSidecarConnection()
            _repositories.value = startingRepositories()

            val secrets = withContext(Dispatchers.IO) {
                SidecarSecrets(
                    openRouterApiKey = secretStore.read(SecretKey.OpenRouter),
                    ollamaApiKey = secretStore.read(SecretKey.Ollama),
                )
            }

            when (val result = sidecarManager.start(usageDays[usageDays.size / 2], secrets)) {
                is SidecarStartResult.Started -> {
                    sidecarConnection = result.connection
                    _repositories.value = TimeboxxingRepositories(
                        usageHistoryRepository = result.connection.repository,
                        amaRepository = result.connection.amaRepository,
                        settingsRepository = DesktopSettingsRepository(
                            delegate = result.connection.settingsRepository,
                            secretStore = secretStore,
                            onSettingsSaved = {
                                scope.launch {
                                    restartSidecar()
                                }
                            },
                        ),
                        projectRepository = result.connection.projectRepository,
                        timesheetRepository = result.connection.timesheetRepository,
                    )
                    _sidecarStatus.value = TimeboxxingSidecarStatus.Ready
                }

                is SidecarStartResult.Failed -> {
                    _repositories.value = failedRepositories(result.message)
                    _sidecarStatus.value = TimeboxxingSidecarStatus.Failed(result.message)
                }
            }
        } catch (throwable: Throwable) {
            DesktopSentry.captureException(throwable)
            throw throwable
        } finally {
            transaction.close()
        }
    }

    override fun close() {
        closeSidecarConnection()
        scope.cancel()
    }

    private fun closeSidecarConnection() {
        sidecarConnection?.close()
        sidecarConnection = null
    }

    private fun startingRepositories(): TimeboxxingRepositories =
        TimeboxxingRepositories(
            usageHistoryRepository = EmptyUsageHistoryRepository(),
            amaRepository = UnavailableAmaRepository(StartingSidecarMessage),
            settingsRepository = UnavailableSettingsRepository(StartingSidecarMessage),
            projectRepository = UnavailableProjectRepository(StartingSidecarMessage),
            timesheetRepository = UnavailableTimesheetRepository(StartingSidecarMessage),
        )

    private fun failedRepositories(message: String): TimeboxxingRepositories =
        TimeboxxingRepositories(
            usageHistoryRepository = UnavailableUsageHistoryRepository(message),
            amaRepository = UnavailableAmaRepository(message),
            settingsRepository = UnavailableSettingsRepository(message),
            projectRepository = UnavailableProjectRepository(message),
            timesheetRepository = UnavailableTimesheetRepository(message),
        )
}

private const val StartingSidecarMessage = "Starting usage sidecar..."
