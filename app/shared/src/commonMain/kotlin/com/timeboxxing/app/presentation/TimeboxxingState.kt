package com.timeboxxing.app.presentation

import com.timeboxxing.data.mock.mockTimeboxxingData
import com.timeboxxing.data.time.usageDayForCalendarDate
import com.timeboxxing.domain.model.AmaAnswer
import com.timeboxxing.domain.model.AmaIndexState
import com.timeboxxing.domain.model.AmaIndexStatus
import com.timeboxxing.domain.model.AmaMessage
import com.timeboxxing.domain.model.AmaMessageRole
import com.timeboxxing.domain.model.AiModelOptions
import com.timeboxxing.domain.model.AiProvider
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.EntryDraft
import com.timeboxxing.domain.model.EntryMode
import com.timeboxxing.domain.model.Project
import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.TimeboxxingMockData
import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.model.UsageEvent
import com.timeboxxing.domain.model.UsageSourceType
import com.timeboxxing.domain.model.formatDuration
import com.timeboxxing.domain.model.plusDays

data class TimeboxxingScreenState(
    val selectedSection: TimeboxxingSection,
    val dateIndex: Int,
    val zoomMinutes: Int,
    val mode: EntryMode,
    val projects: List<Project>,
    val usageEvents: List<UsageEvent>,
    val usageLoading: Boolean,
    val entries: List<TimeEntry>,
    val scheduleFocusEntryId: String?,
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
    val settingsOptions: AiModelOptions = AiModelOptions(),
    val aiSettings: AiSettings = AiSettings(),
    val settingsDraft: AiSettings = AiSettings(),
    val settingsLoading: Boolean = false,
    val settingsSaving: Boolean = false,
    val settingsError: String? = null,
    val settingsSavedMessage: String? = null,
    val appearanceMode: AppearanceMode = AppearanceMode.System,
    val diagnosticsEnabled: Boolean = false,
) {
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

    val primaryActionLabel: String
        get() = when (mode) {
            EntryMode.Timesheet -> "Create timesheet"
            EntryMode.Invoice -> "Create invoice"
        }

    val isAmaConfigured: Boolean
        get() = aiSettings.hasConfiguredAiSecret()

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
        projects.firstOrNull { it.id == projectId } ?: projects.first()

    fun minutesForProject(projectId: String): Int =
        entries.filter { it.projectId == projectId }.sumOf { it.durationMinutes }

    fun invoiceTotalCents(): Int =
        entries.filter { it.billable }.sumOf { entry ->
            val project = projectFor(entry.projectId)
            project.hourlyRateCents * entry.durationMinutes / 60
        }
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
    data class ChangeMode(val mode: EntryMode) : TimeboxxingAction
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
    data object AddDraftEntry : TimeboxxingAction
    data class DuplicateEntry(val entryId: String) : TimeboxxingAction
    data class DeleteEntry(val entryId: String) : TimeboxxingAction
    data object ConfirmPrimaryAction : TimeboxxingAction
    data object DismissNotice : TimeboxxingAction
    data class UpdateAmaInput(val input: String) : TimeboxxingAction
    data object SubmitAmaQuestion : TimeboxxingAction
    data class AmaAnswerSucceeded(val answer: AmaAnswer) : TimeboxxingAction
    data class AmaAnswerFailed(val message: String) : TimeboxxingAction
    data class AmaIndexStatusSucceeded(val status: AmaIndexStatus) : TimeboxxingAction
    data class AmaIndexStatusFailed(val message: String) : TimeboxxingAction
    data object ClearAmaChat : TimeboxxingAction
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
}

fun createInitialTimeboxxingState(
    data: TimeboxxingMockData = mockTimeboxxingData(),
    appearanceMode: AppearanceMode = AppearanceMode.System,
): TimeboxxingScreenState {
    val defaultProject = data.projects.first()
    return TimeboxxingScreenState(
        selectedSection = TimeboxxingSection.Overview,
        dateIndex = data.usageDays.indexOfFirst { it.label == "Thursday, May 1, 2025" }.takeIf { it >= 0 } ?: 0,
        zoomMinutes = 15,
        mode = EntryMode.Timesheet,
        projects = data.projects,
        usageEvents = data.usageEvents,
        usageLoading = false,
        entries = data.initialEntries,
        scheduleFocusEntryId = null,
        selectedUsageIds = emptySet(),
        draft = blankDraft(defaultProject.id),
        notice = null,
        nextEntryNumber = data.initialEntries.size + 1,
        usageDays = data.usageDays,
        appearanceMode = appearanceMode,
    )
}

