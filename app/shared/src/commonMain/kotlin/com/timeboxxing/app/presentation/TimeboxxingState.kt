package com.timeboxxing.app.presentation

import com.timeboxxing.data.mock.mockTimeboxxingData
import com.timeboxxing.data.time.calendarDateForEpochMillis
import com.timeboxxing.data.time.usageDayForCalendarDate
import com.timeboxxing.domain.model.AmaAnswer
import com.timeboxxing.domain.model.AmaIndexState
import com.timeboxxing.domain.model.AmaIndexStatus
import com.timeboxxing.domain.model.AmaMessage
import com.timeboxxing.domain.model.AmaMessageRole
import com.timeboxxing.domain.model.AmaQueryKind
import com.timeboxxing.domain.model.AmaStructuredQuery
import com.timeboxxing.domain.model.AiModelOptions
import com.timeboxxing.domain.model.AiProvider
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.DatabaseMaintenanceStatus
import com.timeboxxing.domain.model.DatabasePruneCounts
import com.timeboxxing.domain.model.DatabasePruneResult
import com.timeboxxing.domain.model.DatabaseVacuumResult
import com.timeboxxing.domain.model.EntryDraft
import com.timeboxxing.domain.model.Project
import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.TimesheetExportFormat
import com.timeboxxing.domain.model.TimeboxxingMockData
import com.timeboxxing.domain.model.UpdateChannel
import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.model.UsageEvent
import com.timeboxxing.domain.model.UsageSourceType
import com.timeboxxing.domain.model.formatDuration
import com.timeboxxing.domain.model.plusDays

private const val DeletedProjectColorArgb: Long = 0xFF8A8D80
private const val NoProjectColorArgb: Long = 0xFF6E7F80
private const val InvalidDraftProjectNotice = "Choose an existing project or No project."
private const val AddEntryBeforeExportNotice = "Add an entry before exporting."
private const val UsageDayMinutes = 24 * 60
internal const val MinimumScheduleUsageDurationMinutes = 1
private const val SidecarUsageIdPrefix = "sidecar-"

data class TimeboxxingScreenState(
    val selectedSection: TimeboxxingSection,
    val dateIndex: Int,
    val zoomMinutes: Int,
    val projects: List<Project>,
    val usageEvents: List<UsageEvent>,
    val usageLoading: Boolean,
    val entries: List<TimeEntry>,
    val entriesLoading: Boolean = false,
    val entrySaving: Boolean = false,
    val deletingEntryIds: Set<String> = emptySet(),
    val timesheetExporting: Boolean = false,
    val scheduleFocusEntryId: String?,
    val scheduleFocusTarget: ScheduleFocusTarget? = null,
    val selectedUsageIds: Set<String>,
    val draft: EntryDraft,
    val notice: String?,
    val nextEntryNumber: Int,
    val usageDays: List<UsageDay>,
    val amaInput: String = "",
    val amaMessages: List<AmaMessage> = emptyList(),
    val amaLoading: Boolean = false,
    val amaError: String? = null,
    val amaIndexStatus: AmaIndexStatus? = null,
    val nextAmaMessageNumber: Int = 1,
    val nextScheduleFocusRequestId: Long = 1,
    val settingsOptions: AiModelOptions = AiModelOptions(),
    val aiSettings: AiSettings = AiSettings(),
    val settingsDraft: AiSettings = AiSettings(),
    val settingsLoading: Boolean = false,
    val settingsSaving: Boolean = false,
    val settingsError: String? = null,
    val settingsSavedMessage: String? = null,
    val databaseMaintenanceStatus: DatabaseMaintenanceStatus = DatabaseMaintenanceStatus(),
    val databaseMaintenanceLoading: Boolean = false,
    val databasePruning: Boolean = false,
    val databaseVacuuming: Boolean = false,
    val databasePruneStartDate: CalendarDate? = null,
    val databasePruneEndDate: CalendarDate? = null,
    val databaseMaintenanceError: String? = null,
    val databaseMaintenanceMessage: String? = null,
    val dataDirectory: String = "",
    val appearanceMode: AppearanceMode = AppearanceMode.System,
    val diagnosticsEnabled: Boolean = false,
    val isStableBuild: Boolean = true,
    val update: AppUpdateUiState = AppUpdateUiState(),
) {
    // Show the update dialog whenever an update is available and hasn't been closed this session.
    // Non-stable builds always show it (forced); stable builds also respect the startup-notify opt-out.
    val showUpdateDialog: Boolean
        get() = update.available != null &&
            !update.dialogDismissed &&
            (!isStableBuild || update.notifyOnStartup)

    val dateLabels: List<String>
        get() = usageDays.map { it.label }

    val selectedDay: UsageDay
        get() = usageDays.getOrElse(dateIndex) { usageDays.first() }

    val dateLabel: String
        get() = selectedDay.label

    val selectedCalendarDate: CalendarDate?
        get() = selectedDay.calendarDate

    val selectedUsageEvents: List<UsageEvent>
        get() = usageEvents
            .filter { it.isSelectableUsage() && it.id in selectedUsageIds }
            .sortedBy { it.startMinute }

    val selectedUsageMinutes: Int
        get() = selectedUsageEvents.sumOf { it.durationMinutes }

    val billableMinutes: Int
        get() = entries.filter { it.billable }.sumOf { it.durationMinutes }

    val capturedMinutes: Int
        get() = usageEvents.filter { it.isCapturedUsage() }.sumOf { it.durationMinutes }

    val unassignedUsageMinutes: Int
        get() {
            val assignedUsageIds = entries.flatMap { it.sourceUsageIds }.toSet()
            return usageEvents
                .filter { it.isCapturedUsage() && it.id !in assignedUsageIds }
                .sumOf { it.durationMinutes }
        }

    val isAmaConfigured: Boolean
        get() = aiSettings.hasConfiguredAiSecret()

    val canPruneDatabaseRange: Boolean
        get() {
            val startedAt = databasePruneStartDate ?: return false
            val endedAt = databasePruneEndDate ?: return false
            return !databasePruning && !databaseVacuuming && startedAt <= endedAt
        }

    val visibleNavigationSections: List<TimeboxxingSection>
        get() = buildList {
            add(TimeboxxingSection.Overview)
            if (isAmaConfigured) {
                add(TimeboxxingSection.Ama)
            }
            if (diagnosticsEnabled) {
                add(TimeboxxingSection.Diagnostics)
            }
            add(TimeboxxingSection.Settings)
        }

    fun projectFor(projectId: String): Project =
        if (projectId.isBlank()) {
            Project(
                id = "",
                name = "No project",
                client = "",
                colorArgb = NoProjectColorArgb,
                hourlyRateCents = 0,
            )
        } else {
            projects.firstOrNull { it.id == projectId }
            ?: Project(
                id = projectId,
                name = "Deleted project",
                client = "",
                colorArgb = DeletedProjectColorArgb,
                hourlyRateCents = 0,
            )
        }

    fun minutesForProject(projectId: String): Int =
        entries.filter { it.projectId == projectId }.sumOf { it.durationMinutes }

    fun invoiceTotalCents(): Int =
        entries.filter { it.billable }.sumOf { entry ->
            val project = projectFor(entry.projectId)
            project.hourlyRateCents * entry.durationMinutes / 60
        }
}

