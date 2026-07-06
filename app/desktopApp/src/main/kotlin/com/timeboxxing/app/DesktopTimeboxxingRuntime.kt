package com.timeboxxing.app

import com.timeboxxing.app.presentation.AppUpdater
import com.timeboxxing.app.presentation.EntriesPdfRenderer
import com.timeboxxing.app.presentation.EntriesTemplateFileReader
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
import com.timeboxxing.domain.model.UpdateChannel
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.CoroutineExceptionHandler
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.nio.file.Files
import java.nio.file.Path

internal class DesktopTimeboxxingRuntime(
    private val secretStore: SecretStore,
    private val appearancePreferences: AppearancePreferences,
    private val updateChannelPreferences: UpdateChannelPreferences,
    private val updateNotificationPreferences: UpdateNotificationPreferences,
    private val sidecarManager: SidecarProcessManager,
    private val sidecarSessionLog: SidecarSessionLog,
    override val diagnosticsEnabled: Boolean,
    private val javaEnv: JavaEnv,
) : TimeboxxingRuntime, AutoCloseable {
    private val scope = CoroutineScope(
        SupervisorJob() + Dispatchers.Default + CoroutineExceptionHandler { _, throwable ->
            DesktopSentry.captureException(throwable)
        },
    )
    private var sidecarConnection: SidecarConnection? = null

    override val usageDays = recentUsageDays()
    override val initialNotice: String = StartingSidecarMessage
    override val initialUpdateInstallFailure: String? =
        readAndClearUpdateFailure(sidecarManager.dataDirectory)
    override val initialAppearanceMode: AppearanceMode = appearancePreferences.load()
    override val dataDirectory: String = sidecarManager.dataDirectory.toString()

    override val initialUpdateChannel: UpdateChannel = updateChannelPreferences.load()
    private val _updateChannel = MutableStateFlow(initialUpdateChannel)

    override val initialNotifyUpdatesOnStartup: Boolean = updateNotificationPreferences.loadNotifyOnStartup()
    private val _notifyUpdatesOnStartup = MutableStateFlow(initialNotifyUpdatesOnStartup)

    override val isStableBuild: Boolean = javaEnv == JavaEnv.Production

    private val _appearanceMode = MutableStateFlow(initialAppearanceMode)
    override val appearanceMode: StateFlow<AppearanceMode> = _appearanceMode

    private val _repositories = MutableStateFlow(startingRepositories())
    override val repositories: StateFlow<TimeboxxingRepositories> = _repositories

    private val _sidecarStatus = MutableStateFlow<TimeboxxingSidecarStatus>(TimeboxxingSidecarStatus.Starting)
    override val sidecarStatus: StateFlow<TimeboxxingSidecarStatus> = _sidecarStatus

    override val diagnosticsLogs = sidecarSessionLog.lines
    override val timesheetExportFileWriter: TimesheetExportFileWriter = DesktopTimesheetExportFileWriter()
    override val entriesPdfRenderer: EntriesPdfRenderer =
        DesktopEntriesPdfRenderer(sidecarManager.dataDirectory.resolve("playwright-browsers").toFile())
    override val entriesTemplateFileReader: EntriesTemplateFileReader = DesktopEntriesTemplateFileReader()
    override val appUpdater: AppUpdater = DesktopAppUpdater(
        currentVersion = DesktopBuildConfig.AppVersion,
        javaEnv = javaEnv,
        dataDirectory = sidecarManager.dataDirectory,
        onBeforeExit = { close() },
    )

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

    override suspend fun setUpdateChannel(channel: UpdateChannel) {
        if (_updateChannel.value == channel) return
        _updateChannel.value = channel
        withContext(Dispatchers.IO) {
            updateChannelPreferences.save(channel)
        }
    }

    override suspend fun setNotifyUpdatesOnStartup(enabled: Boolean) {
        if (_notifyUpdatesOnStartup.value == enabled) return
        _notifyUpdatesOnStartup.value = enabled
        withContext(Dispatchers.IO) {
            updateNotificationPreferences.saveNotifyOnStartup(enabled)
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
        } catch (cancellation: CancellationException) {
            throw cancellation
        } catch (throwable: Throwable) {
            DesktopSentry.captureException(throwable)
            val message = throwable.message ?: "Usage sidecar failed to start."
            _repositories.value = failedRepositories(message)
            _sidecarStatus.value = TimeboxxingSidecarStatus.Failed(message)
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

// Must match the marker file written by DesktopAppUpdater's Windows update script.
internal const val UpdateFailureMarkerName = "update-failed.json"
internal const val DefaultUpdateFailureMessage = "The last update couldn't be installed."

/**
 * Reads the [UpdateFailureMarkerName] marker that [DesktopAppUpdater]'s Windows update script drops
 * in [dataDirectory] when an in-app update did not actually install, turns it into a user-facing
 * message, and deletes it so it is shown only once. Returns null when no update failed since the
 * last launch.
 */
internal fun readAndClearUpdateFailure(dataDirectory: Path): String? {
    val marker = dataDirectory.resolve(UpdateFailureMarkerName)
    if (!Files.exists(marker)) return null
    val message = runCatching { formatUpdateFailureMessage(Files.readString(marker)) }
        .getOrElse { DefaultUpdateFailureMessage }
    runCatching { Files.deleteIfExists(marker) }
    return message
}

/** Formats the JSON payload written by the Windows update script into a user-facing sentence. */
internal fun formatUpdateFailureMessage(markerJson: String): String {
    val obj = Json.parseToJsonElement(markerJson).jsonObject
    val version = obj["version"]?.jsonPrimitive?.contentOrNull?.takeIf { it.isNotBlank() }
    val logPath = obj["log"]?.jsonPrimitive?.contentOrNull?.takeIf { it.isNotBlank() }
    return buildString {
        append("The update")
        if (version != null) append(" to v$version")
        append(" couldn't be installed.")
        if (logPath != null) append(" See $logPath for details.")
    }
}
