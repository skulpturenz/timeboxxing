package com.timeboxxing.app.presentation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.timeboxxing.data.time.usageDayForCalendarDate
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.model.AmaStructuredQuery
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.domain.model.DatabasePruneRange
import com.timeboxxing.domain.model.TimesheetEntryDraft
import com.timeboxxing.domain.model.TimesheetExportFormat
import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.model.plusDays
import com.timeboxxing.domain.repository.AmaRepository
import com.timeboxxing.domain.repository.ProjectRepository
import com.timeboxxing.domain.repository.SettingsRepository
import com.timeboxxing.domain.repository.TimesheetRepository
import com.timeboxxing.domain.repository.UsageHistoryRepository
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.flow.catch
import kotlinx.coroutines.launch

class TimeboxxingViewModel(
    private val runtime: TimeboxxingRuntime,
) : ViewModel() {
    private val _state = MutableStateFlow(
        createSidecarTimeboxxingState(
            usageDays = runtime.usageDays,
            initialNotice = runtime.initialNotice,
            initialUpdateInstallFailure = runtime.initialUpdateInstallFailure,
            dataDirectory = runtime.dataDirectory,
            appVersion = runtime.appUpdater.currentVersion,
            appearanceMode = runtime.initialAppearanceMode,
            updateChannel = runtime.initialUpdateChannel,
            notifyUpdatesOnStartup = runtime.initialNotifyUpdatesOnStartup,
            isStableBuild = runtime.isStableBuild,
        ).copy(diagnosticsEnabled = runtime.diagnosticsEnabled),
    )

    val state: StateFlow<TimeboxxingScreenState> = _state
        .stateIn(viewModelScope, SharingStarted.Eagerly, _state.value)

    private val ready = runtime.sidecarStatus
        .map { it is TimeboxxingSidecarStatus.Ready }
        .distinctUntilChanged()

    init {
        observeAppearanceMode()
        observeUsageLoads()
        observeUsageStream()
        observeProjectLoads()
        observeTimesheetEntryLoads()
        observeSettingsLoads()
        observeAmaIndexStatus()
        checkForUpdatesOnStartup()
    }

    fun dispatch(action: TimeboxxingAction) {
        val currentState = _state.value
        val currentRepositories = runtime.repositories.value
        val amaQuestion = currentState.amaQuestionFor(action)
        val amaStructuredQuery = action.amaStructuredQuery()
        val settingsToSave = currentState.settingsToSaveFor(action)
        val projectToCreate = action.projectToCreate()
        val projectToDelete = action.projectToDelete()
        val draftEntryToCreate = currentState.draftEntryToCreateFor(action)
        val duplicateEntryToCreate = currentState.duplicateEntryToCreateFor(action)
        val entryToDelete = currentState.entryToDeleteFor(action)
        val timesheetToExport = currentState.timesheetToExportFor(action)
        val databasePrune = currentState.databasePruneRangeFor(action)

        reduce(action)

        if (action is TimeboxxingAction.UpdateAppearanceMode) {
            viewModelScope.launch {
                runtime.setAppearanceMode(action.mode)
            }
        }
        if (action is TimeboxxingAction.UpdateUpdateChannel) {
            viewModelScope.launch {
                runtime.setUpdateChannel(action.channel)
            }
        }
        if (action is TimeboxxingAction.SetNotifyUpdatesOnStartup) {
            viewModelScope.launch {
                runtime.setNotifyUpdatesOnStartup(action.enabled)
            }
        }
        if (amaQuestion != null) {
            askAma(currentRepositories.amaRepository, amaQuestion)
        }
        if (amaStructuredQuery != null) {
            askStructuredAma(currentRepositories.amaRepository, amaStructuredQuery)
        }
        if (settingsToSave != null) {
            saveSettings(currentRepositories.settingsRepository, settingsToSave)
        }
        if (projectToCreate != null) {
            createProject(currentRepositories.projectRepository, projectToCreate.name, projectToCreate.colorArgb)
        }
        if (projectToDelete != null) {
            deleteProject(currentRepositories.projectRepository, projectToDelete)
        }
        if (draftEntryToCreate != null) {
            createDraftEntry(currentRepositories.timesheetRepository, draftEntryToCreate)
        }
        if (duplicateEntryToCreate != null) {
            duplicateEntry(currentRepositories.timesheetRepository, duplicateEntryToCreate)
        }
        if (entryToDelete != null) {
            deleteEntry(currentRepositories.timesheetRepository, entryToDelete)
        }
        if (timesheetToExport != null) {
            exportTimesheet(currentRepositories.timesheetRepository, timesheetToExport)
        }
        if (databasePrune != null) {
            pruneDatabaseRange(
                settingsRepository = currentRepositories.settingsRepository,
                usageRepository = currentRepositories.usageHistoryRepository,
                timesheetRepository = currentRepositories.timesheetRepository,
                request = databasePrune,
            )
        }
        if (action is TimeboxxingAction.CheckForUpdates && !currentState.update.busy) {
            checkForUpdates(silent = false)
        }
        if (action is TimeboxxingAction.StartUpdateInstall) {
            val update = currentState.update.available
            if (update != null && currentState.update.downloadProgress == null && !currentState.update.installing) {
                installUpdate(update)
            }
        }
    }

    private fun observeAppearanceMode() {
        viewModelScope.launch {
            runtime.appearanceMode.collect { mode ->
                _state.update { state ->
                    if (state.appearanceMode == mode) {
                        state
                    } else {
                        reduceTimeboxxingState(state, TimeboxxingAction.UpdateAppearanceMode(mode))
                    }
                }
            }
        }
    }

    private fun observeUsageLoads() {
        viewModelScope.launch {
            combine(
                ready,
                runtime.repositories,
                _state.map { it.selectedDay }.distinctUntilChanged(),
            ) { isReady, repositories, selectedDay ->
                UsageLoadRequest(isReady, repositories.usageHistoryRepository, selectedDay)
            }.collectLatest { request ->
                if (request.isReady) {
                    loadUsage(request.repository, request.day)
                }
            }
        }
    }

    private fun observeUsageStream() {
        viewModelScope.launch {
            combine(
                ready,
                runtime.repositories,
                _state.map { it.selectedDay }.distinctUntilChanged(),
            ) { isReady, repositories, selectedDay ->
                UsageLoadRequest(isReady, repositories.usageHistoryRepository, selectedDay)
            }.collectLatest { request ->
                if (!request.isReady) return@collectLatest
                request.repository.watchUsageEvents(request.day)
                    .catch { error ->
                        reduce(
                            TimeboxxingAction.UsageLoadFailed(
                                request.day.startedAtEpochMillis,
                                error.message ?: "Usage sidecar stream stopped.",
                            ),
                        )
                    }
                    .collect { event ->
                        reduce(
                            TimeboxxingAction.MergeUsageEvent(
                                request.day.startedAtEpochMillis,
                                event,
                            ),
                        )
                    }
            }
        }
    }

    private fun observeProjectLoads() {
        viewModelScope.launch {
            combine(ready, runtime.repositories) { isReady, repositories ->
                ProjectLoadRequest(isReady, repositories.projectRepository)
            }.collectLatest { request ->
                if (request.isReady) {
                    loadProjects(request.repository)
                }
            }
        }
    }

    private fun observeTimesheetEntryLoads() {
        viewModelScope.launch {
            combine(
                ready,
                runtime.repositories,
                _state.map { it.selectedDay }.distinctUntilChanged(),
            ) { isReady, repositories, selectedDay ->
                TimesheetEntryLoadRequest(isReady, repositories.timesheetRepository, selectedDay)
            }.collectLatest { request ->
                if (request.isReady) {
                    loadTimesheetEntries(request.repository, request.day)
                }
            }
        }
    }

    private fun observeSettingsLoads() {
        viewModelScope.launch {
            combine(ready, runtime.repositories) { isReady, repositories ->
                SettingsLoadRequest(isReady, repositories.settingsRepository, showLoading = false)
            }.collectLatest { request ->
                if (request.isReady) {
                    loadSettings(request.repository, request.showLoading)
                    loadDatabaseMaintenance(request.repository, request.showLoading)
                }
            }
        }

        viewModelScope.launch {
            combine(
                ready,
                runtime.repositories,
                _state.map { it.selectedSection }.distinctUntilChanged(),
            ) { isReady, repositories, selectedSection ->
                SettingsSectionLoadRequest(
                    isReady = isReady,
                    repository = repositories.settingsRepository,
                    selectedSection = selectedSection,
                )
            }.collectLatest { request ->
                if (request.isReady && request.selectedSection == TimeboxxingSection.Settings) {
                    loadSettings(request.repository, showLoading = true)
                    loadDatabaseMaintenance(request.repository, showLoading = true)
                }
            }
        }
    }

    private fun observeAmaIndexStatus() {
        viewModelScope.launch {
            combine(
                ready,
                runtime.repositories,
                _state.map { it.selectedSection }.distinctUntilChanged(),
            ) { isReady, repositories, selectedSection ->
                AmaStatusRequest(isReady, repositories.amaRepository, selectedSection)
            }.collectLatest { request ->
                if (!request.isReady || request.selectedSection != TimeboxxingSection.Ama) {
                    return@collectLatest
                }
                while (true) {
                    val status = runCatching { request.repository.getSemanticIndexStatus() }
                    status.fold(
                        onSuccess = { reduce(TimeboxxingAction.AmaIndexStatusSucceeded(it)) },
                        onFailure = { error ->
                            reduce(
                                TimeboxxingAction.AmaIndexStatusFailed(
                                    error.message ?: "AMA status is unavailable.",
                                ),
                            )
                        },
                    )
                    delay(5_000)
                }
            }
        }
    }

    private suspend fun loadUsage(
        repository: UsageHistoryRepository,
        day: UsageDay,
    ) {
        reduce(TimeboxxingAction.LoadUsage)
        val loaded = runCatching { repository.getUsageEvents(day) }
        loaded.fold(
            onSuccess = { events ->
                reduce(TimeboxxingAction.UsageLoadSucceeded(day.startedAtEpochMillis, events))
            },
            onFailure = { error ->
                reduce(
                    TimeboxxingAction.UsageLoadFailed(
                        day.startedAtEpochMillis,
                        error.message ?: "Usage sidecar is unavailable.",
                    ),
                )
            },
        )
    }

    private suspend fun loadDatabaseMaintenance(
        repository: SettingsRepository,
        showLoading: Boolean,
    ) {
        if (showLoading) {
            reduce(TimeboxxingAction.LoadDatabaseMaintenance)
        }
        val loaded = runCatching { repository.getDatabaseMaintenanceStatus() }
        loaded.fold(
            onSuccess = { status ->
                reduce(TimeboxxingAction.DatabaseMaintenanceLoadSucceeded(status))
            },
            onFailure = { error ->
                if (showLoading || _state.value.selectedSection == TimeboxxingSection.Settings) {
                    reduce(
                        TimeboxxingAction.DatabaseMaintenanceLoadFailed(
                            error.message ?: "Database maintenance is unavailable.",
                        ),
                    )
                }
            },
        )
    }

    private suspend fun loadProjects(repository: ProjectRepository) {
        reduce(TimeboxxingAction.LoadProjects)
        val loaded = runCatching { repository.listProjects() }
        loaded.fold(
            onSuccess = { projects ->
                reduce(TimeboxxingAction.ProjectsLoadSucceeded(projects))
            },
            onFailure = { error ->
                reduce(TimeboxxingAction.ProjectsLoadFailed(error.message ?: "Projects are unavailable."))
            },
        )
    }

    private suspend fun loadTimesheetEntries(
        repository: TimesheetRepository,
        day: UsageDay,
    ) {
        reduce(TimeboxxingAction.LoadTimesheetEntries)
        val loaded = runCatching { repository.listEntries(day) }
        loaded.fold(
            onSuccess = { entries ->
                reduce(TimeboxxingAction.TimesheetEntriesLoadSucceeded(day.startedAtEpochMillis, entries))
            },
            onFailure = { error ->
                reduce(
                    TimeboxxingAction.TimesheetEntriesLoadFailed(
                        day.startedAtEpochMillis,
                        error.message ?: "Timesheets are unavailable.",
                    ),
                )
            },
        )
    }

    private suspend fun loadSettings(
        repository: SettingsRepository,
        showLoading: Boolean,
    ) {
        if (showLoading) {
            reduce(TimeboxxingAction.LoadSettings)
        }
        val loaded = runCatching {
            repository.listModelOptions() to repository.getAiSettings()
        }
        loaded.fold(
            onSuccess = { (options, settings) ->
                reduce(TimeboxxingAction.SettingsLoadSucceeded(options, settings))
            },
            onFailure = { error ->
                if (showLoading || _state.value.selectedSection == TimeboxxingSection.Settings) {
                    reduce(TimeboxxingAction.SettingsLoadFailed(error.message ?: "Settings are unavailable."))
                }
            },
        )
    }

    private fun askAma(repository: AmaRepository, question: String) {
        viewModelScope.launch {
            val answer = runCatching { repository.ask(question) }
            answer.fold(
                onSuccess = { reduce(TimeboxxingAction.AmaAnswerSucceeded(it)) },
                onFailure = { error ->
                    reduce(TimeboxxingAction.AmaAnswerFailed(error.message ?: "AMA is unavailable."))
                },
            )
        }
    }

    private fun askStructuredAma(repository: AmaRepository, query: AmaStructuredQuery) {
        viewModelScope.launch {
            val answer = runCatching { repository.askStructured(query) }
            answer.fold(
                onSuccess = { reduce(TimeboxxingAction.AmaAnswerSucceeded(it)) },
                onFailure = { error ->
                    reduce(TimeboxxingAction.AmaAnswerFailed(error.message ?: "AMA is unavailable."))
                },
            )
        }
    }

    private fun saveSettings(repository: SettingsRepository, settings: AiSettings) {
        viewModelScope.launch {
            val saved = runCatching { repository.saveAiSettings(settings) }
            saved.fold(
                onSuccess = { reduce(TimeboxxingAction.SettingsSaveSucceeded(it)) },
                onFailure = { error ->
                    reduce(TimeboxxingAction.SettingsSaveFailed(error.message ?: "Settings could not be saved."))
                },
            )
        }
    }

    private fun checkForUpdatesOnStartup() {
        viewModelScope.launch {
            delay(3_000)
            checkForUpdates(silent = true)
        }
    }

    private fun checkForUpdates(silent: Boolean) {
        val channel = _state.value.update.channel
        viewModelScope.launch {
            val checked = runCatching { runtime.appUpdater.check(channel) }
            checked.fold(
                onSuccess = { result ->
                    when {
                        // A background (startup) check stays quiet unless there is an update to offer.
                        silent && result !is UpdateCheckResult.Available ->
                            reduce(TimeboxxingAction.UpdateCheckSucceeded(UpdateCheckResult.Unsupported))
                        // A manual check on a build where updates are disabled (e.g. local/dev)
                        // still gives feedback via the up-to-date toast.
                        !silent && result is UpdateCheckResult.Unsupported ->
                            reduce(TimeboxxingAction.UpdateCheckSucceeded(UpdateCheckResult.UpToDate))
                        else ->
                            reduce(TimeboxxingAction.UpdateCheckSucceeded(result))
                    }
                },
                onFailure = { error ->
                    if (silent) {
                        reduce(TimeboxxingAction.UpdateCheckSucceeded(UpdateCheckResult.Unsupported))
                    } else {
                        reduce(TimeboxxingAction.UpdateCheckFailed(error.message ?: "Update check failed."))
                    }
                },
            )
        }
    }

    private fun installUpdate(update: AvailableUpdate) {
        viewModelScope.launch {
            val installed = runCatching {
                runtime.appUpdater.downloadAndInstall(update) { fraction ->
                    if (fraction >= 1f) {
                        reduce(TimeboxxingAction.UpdateInstallStarted)
                    } else {
                        reduce(TimeboxxingAction.UpdateDownloadProgress(fraction))
                    }
                }
            }
            // On success the updater relaunches and terminates the process, so this only runs on failure.
            installed.onFailure { error ->
                reduce(TimeboxxingAction.UpdateInstallFailed(error.message ?: "Update could not be installed."))
            }
        }
    }

    private fun createProject(repository: ProjectRepository, name: String, colorArgb: Long) {
        viewModelScope.launch {
            val created = runCatching { repository.createProject(name, colorArgb) }
            created.fold(
                onSuccess = { reduce(TimeboxxingAction.CreateProjectSucceeded(it)) },
                onFailure = { error ->
                    reduce(TimeboxxingAction.CreateProjectFailed(error.message ?: "Project could not be saved."))
                },
            )
        }
    }

    private fun deleteProject(repository: ProjectRepository, projectId: String) {
        viewModelScope.launch {
            val deleted = runCatching { repository.deleteProject(projectId) }
            deleted.fold(
                onSuccess = { reduce(TimeboxxingAction.DeleteProjectSucceeded(projectId)) },
                onFailure = { error ->
                    reduce(TimeboxxingAction.DeleteProjectFailed(error.message ?: "Project could not be deleted."))
                },
            )
        }
    }

    private fun createDraftEntry(repository: TimesheetRepository, request: TimesheetEntryCreateRequest) {
        viewModelScope.launch {
            val created = runCatching { repository.createEntry(request.day, request.draft) }
            created.fold(
                onSuccess = {
                    reduce(TimeboxxingAction.AddDraftEntrySucceeded(request.day.startedAtEpochMillis, it))
                },
                onFailure = { error ->
                    reduce(TimeboxxingAction.AddDraftEntryFailed(error.message ?: "Timesheet entry could not be saved."))
                },
            )
        }
    }

    private fun duplicateEntry(repository: TimesheetRepository, request: TimesheetEntryDuplicateRequest) {
        viewModelScope.launch {
            val created = runCatching { repository.createEntry(request.day, request.draft) }
            created.fold(
                onSuccess = {
                    reduce(
                        TimeboxxingAction.DuplicateEntrySucceeded(
                            dayStartedAtEpochMillis = request.day.startedAtEpochMillis,
                            sourceTitle = request.sourceTitle,
                            entry = it,
                        ),
                    )
                },
                onFailure = { error ->
                    reduce(TimeboxxingAction.DuplicateEntryFailed(error.message ?: "Timesheet entry could not be duplicated."))
                },
            )
        }
    }

    private fun deleteEntry(repository: TimesheetRepository, request: TimesheetEntryDeleteRequest) {
        viewModelScope.launch {
            val deleted = runCatching { repository.deleteEntry(request.entryId) }
            deleted.fold(
                onSuccess = { reduce(TimeboxxingAction.DeleteEntrySucceeded(request.entryId)) },
                onFailure = { error ->
                    reduce(
                        TimeboxxingAction.DeleteEntryFailed(
                            entryId = request.entryId,
                            message = error.message ?: "Timesheet entry could not be deleted.",
                        ),
                    )
                },
            )
        }
    }

    private fun exportTimesheet(repository: TimesheetRepository, request: TimesheetExportRequest) {
        viewModelScope.launch {
            val exported = runCatching {
                val export = repository.exportTimesheet(request.day, request.format)
                val destination = runtime.timesheetExportFileWriter.save(export)
                export to destination
            }
            exported.fold(
                onSuccess = { (export, destination) ->
                    if (destination == null) {
                        reduce(TimeboxxingAction.ExportTimesheetCanceled)
                    } else {
                        reduce(TimeboxxingAction.ExportTimesheetSucceeded(export.fileName))
                    }
                },
                onFailure = { error ->
                    reduce(TimeboxxingAction.ExportTimesheetFailed(error.message ?: "Timesheet could not be exported."))
                },
            )
        }
    }

    private fun pruneDatabaseRange(
        settingsRepository: SettingsRepository,
        usageRepository: UsageHistoryRepository,
        timesheetRepository: TimesheetRepository,
        request: DatabasePruneRequest,
    ) {
        viewModelScope.launch {
            val pruned = runCatching { settingsRepository.pruneDatabaseRange(request.range) }
            pruned.fold(
                onSuccess = { result ->
                    reduce(TimeboxxingAction.DatabasePruneSucceeded(result))
                    var vacuumFailureMessage: String? = null
                    if (result.counts.totalDeletedRows > 0L) {
                        reduce(TimeboxxingAction.VacuumDatabase)
                        val vacuumed = runCatching { settingsRepository.vacuumDatabase() }
                        vacuumed.fold(
                            onSuccess = { vacuumResult ->
                                reduce(TimeboxxingAction.DatabaseVacuumSucceeded(vacuumResult, result.counts))
                            },
                            onFailure = { error ->
                                vacuumFailureMessage = "Database rows were pruned, but compaction failed: ${error.message ?: "Please try again."}"
                            },
                        )
                    }
                    loadDatabaseMaintenance(settingsRepository, showLoading = false)
                    vacuumFailureMessage?.let { message ->
                        reduce(TimeboxxingAction.DatabaseVacuumFailed(message))
                    }
                    loadUsage(usageRepository, request.day)
                    loadTimesheetEntries(timesheetRepository, request.day)
                },
                onFailure = { error ->
                    reduce(TimeboxxingAction.DatabasePruneFailed(error.message ?: "Database range could not be pruned."))
                },
            )
        }
    }

    private fun reduce(action: TimeboxxingAction) {
        _state.update { reduceTimeboxxingState(it, action) }
    }
}