data class ScheduleFocusTarget(
    val date: CalendarDate,
    val minute: Int,
    val usageId: String?,
    val requestId: Long,
)

data class AppUpdateUiState(
    val currentVersion: String = "",
    val channel: UpdateChannel = UpdateChannel.Stable,
    val checkInProgress: Boolean = false,
    val available: AvailableUpdate? = null,
    val error: String? = null,
    val downloadProgress: Float? = null,
    val installing: Boolean = false,
    // Whether the auto-check on startup surfaces the update dialog (stable builds can opt out).
    val notifyOnStartup: Boolean = true,
    // Session-only: the user closed the update dialog (stable builds only).
    val dialogDismissed: Boolean = false,
) {
    val busy: Boolean
        get() = checkInProgress || downloadProgress != null || installing
}

enum class TimeboxxingSection {
    Overview,
    Ama,
    Diagnostics,
    Settings,
}

sealed interface TimeboxxingAction {
    data class SelectSection(val section: TimeboxxingSection) : TimeboxxingAction
    data class MoveDate(val delta: Int) : TimeboxxingAction
    data class SelectDate(val date: CalendarDate) : TimeboxxingAction
    data class ChangeZoom(val minutes: Int) : TimeboxxingAction
    data object LoadUsage : TimeboxxingAction
    data class UsageLoadSucceeded(val dayStartedAtEpochMillis: Long, val events: List<UsageEvent>) : TimeboxxingAction
    data class UsageLoadFailed(val dayStartedAtEpochMillis: Long, val message: String) : TimeboxxingAction
    data class MergeUsageEvent(val dayStartedAtEpochMillis: Long, val event: UsageEvent) : TimeboxxingAction
    data class ToggleUsageSelection(val usageId: String) : TimeboxxingAction
    data object ClearUsageSelection : TimeboxxingAction
    data object NewBlankDraft : TimeboxxingAction
    data class UpdateDraftProject(val projectId: String) : TimeboxxingAction
    data class UpdateDraftTitle(val title: String) : TimeboxxingAction
    data class UpdateDraftNotes(val notes: String) : TimeboxxingAction
    data class UpdateDraftStart(val startMinute: Int) : TimeboxxingAction
    data class UpdateDraftDuration(val durationMinutes: Int) : TimeboxxingAction
    data class UpdateDraftBillable(val billable: Boolean) : TimeboxxingAction
    data object LoadProjects : TimeboxxingAction
    data class ProjectsLoadSucceeded(val projects: List<Project>) : TimeboxxingAction
    data class ProjectsLoadFailed(val message: String) : TimeboxxingAction
    data class CreateProject(val name: String, val colorArgb: Long) : TimeboxxingAction
    data class CreateProjectSucceeded(val project: Project) : TimeboxxingAction
    data class CreateProjectFailed(val message: String) : TimeboxxingAction
    data class DeleteProject(val projectId: String) : TimeboxxingAction
    data class DeleteProjectSucceeded(val projectId: String) : TimeboxxingAction
    data class DeleteProjectFailed(val message: String) : TimeboxxingAction
    data object LoadTimesheetEntries : TimeboxxingAction
    data class TimesheetEntriesLoadSucceeded(
        val dayStartedAtEpochMillis: Long,
        val entries: List<TimeEntry>,
    ) : TimeboxxingAction
    data class TimesheetEntriesLoadFailed(
        val dayStartedAtEpochMillis: Long,
        val message: String,
    ) : TimeboxxingAction
    data object AddDraftEntry : TimeboxxingAction
    data class AddDraftEntrySucceeded(
        val dayStartedAtEpochMillis: Long,
        val entry: TimeEntry,
    ) : TimeboxxingAction
    data class AddDraftEntryFailed(val message: String) : TimeboxxingAction
    data class DuplicateEntry(val entryId: String) : TimeboxxingAction
    data class DuplicateEntrySucceeded(
        val dayStartedAtEpochMillis: Long,
        val sourceTitle: String,
        val entry: TimeEntry,
    ) : TimeboxxingAction
    data class DuplicateEntryFailed(val message: String) : TimeboxxingAction
    data class DeleteEntry(val entryId: String) : TimeboxxingAction
    data class DeleteEntrySucceeded(val entryId: String) : TimeboxxingAction
    data class DeleteEntryFailed(
        val entryId: String,
        val message: String,
    ) : TimeboxxingAction
    data class ExportTimesheet(val format: TimesheetExportFormat) : TimeboxxingAction
    data class ExportTimesheetSucceeded(val fileName: String) : TimeboxxingAction
    data object ExportTimesheetCanceled : TimeboxxingAction
    data class ExportTimesheetFailed(val message: String) : TimeboxxingAction
    data object DismissNotice : TimeboxxingAction
    data class UpdateAmaInput(val input: String) : TimeboxxingAction
    data object SubmitAmaQuestion : TimeboxxingAction
    data class SubmitAmaStructuredQuery(val query: AmaStructuredQuery) : TimeboxxingAction
    data class AmaAnswerSucceeded(val answer: AmaAnswer) : TimeboxxingAction
    data class AmaAnswerFailed(val message: String) : TimeboxxingAction
    data class AmaIndexStatusSucceeded(val status: AmaIndexStatus) : TimeboxxingAction
    data class AmaIndexStatusFailed(val message: String) : TimeboxxingAction
    data object ClearAmaChat : TimeboxxingAction
    data class OpenAmaUsageSource(
        val startedAtEpochMillis: Long?,
        val transitionEventId: Long,
    ) : TimeboxxingAction
    data object LoadSettings : TimeboxxingAction
    data class SettingsLoadSucceeded(val options: AiModelOptions, val settings: AiSettings) : TimeboxxingAction
    data class SettingsLoadFailed(val message: String) : TimeboxxingAction
    data class UpdateSettingsProvider(val provider: AiProvider) : TimeboxxingAction
    data class UpdateOpenRouterBaseUrl(val value: String) : TimeboxxingAction
    data class UpdateOllamaBaseUrl(val value: String) : TimeboxxingAction
    data class UpdateOpenRouterApiKey(val value: String) : TimeboxxingAction
    data class UpdateOllamaApiKey(val value: String) : TimeboxxingAction
    data class UpdateSettingsEmbeddingModel(val id: Long) : TimeboxxingAction
    data class UpdateSettingsSemanticModel(val id: Long) : TimeboxxingAction
    data class UpdateAppearanceMode(val mode: AppearanceMode) : TimeboxxingAction
    data object SaveSettings : TimeboxxingAction
    data class SettingsSaveSucceeded(val settings: AiSettings) : TimeboxxingAction
    data class SettingsSaveFailed(val message: String) : TimeboxxingAction
    data object LoadDatabaseMaintenance : TimeboxxingAction
    data class DatabaseMaintenanceLoadSucceeded(val status: DatabaseMaintenanceStatus) : TimeboxxingAction
    data class DatabaseMaintenanceLoadFailed(val message: String) : TimeboxxingAction
    data class UpdateDatabasePruneStartDate(val date: CalendarDate) : TimeboxxingAction
    data class UpdateDatabasePruneEndDate(val date: CalendarDate) : TimeboxxingAction
    data object PruneDatabaseRange : TimeboxxingAction
    data class DatabasePruneSucceeded(val result: DatabasePruneResult) : TimeboxxingAction
    data class DatabasePruneFailed(val message: String) : TimeboxxingAction
    data object VacuumDatabase : TimeboxxingAction
    data class DatabaseVacuumSucceeded(
        val result: DatabaseVacuumResult,
        val prunedCounts: DatabasePruneCounts,
    ) : TimeboxxingAction
    data class DatabaseVacuumFailed(val message: String) : TimeboxxingAction
    data class UpdateUpdateChannel(val channel: UpdateChannel) : TimeboxxingAction
    data object CheckForUpdates : TimeboxxingAction
    data class UpdateCheckSucceeded(val result: UpdateCheckResult) : TimeboxxingAction
    data class UpdateCheckFailed(val message: String) : TimeboxxingAction
    data object DismissUpdateDialog : TimeboxxingAction
    data class SetNotifyUpdatesOnStartup(val enabled: Boolean) : TimeboxxingAction
    data object StartUpdateInstall : TimeboxxingAction
    data class UpdateDownloadProgress(val fraction: Float) : TimeboxxingAction
    data object UpdateInstallStarted : TimeboxxingAction
    data class UpdateInstallFailed(val message: String) : TimeboxxingAction
}

