package com.timeboxxing.app.state

import com.timeboxxing.app.data.mockTimeboxxingData
import com.timeboxxing.app.data.usageDayForCalendarDate
import com.timeboxxing.app.model.AmaAnswer
import com.timeboxxing.app.model.AmaIndexStatus
import com.timeboxxing.app.model.AmaMessage
import com.timeboxxing.app.model.AmaMessageRole
import com.timeboxxing.app.model.CalendarDate
import com.timeboxxing.app.model.EntryDraft
import com.timeboxxing.app.model.EntryMode
import com.timeboxxing.app.model.Project
import com.timeboxxing.app.model.TimeEntry
import com.timeboxxing.app.model.TimeboxxingMockData
import com.timeboxxing.app.model.UsageDay
import com.timeboxxing.app.model.UsageEvent
import com.timeboxxing.app.model.formatDuration
import com.timeboxxing.app.model.plusDays

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
    val collapsedPanes: Set<WorkspacePane> = emptySet(),
    val amaInput: String = "",
    val amaMessages: List<AmaMessage> = emptyList(),
    val amaLoading: Boolean = false,
    val amaError: String? = null,
    val amaIndexStatus: AmaIndexStatus? = null,
    val nextAmaMessageNumber: Int = 1,
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
        get() = usageEvents.filter { !it.isActive && it.id in selectedUsageIds }.sortedBy { it.startMinute }

    val selectedUsageMinutes: Int
        get() = selectedUsageEvents.sumOf { it.durationMinutes }

    val billableMinutes: Int
        get() = entries.filter { it.billable }.sumOf { it.durationMinutes }

    val capturedMinutes: Int
        get() = usageEvents.sumOf { it.durationMinutes }

    val unassignedUsageMinutes: Int
        get() {
            val assignedUsageIds = entries.flatMap { it.sourceUsageIds }.toSet()
            return usageEvents.filterNot { it.id in assignedUsageIds }.sumOf { it.durationMinutes }
        }

    val primaryActionLabel: String
        get() = when (mode) {
            EntryMode.Timesheet -> "Create timesheet"
            EntryMode.Invoice -> "Create invoice"
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
}

enum class WorkspacePane {
    UsageSchedule,
    TimeEntries,
    Projects,
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
    data class ToggleWorkspacePaneCollapsed(val pane: WorkspacePane) : TimeboxxingAction
    data class SetCollapsedWorkspacePanes(val panes: Set<WorkspacePane>) : TimeboxxingAction
    data class UpdateAmaInput(val input: String) : TimeboxxingAction
    data object SubmitAmaQuestion : TimeboxxingAction
    data class AmaAnswerSucceeded(val answer: AmaAnswer) : TimeboxxingAction
    data class AmaAnswerFailed(val message: String) : TimeboxxingAction
    data class AmaIndexStatusSucceeded(val status: AmaIndexStatus) : TimeboxxingAction
    data class AmaIndexStatusFailed(val message: String) : TimeboxxingAction
    data object ClearAmaChat : TimeboxxingAction
}

fun createInitialTimeboxxingState(
    data: TimeboxxingMockData = mockTimeboxxingData(),
    collapsedPanes: Set<WorkspacePane> = emptySet(),
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
        collapsedPanes = sanitizeCollapsedWorkspacePanes(collapsedPanes),
    )
}

fun createSidecarTimeboxxingState(
    usageDays: List<UsageDay>,
    initialNotice: String? = null,
    data: TimeboxxingMockData = mockTimeboxxingData(),
    collapsedPanes: Set<WorkspacePane> = emptySet(),
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
        collapsedPanes = sanitizeCollapsedWorkspacePanes(collapsedPanes),
    )
}

fun reduceTimeboxxingState(
    state: TimeboxxingScreenState,
    action: TimeboxxingAction,
): TimeboxxingScreenState =
    when (action) {
        is TimeboxxingAction.SelectSection -> state.copy(
            selectedSection = action.section,
            amaError = if (action.section == TimeboxxingSection.Ama) state.amaError else null,
            notice = if (action.section == TimeboxxingSection.Overview) state.notice else null,
        )

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
                    selectedUsageIds = state.selectedUsageIds.intersect(mergedEvents.completedUsageIds()),
                )
            }
        }

        is TimeboxxingAction.ToggleUsageSelection -> {
            if (state.usageEvents.firstOrNull { it.id == action.usageId }?.isActive == true) {
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

        is TimeboxxingAction.ToggleWorkspacePaneCollapsed -> {
            val nextPanes = if (action.pane in state.collapsedPanes) {
                state.collapsedPanes - action.pane
            } else {
                state.collapsedPanes + action.pane
            }
            state.copy(collapsedPanes = sanitizeCollapsedWorkspacePanes(nextPanes))
        }

        is TimeboxxingAction.SetCollapsedWorkspacePanes -> state.copy(
            collapsedPanes = sanitizeCollapsedWorkspacePanes(action.panes),
        )

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
            amaIndexStatus = state.amaIndexStatus,
            amaError = action.message,
        )

        TimeboxxingAction.ClearAmaChat -> state.copy(
            amaInput = "",
            amaMessages = emptyList(),
            amaLoading = false,
            amaError = null,
            amaIndexStatus = state.amaIndexStatus,
            nextAmaMessageNumber = 1,
        )
    }

fun sanitizeCollapsedWorkspacePanes(panes: Set<WorkspacePane>): Set<WorkspacePane> {
    val validPanes = panes.intersect(WorkspacePane.entries.toSet())
    return if (validPanes.size >= WorkspacePane.entries.size) {
        validPanes - WorkspacePane.UsageSchedule
    } else {
        validPanes
    }
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
    val selectedEvents = state.usageEvents.filter { !it.isActive && it.id in selectedUsageIds }.sortedBy { it.startMinute }
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

private fun List<UsageEvent>.completedUsageIds(): Set<String> =
    filterNot { it.isActive }.map { it.id }.toSet()

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