private data class UsageLoadRequest(
    val isReady: Boolean,
    val repository: UsageHistoryRepository,
    val day: UsageDay,
)

private data class SettingsLoadRequest(
    val isReady: Boolean,
    val repository: SettingsRepository,
    val showLoading: Boolean,
)

private data class ProjectLoadRequest(
    val isReady: Boolean,
    val repository: ProjectRepository,
)

private data class TimesheetEntryLoadRequest(
    val isReady: Boolean,
    val repository: TimesheetRepository,
    val day: UsageDay,
)

private data class SettingsSectionLoadRequest(
    val isReady: Boolean,
    val repository: SettingsRepository,
    val selectedSection: TimeboxxingSection,
)

private data class AmaStatusRequest(
    val isReady: Boolean,
    val repository: AmaRepository,
    val selectedSection: TimeboxxingSection,
)

private fun TimeboxxingScreenState.amaQuestionFor(action: TimeboxxingAction): String? =
    if (
        action == TimeboxxingAction.SubmitAmaQuestion &&
        amaInput.trim().isNotEmpty() &&
        !amaLoading
    ) {
        amaInput.trim()
    } else {
        null
    }

private fun TimeboxxingAction.amaStructuredQuery(): AmaStructuredQuery? =
    when (this) {
        is TimeboxxingAction.SubmitAmaStructuredQuery -> query
        else -> null
    }