fun createInitialTimeboxxingState(
    data: TimeboxxingMockData = mockTimeboxxingData(),
    appearanceMode: AppearanceMode = AppearanceMode.System,
): TimeboxxingScreenState {
    val defaultProjectId = data.projects.firstOrNull()?.id.orEmpty()
    return TimeboxxingScreenState(
        selectedSection = TimeboxxingSection.Overview,
        dateIndex = data.usageDays.indexOfFirst { it.label == "Thursday, May 1, 2025" }.takeIf { it >= 0 } ?: 0,
        zoomMinutes = 15,
        projects = data.projects,
        usageEvents = normalizeUsageEvents(data.usageEvents),
        usageLoading = false,
        entries = data.initialEntries,
        scheduleFocusEntryId = null,
        selectedUsageIds = emptySet(),
        draft = blankDraft(defaultProjectId),
        notice = null,
        nextEntryNumber = data.initialEntries.size + 1,
        usageDays = data.usageDays,
        appearanceMode = appearanceMode,
    )
}

fun createSidecarTimeboxxingState(
    usageDays: List<UsageDay>,
    initialNotice: String? = null,
    dataDirectory: String = "",
    appVersion: String = "",
    data: TimeboxxingMockData = mockTimeboxxingData(),
    appearanceMode: AppearanceMode = AppearanceMode.System,
    updateChannel: UpdateChannel = UpdateChannel.Stable,
    notifyUpdatesOnStartup: Boolean = true,
    isStableBuild: Boolean = true,
): TimeboxxingScreenState {
    val defaultProjectId = data.projects.firstOrNull()?.id.orEmpty()
    val safeUsageDays = usageDays.ifEmpty { data.usageDays }
    return TimeboxxingScreenState(
        selectedSection = TimeboxxingSection.Overview,
        dateIndex = (safeUsageDays.size / 2).coerceAtMost(safeUsageDays.lastIndex),
        zoomMinutes = 15,
        projects = data.projects,
        usageEvents = emptyList(),
        usageLoading = true,
        entries = emptyList(),
        scheduleFocusEntryId = null,
        selectedUsageIds = emptySet(),
        draft = blankDraft(defaultProjectId),
        notice = initialNotice,
        nextEntryNumber = 1,
        usageDays = safeUsageDays,
        dataDirectory = dataDirectory,
        appearanceMode = appearanceMode,
        isStableBuild = isStableBuild,
        update = AppUpdateUiState(
            currentVersion = appVersion,
            channel = updateChannel,
            notifyOnStartup = notifyUpdatesOnStartup,
        ),
    )
}