fun createSidecarTimeboxxingState(
    usageDays: List<UsageDay>,
    initialNotice: String? = null,
    data: TimeboxxingMockData = mockTimeboxxingData(),
    appearanceMode: AppearanceMode = AppearanceMode.System,
): TimeboxxingScreenState {
    val defaultProject = data.projects.first()
    val safeUsageDays = usageDays.ifEmpty { data.usageDays }
    return TimeboxxingScreenState(
        selectedSection = TimeboxxingSection.Overview,
        dateIndex = (safeUsageDays.size / 2).coerceAtMost(safeUsageDays.lastIndex),
        zoomMinutes = 15,
        mode = EntryMode.Timesheet,
        projects = data.projects,
        usageEvents = emptyList(),
        usageLoading = true,
        entries = emptyList(),
        scheduleFocusEntryId = null,
        selectedUsageIds = emptySet(),
        draft = blankDraft(defaultProject.id),
        notice = initialNotice,
        nextEntryNumber = 1,
        usageDays = safeUsageDays,
        appearanceMode = appearanceMode,
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
            )
        }

        is TimeboxxingAction.MoveDate -> moveDate(state, action.delta)

        is TimeboxxingAction.SelectDate -> selectDate(state, action.date)

        is TimeboxxingAction.ChangeZoom -> state.copy(
            zoomMinutes = action.minutes.coerceIn(5, 60),
            notice = null,
        )

        is TimeboxxingAction.ChangeMode -> state.copy(
            mode = action.mode,
            scheduleFocusEntryId = null,
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
                state.copy(
                    usageEvents = normalizeUsageEvents(action.events),
                    usageLoading = false,
                    scheduleFocusEntryId = null,
                    selectedUsageIds = emptySet(),
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
                    selectedUsageIds = state.selectedUsageIds.intersect(mergedEvents.selectableUsageIds()),
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
                    notice = null,
                )
            }
        }

        TimeboxxingAction.ClearUsageSelection -> state.copy(
            selectedUsageIds = emptySet(),
            draft = blankDraft(state.draft.projectId),
            scheduleFocusEntryId = null,
            notice = null,
        )

        TimeboxxingAction.NewBlankDraft -> state.copy(
            selectedUsageIds = emptySet(),
            draft = blankDraft(state.draft.projectId),
            scheduleFocusEntryId = null,
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

        TimeboxxingAction.AddDraftEntry -> addEntryFromDraft(state)

        is TimeboxxingAction.DuplicateEntry -> duplicateEntry(state, action.entryId)

        is TimeboxxingAction.DeleteEntry -> state.copy(
            entries = state.entries.filterNot { it.id == action.entryId },
            scheduleFocusEntryId = null,
            notice = null,
        )

        TimeboxxingAction.ConfirmPrimaryAction -> state.copy(
            scheduleFocusEntryId = null,
            notice = "${state.primaryActionLabel} draft ready: ${state.entries.size} entries, ${formatDuration(state.billableMinutes)} billable.",
        )

        TimeboxxingAction.DismissNotice -> state.copy(scheduleFocusEntryId = null, notice = null)

        is TimeboxxingAction.UpdateAmaInput -> state.copy(
            amaInput = action.input,
            amaError = null,
        )

        TimeboxxingAction.SubmitAmaQuestion -> submitAmaQuestion(state)

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
        scheduleFocusEntryId = null,
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
    val hintedProjectId = selectedEvents.firstNotNullOfOrNull { it.projectHintId } ?: state.draft.projectId
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
    val completed = events.filterNot { it.isActive }
    val active = events.lastOrNull { it.isActive }
        ?.takeUnless { activeEvent -> completed.any { activeEvent.matchesCompletedUsage(it) } }
    return (completed + listOfNotNull(active)).sortedBy { it.startMinute }
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
    !isActive && !isIdleUsage()

private fun UsageEvent.isCapturedUsage(): Boolean =
    !isActive && !isIdleUsage()

private fun UsageEvent.isIdleUsage(): Boolean =
    sourceType == UsageSourceType.Idle

private fun addEntryFromDraft(state: TimeboxxingScreenState): TimeboxxingScreenState {
    val draft = state.draft
    val entry = TimeEntry(
        id = "entry-${state.nextEntryNumber}",
        projectId = draft.projectId,
        title = draft.title.ifBlank { "New time entry" },
        notes = draft.notes,
        startMinute = draft.startMinute,
        durationMinutes = draft.durationMinutes,
        billable = draft.billable,
        sourceUsageIds = state.selectedUsageIds,
    )

    return state.copy(
        entries = state.entries + entry,
        scheduleFocusEntryId = entry.id,
        selectedUsageIds = emptySet(),
        draft = blankDraft(draft.projectId).copy(startMinute = draft.startMinute + draft.durationMinutes),
        notice = "Added ${formatDuration(entry.durationMinutes)} to ${state.projectFor(entry.projectId).name}.",
        nextEntryNumber = state.nextEntryNumber + 1,
    )
}

private fun duplicateEntry(
    state: TimeboxxingScreenState,
    entryId: String,
): TimeboxxingScreenState {
    val source = state.entries.firstOrNull { it.id == entryId } ?: return state
    val duplicate = source.copy(
        id = "entry-${state.nextEntryNumber}",
        title = "Copy of ${source.title}",
        startMinute = source.startMinute + source.durationMinutes,
        sourceUsageIds = emptySet(),
    )

    return state.copy(
        entries = state.entries + duplicate,
        scheduleFocusEntryId = duplicate.id,
        nextEntryNumber = state.nextEntryNumber + 1,
        notice = "Duplicated ${source.title}.",
    )
}

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