private fun TimeboxxingScreenState.settingsToSaveFor(action: TimeboxxingAction): AiSettings? =
    if (action == TimeboxxingAction.SaveSettings && !settingsSaving) settingsDraft else null

private data class ProjectCreateRequest(
    val name: String,
    val colorArgb: Long,
)

private fun TimeboxxingAction.projectToCreate(): ProjectCreateRequest? =
    when (this) {
        is TimeboxxingAction.CreateProject -> ProjectCreateRequest(name, colorArgb)
        else -> null
    }

private fun TimeboxxingAction.projectToDelete(): String? =
    when (this) {
        is TimeboxxingAction.DeleteProject -> projectId
        else -> null
    }

private data class TimesheetEntryCreateRequest(
    val day: UsageDay,
    val draft: TimesheetEntryDraft,
)

private data class TimesheetEntryDuplicateRequest(
    val day: UsageDay,
    val sourceTitle: String,
    val draft: TimesheetEntryDraft,
)

private data class TimesheetEntryDeleteRequest(
    val entryId: String,
)

private data class TimesheetExportRequest(
    val day: UsageDay,
    val format: TimesheetExportFormat,
)

private data class DatabasePruneRequest(
    val range: DatabasePruneRange,
    val day: UsageDay,
)

private fun TimeboxxingScreenState.draftEntryToCreateFor(action: TimeboxxingAction): TimesheetEntryCreateRequest? {
    if (action != TimeboxxingAction.AddDraftEntry || entrySaving || !hasValidDraftProject()) return null
    return TimesheetEntryCreateRequest(
        day = selectedDay,
        draft = TimesheetEntryDraft(
            projectId = validProjectIdOrBlank(draft.projectId),
            title = draft.title.ifBlank { "New time entry" },
            notes = draft.notes,
            startMinute = draft.startMinute,
            durationMinutes = draft.durationMinutes,
            billable = draft.billable,
            sourceUsageIds = selectedUsageEvents.map { it.id },
        ),
    )
}