fun reduceTimeboxxingState(
    state: TimeboxxingScreenState,
    action: TimeboxxingAction,
): TimeboxxingScreenState =
    when (action) {
        is TimeboxxingAction.SelectSection -> {
            val selectedSection = when {
                action.section == TimeboxxingSection.Ama && !state.isAmaConfigured -> TimeboxxingSection.Settings
                action.section == TimeboxxingSection.Diagnostics && !state.diagnosticsEnabled -> state.selectedSection
                else -> action.section
            }
            state.copy(
                selectedSection = selectedSection,
                amaError = if (selectedSection == TimeboxxingSection.Ama) state.amaError else null,
                notice = if (selectedSection == TimeboxxingSection.Overview) state.notice else null,
                settingsError = if (selectedSection == TimeboxxingSection.Settings) state.settingsError else null,
                settingsSavedMessage = if (selectedSection == TimeboxxingSection.Settings) state.settingsSavedMessage else null,
                databaseMaintenanceError = if (selectedSection == TimeboxxingSection.Settings) state.databaseMaintenanceError else null,
                databaseMaintenanceMessage = if (selectedSection == TimeboxxingSection.Settings) state.databaseMaintenanceMessage else null,
            )
        }

        is TimeboxxingAction.MoveDate -> moveDate(state, action.delta)

        is TimeboxxingAction.SelectDate -> selectDate(state, action.date)

        is TimeboxxingAction.ChangeZoom -> state.copy(
            zoomMinutes = action.minutes.coerceIn(5, 60),
            notice = null,
        )

        TimeboxxingAction.LoadUsage -> state.copy(
            usageLoading = true,
            usageEvents = emptyList(),
            scheduleFocusEntryId = null,
            selectedUsageIds = emptySet(),
            draft = blankDraft(state.draft.projectId),
            notice = null,
        )

        is TimeboxxingAction.UsageLoadSucceeded -> {
            if (action.dayStartedAtEpochMillis != state.selectedDay.startedAtEpochMillis) {
                state
            } else {
                val normalizedEvents = normalizeUsageEvents(action.events)
                state.copy(
                    usageEvents = normalizedEvents,
                    usageLoading = false,
                    scheduleFocusEntryId = null,
                    selectedUsageIds = focusedUsageSelection(state.scheduleFocusTarget, normalizedEvents),
                    draft = blankDraft(state.draft.projectId),
                    notice = null,
                )
            }
        }

        is TimeboxxingAction.UsageLoadFailed -> {
            if (action.dayStartedAtEpochMillis != state.selectedDay.startedAtEpochMillis) {
                state
            } else {
                state.copy(
                    usageEvents = emptyList(),
                    usageLoading = false,
                    scheduleFocusEntryId = null,
                    selectedUsageIds = emptySet(),
                    draft = blankDraft(state.draft.projectId),
                    notice = action.message,
                )
            }
        }

        is TimeboxxingAction.MergeUsageEvent -> {
            if (action.dayStartedAtEpochMillis != state.selectedDay.startedAtEpochMillis) {
                state
            } else {
                val mergedEvents = mergeUsageEvent(state.usageEvents, action.event)
                state.copy(
                    usageEvents = mergedEvents,
                    usageLoading = false,
                    selectedUsageIds = mergedFocusedUsageSelection(state, mergedEvents),
                )
            }
        }

        is TimeboxxingAction.ToggleUsageSelection -> {
            if (state.usageEvents.firstOrNull { it.id == action.usageId }?.isSelectableUsage() != true) {
                state
            } else {
                val nextSelection = state.selectedUsageIds.toggle(action.usageId)
                state.copy(
                    selectedUsageIds = nextSelection,
                    draft = draftFromSelection(state, nextSelection),
                    scheduleFocusEntryId = null,
                    scheduleFocusTarget = null,
                    notice = null,
                )
            }
        }

        TimeboxxingAction.ClearUsageSelection -> state.copy(
            selectedUsageIds = emptySet(),
            draft = blankDraft(state.draft.projectId),
            scheduleFocusEntryId = null,
            scheduleFocusTarget = null,
            notice = null,
        )

        TimeboxxingAction.NewBlankDraft -> state.copy(
            selectedUsageIds = emptySet(),
            draft = blankDraft(state.draft.projectId),
            scheduleFocusEntryId = null,
            scheduleFocusTarget = null,
            notice = null,
        )

        is TimeboxxingAction.UpdateDraftProject -> state.copy(
            draft = state.draft.copy(projectId = action.projectId),
            scheduleFocusEntryId = null,
            notice = null,
        )

        is TimeboxxingAction.UpdateDraftTitle -> state.copy(
            draft = state.draft.copy(title = action.title),
            scheduleFocusEntryId = null,
            notice = null,
        )

        is TimeboxxingAction.UpdateDraftNotes -> state.copy(
            draft = state.draft.copy(notes = action.notes),
            scheduleFocusEntryId = null,
            notice = null,
        )

        is TimeboxxingAction.UpdateDraftStart -> state.copy(
            draft = state.draft.copy(startMinute = action.startMinute.coerceIn(0, 24 * 60 - 1)),
            scheduleFocusEntryId = null,
            notice = null,
        )

        is TimeboxxingAction.UpdateDraftDuration -> state.copy(
            draft = state.draft.copy(durationMinutes = action.durationMinutes.coerceIn(5, 12 * 60)),
            scheduleFocusEntryId = null,
            notice = null,
        )

        is TimeboxxingAction.UpdateDraftBillable -> state.copy(
            draft = state.draft.copy(billable = action.billable),
            scheduleFocusEntryId = null,
            notice = null,
        )

        TimeboxxingAction.LoadProjects -> state.copy(notice = null)

        is TimeboxxingAction.ProjectsLoadSucceeded -> projectsLoaded(state, action.projects)

        is TimeboxxingAction.ProjectsLoadFailed -> state.copy(notice = action.message)

        is TimeboxxingAction.CreateProject -> state

        is TimeboxxingAction.CreateProjectSucceeded -> createProjectSucceeded(state, action.project)

        is TimeboxxingAction.CreateProjectFailed -> state.copy(notice = action.message)

        is TimeboxxingAction.DeleteProject -> state

        is TimeboxxingAction.DeleteProjectSucceeded -> deleteProject(state, action.projectId)

        is TimeboxxingAction.DeleteProjectFailed -> state.copy(notice = action.message)

        TimeboxxingAction.LoadTimesheetEntries -> state.copy(
            entriesLoading = true,
            entries = emptyList(),
            scheduleFocusEntryId = null,
            deletingEntryIds = emptySet(),
            notice = null,
        )

        is TimeboxxingAction.TimesheetEntriesLoadSucceeded -> {
            if (action.dayStartedAtEpochMillis != state.selectedDay.startedAtEpochMillis) {
                state
            } else {
                state.copy(
                    entries = action.entries,
                    entriesLoading = false,
                    scheduleFocusEntryId = null,
                    deletingEntryIds = state.deletingEntryIds.intersect(action.entries.map { it.id }.toSet()),
                    notice = null,
                )
            }
        }

        is TimeboxxingAction.TimesheetEntriesLoadFailed -> {
            if (action.dayStartedAtEpochMillis != state.selectedDay.startedAtEpochMillis) {
                state
            } else {
                state.copy(
                    entries = emptyList(),
                    entriesLoading = false,
                    scheduleFocusEntryId = null,
                    notice = action.message,
                )
            }
        }

        TimeboxxingAction.AddDraftEntry -> addDraftEntryStarted(state)

        is TimeboxxingAction.AddDraftEntrySucceeded -> addDraftEntrySucceeded(
            state = state,
            dayStartedAtEpochMillis = action.dayStartedAtEpochMillis,
            entry = action.entry,
        )

        is TimeboxxingAction.AddDraftEntryFailed -> state.copy(
            entrySaving = false,
            notice = action.message,
        )

        is TimeboxxingAction.DuplicateEntry -> duplicateEntryStarted(state, action.entryId)

        is TimeboxxingAction.DuplicateEntrySucceeded -> duplicateEntrySucceeded(
            state = state,
            dayStartedAtEpochMillis = action.dayStartedAtEpochMillis,
            sourceTitle = action.sourceTitle,
            entry = action.entry,
        )

        is TimeboxxingAction.DuplicateEntryFailed -> state.copy(
            entrySaving = false,
            notice = action.message,
        )

        is TimeboxxingAction.DeleteEntry -> deleteEntryStarted(state, action.entryId)

        is TimeboxxingAction.DeleteEntrySucceeded -> state.copy(
            entries = state.entries.filterNot { it.id == action.entryId },
            deletingEntryIds = state.deletingEntryIds - action.entryId,
            scheduleFocusEntryId = null,
            notice = null,
        )

        is TimeboxxingAction.DeleteEntryFailed -> state.copy(
            deletingEntryIds = state.deletingEntryIds - action.entryId,
            notice = action.message,
        )

        is TimeboxxingAction.ExportTimesheet -> exportTimesheetStarted(state)

        is TimeboxxingAction.ExportTimesheetSucceeded -> state.copy(
            timesheetExporting = false,
            notice = "Exported ${action.fileName}.",
        )

        TimeboxxingAction.ExportTimesheetCanceled -> state.copy(
            timesheetExporting = false,
            notice = null,
        )

        is TimeboxxingAction.ExportTimesheetFailed -> state.copy(
            timesheetExporting = false,
            notice = action.message,
        )

        TimeboxxingAction.DismissNotice -> state.copy(scheduleFocusEntryId = null, notice = null)

        is TimeboxxingAction.UpdateAmaInput -> state.copy(
            amaInput = action.input,
            amaError = null,
        )

        TimeboxxingAction.SubmitAmaQuestion -> submitAmaQuestion(state)

        is TimeboxxingAction.SubmitAmaStructuredQuery -> submitAmaStructuredQuery(state, action.query)

        is TimeboxxingAction.AmaAnswerSucceeded -> {
            val message = AmaMessage(
                id = "ama-${state.nextAmaMessageNumber}",
                role = AmaMessageRole.Assistant,
                content = action.answer.answer,
                model = action.answer.model,
                sources = action.answer.sources,
                artifacts = action.answer.artifacts,
            )
            state.copy(
                amaMessages = state.amaMessages + message,
                amaLoading = false,
                amaError = null,
                amaIndexStatus = action.answer.indexStatus ?: state.amaIndexStatus,
                nextAmaMessageNumber = state.nextAmaMessageNumber + 1,
            )
        }

        is TimeboxxingAction.AmaAnswerFailed -> state.copy(
            amaLoading = false,
            amaError = action.message,
        )

        is TimeboxxingAction.AmaIndexStatusSucceeded -> state.copy(
            amaIndexStatus = action.status,
            amaError = null,
        )

        is TimeboxxingAction.AmaIndexStatusFailed -> state.copy(
            amaIndexStatus = AmaIndexStatus(
                state = AmaIndexState.Unavailable,
                completedEventCount = state.amaIndexStatus?.completedEventCount ?: 0,
                indexedEventCount = state.amaIndexStatus?.indexedEventCount ?: 0,
                pendingEventCount = state.amaIndexStatus?.pendingEventCount ?: 0,
                backfillRunning = false,
                message = action.message,
            ),
            amaError = null,
        )

        TimeboxxingAction.ClearAmaChat -> state.copy(
            amaInput = "",
            amaMessages = emptyList(),
            amaLoading = false,
            amaError = null,
            amaIndexStatus = state.amaIndexStatus,
            nextAmaMessageNumber = 1,
        )

        is TimeboxxingAction.OpenAmaUsageSource -> openAmaUsageSource(
            state = state,
            startedAtEpochMillis = action.startedAtEpochMillis,
            transitionEventId = action.transitionEventId,
        )

        TimeboxxingAction.LoadSettings -> state.copy(
            settingsLoading = true,
            settingsError = null,
            settingsSavedMessage = null,
        )

        is TimeboxxingAction.SettingsLoadSucceeded -> {
            val settings = sanitizeAiSettings(action.settings, action.options)
            state.copy(
                selectedSection = selectVisibleSectionAfterSettingsChange(state.selectedSection, settings),
                settingsOptions = action.options,
                aiSettings = settings,
                settingsDraft = settings,
                settingsLoading = false,
                settingsSaving = false,
                settingsError = null,
            )
        }

        is TimeboxxingAction.SettingsLoadFailed -> state.copy(
            settingsLoading = false,
            settingsSaving = false,
            settingsError = action.message,
        )

        is TimeboxxingAction.UpdateSettingsProvider -> state.copy(
            settingsDraft = sanitizeAiSettings(state.settingsDraft.copy(provider = action.provider), state.settingsOptions),
            settingsError = null,
            settingsSavedMessage = null,
        )

        is TimeboxxingAction.UpdateOpenRouterBaseUrl -> state.copy(
            settingsDraft = state.settingsDraft.copy(openRouterBaseUrl = action.value),
            settingsError = null,
            settingsSavedMessage = null,
        )

        is TimeboxxingAction.UpdateOllamaBaseUrl -> state.copy(
            settingsDraft = state.settingsDraft.copy(ollamaBaseUrl = action.value),
            settingsError = null,
            settingsSavedMessage = null,
        )

        is TimeboxxingAction.UpdateOpenRouterApiKey -> state.copy(
            settingsDraft = state.settingsDraft.copy(openRouterApiKey = action.value),
            settingsError = null,
            settingsSavedMessage = null,
        )

        is TimeboxxingAction.UpdateOllamaApiKey -> state.copy(
            settingsDraft = state.settingsDraft.copy(ollamaApiKey = action.value),
            settingsError = null,
            settingsSavedMessage = null,
        )

        is TimeboxxingAction.UpdateSettingsEmbeddingModel -> state.copy(
            settingsDraft = state.settingsDraft.copy(embeddingModelId = action.id),
            settingsError = null,
            settingsSavedMessage = null,
        )

        is TimeboxxingAction.UpdateSettingsSemanticModel -> state.copy(
            settingsDraft = state.settingsDraft.copy(semanticModelId = action.id),
            settingsError = null,
            settingsSavedMessage = null,
        )

        is TimeboxxingAction.UpdateAppearanceMode -> state.copy(
            appearanceMode = action.mode,
            settingsError = null,
            settingsSavedMessage = null,
        )

        TimeboxxingAction.SaveSettings -> state.copy(
            settingsSaving = true,
            settingsError = null,
            settingsSavedMessage = null,
        )

        is TimeboxxingAction.SettingsSaveSucceeded -> {
            val settings = sanitizeAiSettings(action.settings, state.settingsOptions)
            state.copy(
                selectedSection = selectVisibleSectionAfterSettingsChange(state.selectedSection, settings),
                aiSettings = settings,
                settingsDraft = settings,
                settingsSaving = false,
                settingsError = null,
                settingsSavedMessage = "Settings saved. Restarting sidecar...",
            )
        }

        is TimeboxxingAction.SettingsSaveFailed -> state.copy(
            settingsSaving = false,
            settingsError = action.message,
        )

        TimeboxxingAction.LoadDatabaseMaintenance -> state.copy(
            databaseMaintenanceLoading = true,
            databaseMaintenanceError = null,
        )

        is TimeboxxingAction.DatabaseMaintenanceLoadSucceeded -> state.copy(
            databaseMaintenanceStatus = action.status,
            databaseMaintenanceLoading = false,
            databaseMaintenanceError = null,
        )

        is TimeboxxingAction.DatabaseMaintenanceLoadFailed -> state.copy(
            databaseMaintenanceLoading = false,
            databaseMaintenanceError = action.message,
        )

        is TimeboxxingAction.UpdateDatabasePruneStartDate -> state.copy(
            databasePruneStartDate = action.date,
            databaseMaintenanceError = null,
            databaseMaintenanceMessage = null,
        )

        is TimeboxxingAction.UpdateDatabasePruneEndDate -> state.copy(
            databasePruneEndDate = action.date,
            databaseMaintenanceError = null,
            databaseMaintenanceMessage = null,
        )

        TimeboxxingAction.PruneDatabaseRange -> {
            if (!state.canPruneDatabaseRange) {
                state
            } else {
                state.copy(
                    databasePruning = true,
                    databaseMaintenanceError = null,
                    databaseMaintenanceMessage = null,
                )
            }
        }

        is TimeboxxingAction.DatabasePruneSucceeded -> state.copy(
            databaseMaintenanceStatus = action.result.status,
            databasePruning = false,
            databaseMaintenanceError = null,
            databaseMaintenanceMessage = if (action.result.counts.totalDeletedRows == 0L) {
                databasePruneMessage(action.result.counts)
            } else {
                null
            },
        )

        is TimeboxxingAction.DatabasePruneFailed -> state.copy(
            databasePruning = false,
            databaseMaintenanceError = action.message,
        )

        TimeboxxingAction.VacuumDatabase -> state.copy(
            databaseVacuuming = true,
            databaseMaintenanceError = null,
            databaseMaintenanceMessage = null,
        )

        is TimeboxxingAction.DatabaseVacuumSucceeded -> state.copy(
            databaseMaintenanceStatus = DatabaseMaintenanceStatus(sizeBytes = action.result.sizeAfterBytes),
            databaseVacuuming = false,
            databaseMaintenanceError = null,
            databaseMaintenanceMessage = databaseVacuumMessage(action.prunedCounts),
        )

        is TimeboxxingAction.DatabaseVacuumFailed -> state.copy(
            databaseVacuuming = false,
            databaseMaintenanceError = action.message,
        )

        is TimeboxxingAction.UpdateUpdateChannel -> state.copy(
            update = state.update.copy(
                channel = action.channel,
                // A channel switch invalidates any pending update found on the previous channel.
                available = null,
                error = null,
            ),
        )

        // Closes the dialog for this session; it reappears on the next startup check unless the
        // user (on a stable build) has also opted out of startup notifications.
        TimeboxxingAction.DismissUpdateDialog -> state.copy(
            update = state.update.copy(dialogDismissed = true),
        )

        is TimeboxxingAction.SetNotifyUpdatesOnStartup -> state.copy(
            update = state.update.copy(
                notifyOnStartup = action.enabled,
                // Turning notifications off also closes the current dialog.
                dialogDismissed = if (!action.enabled) true else state.update.dialogDismissed,
            ),
        )

        TimeboxxingAction.CheckForUpdates -> state.copy(
            update = state.update.copy(
                checkInProgress = true,
                error = null,
            ),
        )

        is TimeboxxingAction.UpdateCheckSucceeded -> when (val result = action.result) {
            is UpdateCheckResult.Available -> state.copy(
                update = state.update.copy(
                    checkInProgress = false,
                    available = result.update,
                    error = null,
                    // A freshly found update re-opens the dialog even if a prior one was closed.
                    dialogDismissed = false,
                ),
            )

            UpdateCheckResult.UpToDate -> state.copy(
                update = state.update.copy(
                    checkInProgress = false,
                    available = null,
                    error = null,
                ),
                notice = "You're on the latest version.",
            )

            UpdateCheckResult.Unsupported -> state.copy(
                update = state.update.copy(
                    checkInProgress = false,
                    available = null,
                    error = null,
                ),
            )
        }

        is TimeboxxingAction.UpdateCheckFailed -> state.copy(
            update = state.update.copy(
                checkInProgress = false,
                error = action.message,
            ),
        )

        TimeboxxingAction.StartUpdateInstall -> state.copy(
            update = state.update.copy(
                downloadProgress = 0f,
                error = null,
            ),
        )

        is TimeboxxingAction.UpdateDownloadProgress -> state.copy(
            update = state.update.copy(
                downloadProgress = action.fraction.coerceIn(0f, 1f),
            ),
        )

        TimeboxxingAction.UpdateInstallStarted -> state.copy(
            update = state.update.copy(
                downloadProgress = null,
                installing = true,
            ),
        )

        is TimeboxxingAction.UpdateInstallFailed -> state.copy(
            update = state.update.copy(
                downloadProgress = null,
                installing = false,
                error = action.message,
            ),
        )
    }

private fun AiSettings.hasConfiguredAiSecret(): Boolean =
    openRouterSecretExists || ollamaSecretExists

private fun selectVisibleSectionAfterSettingsChange(
    selectedSection: TimeboxxingSection,
    settings: AiSettings,
): TimeboxxingSection =
    if (selectedSection == TimeboxxingSection.Ama && !settings.hasConfiguredAiSecret()) {
        TimeboxxingSection.Settings
    } else {
        selectedSection
    }

private fun sanitizeAiSettings(settings: AiSettings, options: AiModelOptions): AiSettings {
    val embeddingModels = options.embeddingModels.filter { it.supports(settings.provider) }
    val semanticModels = options.semanticModels.filter { it.supports(settings.provider) }
    val embeddingModelID = settings.embeddingModelId
        .takeIf { id -> embeddingModels.any { it.id == id } }
        ?: embeddingModels.firstOrNull()?.id
        ?: settings.embeddingModelId
    val semanticModelID = settings.semanticModelId
        .takeIf { id -> semanticModels.any { it.id == id } }
        ?: semanticModels.firstOrNull()?.id
        ?: settings.semanticModelId

    return settings.copy(
        embeddingModelId = embeddingModelID,
        semanticModelId = semanticModelID,
    )
}

private fun databasePruneMessage(counts: DatabasePruneCounts): String =
    if (counts.totalDeletedRows == 0L) {
        "No database rows matched that range."
    } else {
        "Pruned ${counts.totalDeletedRows} database rows. SQLite can reuse the freed space."
    }