private fun TimeboxxingScreenState.duplicateEntryToCreateFor(action: TimeboxxingAction): TimesheetEntryDuplicateRequest? {
    if (action !is TimeboxxingAction.DuplicateEntry || entrySaving) return null
    val source = entries.firstOrNull { it.id == action.entryId } ?: return null
    return TimesheetEntryDuplicateRequest(
        day = selectedDay,
        sourceTitle = source.title,
        draft = TimesheetEntryDraft(
            projectId = validProjectIdOrBlank(source.projectId),
            title = "Copy of ${source.title}",
            notes = source.notes,
            startMinute = (source.startMinute + source.durationMinutes).coerceIn(0, 24 * 60 - 1),
            durationMinutes = source.durationMinutes,
            billable = source.billable,
            sourceUsageIds = emptyList(),
        ),
    )
}

private fun TimeboxxingScreenState.entryToDeleteFor(action: TimeboxxingAction): TimesheetEntryDeleteRequest? {
    if (action !is TimeboxxingAction.DeleteEntry) return null
    if (action.entryId in deletingEntryIds) return null
    if (entries.none { it.id == action.entryId }) return null
    return TimesheetEntryDeleteRequest(action.entryId)
}

private fun TimeboxxingScreenState.timesheetToExportFor(action: TimeboxxingAction): TimesheetExportRequest? {
    if (action !is TimeboxxingAction.ExportTimesheet) return null
    if (entries.isEmpty() || timesheetExporting) return null
    return TimesheetExportRequest(selectedDay, action.format)
}

private fun TimeboxxingScreenState.databasePruneRangeFor(action: TimeboxxingAction): DatabasePruneRequest? {
    if (action != TimeboxxingAction.PruneDatabaseRange || databasePruning || databaseVacuuming) return null
    val startDate = databasePruneStartDate ?: return null
    val endDate = databasePruneEndDate ?: return null
    if (startDate > endDate) return null

    val startedAt = usageDayForCalendarDate(startDate)
    val endedAt = usageDayForCalendarDate(endDate.plusDays(1))
    return DatabasePruneRequest(
        range = DatabasePruneRange(
            startedAtEpochMillis = startedAt.startedAtEpochMillis,
            endedAtEpochMillis = endedAt.startedAtEpochMillis,
        ),
        day = selectedDay,
    )
}

private fun TimeboxxingScreenState.hasValidDraftProject(): Boolean =
    draft.projectId.isBlank() || projects.any { it.id == draft.projectId }

private fun TimeboxxingScreenState.validProjectIdOrBlank(projectId: String): String =
    projectId.takeIf { id -> id.isNotBlank() && projects.any { it.id == id } }.orEmpty()