private fun databaseVacuumMessage(counts: DatabasePruneCounts): String =
    "Pruned ${counts.totalDeletedRows} database rows and compacted the database."

private fun moveDate(state: TimeboxxingScreenState, delta: Int): TimeboxxingScreenState {
    val selectedDate = state.selectedCalendarDate
    if (selectedDate != null) {
        return selectDate(state, selectedDate.plusDays(delta))
    }

    val nextIndex = (state.dateIndex + delta).coerceIn(0, state.dateLabels.lastIndex)
    if (nextIndex == state.dateIndex) return state
    return state.copyForSelectedDate(
        usageDays = state.usageDays,
        dateIndex = nextIndex,
    )
}

private fun selectDate(state: TimeboxxingScreenState, date: CalendarDate): TimeboxxingScreenState {
    if (state.selectedCalendarDate == date) return state

    val existingIndex = state.usageDays.indexOfFirst { it.calendarDate == date }
    val nextUsageDays = if (existingIndex >= 0) {
        state.usageDays
    } else {
        (state.usageDays + usageDayForCalendarDate(date)).sortedBy { it.startedAtEpochMillis }
    }
    val nextIndex = nextUsageDays.indexOfFirst { it.calendarDate == date }
        .takeIf { it >= 0 }
        ?: state.dateIndex

    return state.copyForSelectedDate(
        usageDays = nextUsageDays,
        dateIndex = nextIndex,
    )
}

private fun TimeboxxingScreenState.copyForSelectedDate(
    usageDays: List<UsageDay>,
    dateIndex: Int,
): TimeboxxingScreenState =
    copy(
        usageDays = usageDays,
        dateIndex = dateIndex,
        usageEvents = emptyList(),
        usageLoading = true,
        entries = emptyList(),
        entriesLoading = true,
        entrySaving = false,
        deletingEntryIds = emptySet(),
        timesheetExporting = false,
        scheduleFocusEntryId = null,
        scheduleFocusTarget = null,
        selectedUsageIds = emptySet(),
        draft = blankDraft(draft.projectId),
        notice = null,
    )

private fun blankDraft(projectId: String): EntryDraft =
    EntryDraft(
        projectId = projectId,
        title = "New time entry",
        notes = "",
        startMinute = 13 * 60 + 30,
        durationMinutes = 30,
        billable = true,
    )

private fun draftFromSelection(
    state: TimeboxxingScreenState,
    selectedUsageIds: Set<String>,
): EntryDraft {
    val selectedEvents = state.usageEvents
        .filter { it.isSelectableUsage() && it.id in selectedUsageIds }
        .sortedBy { it.startMinute }
    if (selectedEvents.isEmpty()) return blankDraft(state.draft.projectId)

    val firstEvent = selectedEvents.first()
    val projectIds = state.projects.map { it.id }.toSet()
    val hintedProjectId = selectedEvents.firstNotNullOfOrNull { event ->
        event.projectHintId?.takeIf { it in projectIds }
    } ?: state.draft.projectId
    val title = if (selectedEvents.size == 1) {
        firstEvent.title
    } else {
        "Bundled work from ${selectedEvents.size} captured activities"
    }
    val notes = selectedEvents.joinToString(separator = "\n") { "${it.sourceName}: ${it.title}" }

    return state.draft.copy(
        projectId = hintedProjectId,
        title = title,
        notes = notes,
        startMinute = firstEvent.startMinute,
        durationMinutes = selectedEvents.sumOf { it.durationMinutes },
        billable = true,
    )
}

private fun mergeUsageEvent(
    existing: List<UsageEvent>,
    incoming: UsageEvent,
): List<UsageEvent> {
    val filtered = if (incoming.isActive) {
        existing.filterNot { it.isActive || it.id == incoming.id }
    } else {
        existing.filterNot { event ->
            event.id == incoming.id || event.matchesCompletedUsage(incoming)
        }
    }
    return normalizeUsageEvents(filtered + incoming)
}

private fun normalizeUsageEvents(events: List<UsageEvent>): List<UsageEvent> {
    val active = events.lastOrNull { it.isActive }
    val rawCompleted = events.filterNot { event ->
        event.isActive || active?.matchesCompletedUsage(event) == true
    }
    val completed = linearizedCompletedUsageEvents(rawCompleted)
    return (completed + listOfNotNull(active)).sortedBy { it.startMinute }
}

private data class OrderedUsageEvent(
    val event: UsageEvent,
    val inputIndex: Int,
    val reportOrder: Long,
)

private fun linearizedCompletedUsageEvents(events: List<UsageEvent>): List<UsageEvent> {
    if (events.isEmpty()) return emptyList()

    val orderedEvents = events
        .mapIndexed { index, event ->
            OrderedUsageEvent(
                event = event,
                inputIndex = index,
                reportOrder = event.reportOrder(index),
            )
        }
        .sortedWith(
            compareBy<OrderedUsageEvent> { it.reportOrder }
                .thenBy { it.inputIndex },
        )
        .fold(linkedMapOf<String, OrderedUsageEvent>()) { latestById, orderedEvent ->
            latestById[orderedEvent.event.id] = orderedEvent
            latestById
        }
        .values
        .sortedWith(
            compareBy<OrderedUsageEvent> { it.reportOrder }
                .thenBy { it.inputIndex },
        )

    val minuteOwners = arrayOfNulls<OrderedUsageEvent>(UsageDayMinutes)
    orderedEvents.forEach { orderedEvent ->
        val event = orderedEvent.event
        val startMinute = event.startMinute.coerceIn(0, UsageDayMinutes)
        val endMinute = (event.startMinute + event.durationMinutes).coerceIn(0, UsageDayMinutes)
        if (endMinute <= startMinute) return@forEach

        for (minute in startMinute until endMinute) {
            minuteOwners[minute] = orderedEvent
        }
    }

    val emittedUsageIds = mutableSetOf<String>()
    val linearEvents = mutableListOf<UsageEvent>()
    var minute = 0
    while (minute < UsageDayMinutes) {
        val owner = minuteOwners[minute]
        if (owner == null || owner.event.id in emittedUsageIds) {
            minute += 1
            continue
        }

        val startMinute = minute
        var endMinute = minute + 1
        while (
            endMinute < UsageDayMinutes &&
            minuteOwners[endMinute]?.event?.id == owner.event.id
        ) {
            endMinute += 1
        }

        emittedUsageIds += owner.event.id
        linearEvents += owner.event.copy(
            startMinute = startMinute,
            durationMinutes = endMinute - startMinute,
        )
        minute = endMinute
    }

    return linearEvents.sortedWith(
        compareBy<UsageEvent> { it.startMinute }
            .thenBy { it.id },
    )
}

private fun UsageEvent.reportOrder(inputIndex: Int): Long =
    if (id.startsWith(SidecarUsageIdPrefix)) {
        id.removePrefix(SidecarUsageIdPrefix).toLongOrNull() ?: inputIndex.toLong()
    } else {
        inputIndex.toLong()
    }

private fun UsageEvent.matchesCompletedUsage(completed: UsageEvent): Boolean =
    isActive &&
        !completed.isActive &&
        startMinute == completed.startMinute &&
        sourceType == completed.sourceType &&
        sourceName == completed.sourceName &&
        title == completed.title

private fun List<UsageEvent>.selectableUsageIds(): Set<String> =
    filter { it.isSelectableUsage() }.map { it.id }.toSet()

private fun UsageEvent.isSelectableUsage(): Boolean =
    !isActive && !isIdleUsage() && hasScheduleVisibleDuration()

private fun UsageEvent.isCapturedUsage(): Boolean =
    !isActive && !isIdleUsage()

internal fun UsageEvent.hasScheduleVisibleDuration(): Boolean =
    durationMinutes >= MinimumScheduleUsageDurationMinutes

private fun UsageEvent.isIdleUsage(): Boolean =
    sourceType == UsageSourceType.Idle

private fun projectsLoaded(
    state: TimeboxxingScreenState,
    projects: List<Project>,
): TimeboxxingScreenState {
    val projectIds = projects.map { it.id }.toSet()
    val nextDraftProjectId = state.draft.projectId
        .takeIf { it.isBlank() || it in projectIds }
        .orEmpty()

    return state.copy(
        projects = projects,
        draft = state.draft.copy(projectId = nextDraftProjectId),
        scheduleFocusEntryId = null,
        notice = null,
    )
}

private fun createProjectSucceeded(
    state: TimeboxxingScreenState,
    project: Project,
): TimeboxxingScreenState {
    val projects = if (state.projects.any { it.id == project.id }) {
        state.projects.map { existing -> if (existing.id == project.id) project else existing }
    } else {
        state.projects + project
    }

    return state.copy(
        projects = projects,
        draft = state.draft.copy(projectId = project.id),
        scheduleFocusEntryId = null,
        notice = "Created ${project.name}.",
    )
}

private fun deleteProject(
    state: TimeboxxingScreenState,
    projectId: String,
): TimeboxxingScreenState {
    if (state.projects.none { it.id == projectId }) return state

    val remainingProjects = state.projects.filterNot { it.id == projectId }
    val nextDraftProjectId = if (state.draft.projectId == projectId) {
        ""
    } else {
        state.draft.projectId
    }

    return state.copy(
        projects = remainingProjects,
        entries = state.entries.map { entry ->
            if (entry.projectId == projectId) entry.copy(projectId = "") else entry
        },
        draft = state.draft.copy(projectId = nextDraftProjectId),
        scheduleFocusEntryId = null,
        notice = "Deleted project.",
    )
}

private fun addDraftEntryStarted(state: TimeboxxingScreenState): TimeboxxingScreenState {
    val draft = state.draft
    if (state.entrySaving) return state
    if (!state.hasValidDraftProject()) {
        return state.copy(
            scheduleFocusEntryId = null,
            notice = InvalidDraftProjectNotice,
        )
    }

    return state.copy(
        entrySaving = true,
        scheduleFocusEntryId = null,
        notice = null,
    )
}

private fun addDraftEntrySucceeded(
    state: TimeboxxingScreenState,
    dayStartedAtEpochMillis: Long,
    entry: TimeEntry,
): TimeboxxingScreenState {
    if (dayStartedAtEpochMillis != state.selectedDay.startedAtEpochMillis) {
        return state.copy(entrySaving = false)
    }

    return state.copy(
        entries = state.entries + entry,
        entrySaving = false,
        scheduleFocusEntryId = entry.id,
        selectedUsageIds = emptySet(),
        draft = blankDraft(state.draft.projectId).copy(startMinute = state.draft.startMinute + state.draft.durationMinutes),
        notice = "Added ${formatDuration(entry.durationMinutes)} to ${state.projectFor(entry.projectId).name}.",
        nextEntryNumber = state.nextEntryNumber + 1,
    )
}

private fun duplicateEntryStarted(
    state: TimeboxxingScreenState,
    entryId: String,
): TimeboxxingScreenState {
    if (state.entrySaving) return state
    state.entries.firstOrNull { it.id == entryId } ?: return state
    return state.copy(
        entrySaving = true,
        scheduleFocusEntryId = null,
        notice = null,
    )
}

private fun duplicateEntrySucceeded(
    state: TimeboxxingScreenState,
    dayStartedAtEpochMillis: Long,
    sourceTitle: String,
    entry: TimeEntry,
): TimeboxxingScreenState {
    if (dayStartedAtEpochMillis != state.selectedDay.startedAtEpochMillis) {
        return state.copy(entrySaving = false)
    }

    return state.copy(
        entries = state.entries + entry,
        entrySaving = false,
        scheduleFocusEntryId = entry.id,
        nextEntryNumber = state.nextEntryNumber + 1,
        notice = "Duplicated $sourceTitle.",
    )
}

private fun deleteEntryStarted(
    state: TimeboxxingScreenState,
    entryId: String,
): TimeboxxingScreenState {
    if (state.entries.none { it.id == entryId }) return state
    if (entryId in state.deletingEntryIds) return state
    return state.copy(
        deletingEntryIds = state.deletingEntryIds + entryId,
        scheduleFocusEntryId = null,
        notice = null,
    )
}

private fun exportTimesheetStarted(state: TimeboxxingScreenState): TimeboxxingScreenState {
    if (state.entries.isEmpty()) {
        return state.copy(
            timesheetExporting = false,
            scheduleFocusEntryId = null,
            notice = AddEntryBeforeExportNotice,
        )
    }
    return state.copy(
        timesheetExporting = true,
        scheduleFocusEntryId = null,
        notice = null,
    )
}

private fun TimeboxxingScreenState.hasValidDraftProject(): Boolean =
    draft.projectId.isBlank() || projects.any { it.id == draft.projectId }

private fun Set<String>.toggle(value: String): Set<String> =
    if (value in this) this - value else this + value

private fun submitAmaQuestion(state: TimeboxxingScreenState): TimeboxxingScreenState {
    val question = state.amaInput.trim()
    if (question.isEmpty() || state.amaLoading) {
        return state
    }

    val message = AmaMessage(
        id = "ama-${state.nextAmaMessageNumber}",
        role = AmaMessageRole.User,
        content = question,
    )
    return state.copy(
        selectedSection = TimeboxxingSection.Ama,
        amaInput = "",
        amaMessages = state.amaMessages + message,
        amaLoading = true,
        amaError = null,
        nextAmaMessageNumber = state.nextAmaMessageNumber + 1,
    )
}

private fun submitAmaStructuredQuery(
    state: TimeboxxingScreenState,
    query: AmaStructuredQuery,
): TimeboxxingScreenState {
    if (state.amaLoading) {
        return state
    }

    val message = AmaMessage(
        id = "ama-${state.nextAmaMessageNumber}",
        role = AmaMessageRole.User,
        content = query.userFacingQuestion(),
    )
    return state.copy(
        selectedSection = TimeboxxingSection.Ama,
        amaInput = "",
        amaMessages = state.amaMessages + message,
        amaLoading = true,
        amaError = null,
        nextAmaMessageNumber = state.nextAmaMessageNumber + 1,
    )
}

private fun AmaStructuredQuery.userFacingQuestion(): String {
    val label = periodLabel.ifBlank { "selected period" }
    return when (kind) {
        AmaQueryKind.AppTotals -> "Show app totals for $label"
        AmaQueryKind.Timeline -> "Show usage timeline for $label"
        AmaQueryKind.Habits -> "Summarize habits for $label"
        AmaQueryKind.ComparePeriods -> {
            val baseline = baselinePeriodLabel.ifBlank { "baseline period" }
            "Compare $label with $baseline"
        }
    }
}

private fun openAmaUsageSource(
    state: TimeboxxingScreenState,
    startedAtEpochMillis: Long?,
    transitionEventId: Long,
): TimeboxxingScreenState {
    val startedAt = startedAtEpochMillis ?: return state.copy(
        selectedSection = TimeboxxingSection.Overview,
        notice = "That AMA source does not include a schedule time.",
    )
    val date = calendarDateForEpochMillis(startedAt)
    val day = usageDayForCalendarDate(date)
    val minute = ((startedAt - day.startedAtEpochMillis) / 60_000L)
        .toInt()
        .coerceIn(0, UsageDayMinutes - 1)
    val usageId = transitionEventId
        .takeIf { it > 0L }
        ?.let { "$SidecarUsageIdPrefix$it" }
    val target = ScheduleFocusTarget(
        date = date,
        minute = minute,
        usageId = usageId,
        requestId = state.nextScheduleFocusRequestId,
    )
    val selectedDateState = selectDate(state, date)
    val selectedUsageIds = focusedUsageSelection(target, selectedDateState.usageEvents)
    return selectedDateState.copy(
        selectedSection = TimeboxxingSection.Overview,
        scheduleFocusEntryId = null,
        scheduleFocusTarget = target,
        selectedUsageIds = selectedUsageIds,
        nextScheduleFocusRequestId = state.nextScheduleFocusRequestId + 1,
        notice = null,
    )
}

private fun focusedUsageSelection(
    target: ScheduleFocusTarget?,
    events: List<UsageEvent>,
): Set<String> {
    val usageId = target?.usageId ?: return emptySet()
    return if (events.firstOrNull { it.id == usageId }?.isSelectableUsage() == true) {
        setOf(usageId)
    } else {
        emptySet()
    }
}

private fun mergedFocusedUsageSelection(
    state: TimeboxxingScreenState,
    events: List<UsageEvent>,
): Set<String> {
    val focused = focusedUsageSelection(state.scheduleFocusTarget, events)
    if (focused.isNotEmpty()) {
        return focused
    }
    return state.selectedUsageIds.intersect(events.selectableUsageIds())
}
