package com.timeboxxing.app

import androidx.compose.ui.graphics.Color
import com.timeboxxing.domain.model.AmaAnswer
import com.timeboxxing.domain.model.AmaAppUsageBucket
import com.timeboxxing.domain.model.AmaAppUsageChart
import com.timeboxxing.domain.model.AmaIndexState
import com.timeboxxing.domain.model.AmaIndexStatus
import com.timeboxxing.domain.model.AmaMessageRole
import com.timeboxxing.domain.model.AmaSource
import com.timeboxxing.data.mock.mockTimeboxxingData
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.EntryMode
import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.UsageEvent
import com.timeboxxing.domain.model.UsageSourceType
import com.timeboxxing.domain.model.formatClockTime
import com.timeboxxing.domain.model.formatDuration
import com.timeboxxing.app.presentation.TimeboxxingAction
import com.timeboxxing.app.presentation.TimeboxxingSection
import com.timeboxxing.app.presentation.createInitialTimeboxxingState
import com.timeboxxing.app.presentation.reduceTimeboxxingState
import com.timeboxxing.app.ui.buildTimelineGrid
import com.timeboxxing.app.ui.timelineFirstEventScrollDp
import com.timeboxxing.app.ui.timelineDpPerMinuteForZoom
import com.timeboxxing.app.ui.timelineEntryScrollDp
import com.timeboxxing.app.ui.timelineMinuteOffsetDp
import com.timeboxxing.app.ui.timelineNowScrollDirection
import com.timeboxxing.app.ui.timelineRowHeightDpForZoom
import com.timeboxxing.app.ui.timelineScrollToMinuteDp
import com.timeboxxing.app.ui.timelineSegmentsForRow
import com.timeboxxing.app.ui.timelineIntervalMarkerOffsets
import com.timeboxxing.app.ui.SchedulePaneAutoScrollState
import com.timeboxxing.app.ui.TimelineScrollDirection
import com.timeboxxing.app.ui.TbDarkColors
import com.timeboxxing.app.ui.TbLightColors
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue
import kotlin.math.abs
import kotlin.math.max
import kotlin.math.min
import kotlin.math.pow

class TimeboxxingReducerTest {

    @Test
    fun appearanceModeDefaultsToSystem() {
        val initial = createInitialTimeboxxingState()

        assertEquals(AppearanceMode.System, initial.appearanceMode)
    }

    @Test
    fun updatingAppearanceModeChangesOnlyAppearanceState() {
        val initial = createInitialTimeboxxingState().copy(
            settingsError = "Old error",
            settingsSavedMessage = "Saved",
        )

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.UpdateAppearanceMode(AppearanceMode.Dark))

        assertEquals(AppearanceMode.Dark, state.appearanceMode)
        assertEquals(initial.settingsDraft, state.settingsDraft)
        assertEquals(initial.aiSettings, state.aiSettings)
        assertEquals(null, state.settingsError)
        assertEquals(null, state.settingsSavedMessage)
    }

    @Test
    fun selectingUsageBuildsDraftFromUsageHistory() {
        val initial = createInitialTimeboxxingState()

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.ToggleUsageSelection("usage-intro-email"),
        )

        assertEquals(setOf("usage-intro-email"), state.selectedUsageIds)
        assertEquals("axion", state.draft.projectId)
        assertEquals("Re: Intro Axion Ltd.", state.draft.title)
        assertEquals(20, state.draft.durationMinutes)
        assertTrue(state.draft.notes.contains("Gmail"))
    }

    @Test
    fun defaultSettingsHideAmaNavigation() {
        val initial = createInitialTimeboxxingState()

        assertFalse(initial.isAmaConfigured)
        assertFalse(TimeboxxingSection.Ama in initial.visibleNavigationSections)
        assertFalse(TimeboxxingSection.Diagnostics in initial.visibleNavigationSections)
        assertEquals(listOf(TimeboxxingSection.Overview, TimeboxxingSection.Settings), initial.visibleNavigationSections)
    }

    @Test
    fun diagnosticsNavigationIsVisibleOnlyWhenEnabled() {
        val disabled = createInitialTimeboxxingState()
        val enabled = disabled.copy(diagnosticsEnabled = true)

        assertFalse(TimeboxxingSection.Diagnostics in disabled.visibleNavigationSections)
        assertEquals(
            listOf(TimeboxxingSection.Overview, TimeboxxingSection.Diagnostics, TimeboxxingSection.Settings),
            enabled.visibleNavigationSections,
        )
    }

    @Test
    fun selectingDiagnosticsRequiresDiagnosticsFlag() {
        val disabled = createInitialTimeboxxingState()
        val enabled = disabled.copy(diagnosticsEnabled = true)

        val blocked = reduceTimeboxxingState(disabled, TimeboxxingAction.SelectSection(TimeboxxingSection.Diagnostics))
        val selected = reduceTimeboxxingState(enabled, TimeboxxingAction.SelectSection(TimeboxxingSection.Diagnostics))

        assertEquals(TimeboxxingSection.Overview, blocked.selectedSection)
        assertEquals(TimeboxxingSection.Diagnostics, selected.selectedSection)
    }

    @Test
    fun openRouterSecretShowsAmaNavigation() {
        val state = createInitialTimeboxxingState().copy(
            aiSettings = AiSettings(openRouterSecretExists = true),
        )

        assertTrue(state.isAmaConfigured)
        assertTrue(TimeboxxingSection.Ama in state.visibleNavigationSections)
    }

    @Test
    fun amaAndDiagnosticsNavigationUseExplicitOrder() {
        val state = createInitialTimeboxxingState().copy(
            aiSettings = AiSettings(openRouterSecretExists = true),
            diagnosticsEnabled = true,
        )

        assertEquals(
            listOf(
                TimeboxxingSection.Overview,
                TimeboxxingSection.Ama,
                TimeboxxingSection.Diagnostics,
                TimeboxxingSection.Settings,
            ),
            state.visibleNavigationSections,
        )
    }

    @Test
    fun ollamaSecretShowsAmaNavigation() {
        val state = createInitialTimeboxxingState().copy(
            aiSettings = AiSettings(ollamaSecretExists = true),
        )

        assertTrue(state.isAmaConfigured)
        assertTrue(TimeboxxingSection.Ama in state.visibleNavigationSections)
    }

    @Test
    fun selectingAmaWithoutConfiguredSecretRedirectsToSettings() {
        val initial = createInitialTimeboxxingState()

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.SelectSection(TimeboxxingSection.Ama))

        assertEquals(TimeboxxingSection.Overview, initial.selectedSection)
        assertEquals(TimeboxxingSection.Settings, state.selectedSection)
    }

    @Test
    fun selectingAmaWithConfiguredSecretStillWorks() {
        val initial = createAmaConfiguredState()

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.SelectSection(TimeboxxingSection.Ama))

        assertEquals(TimeboxxingSection.Overview, initial.selectedSection)
        assertEquals(TimeboxxingSection.Ama, state.selectedSection)
    }

    @Test
    fun submittingAmaQuestionAddsUserMessageAndClearsInput() {
        val withInput = reduceTimeboxxingState(
            createAmaConfiguredState(),
            TimeboxxingAction.UpdateAmaInput("  What did I do today?  "),
        )

        val state = reduceTimeboxxingState(withInput, TimeboxxingAction.SubmitAmaQuestion)

        assertEquals(TimeboxxingSection.Ama, state.selectedSection)
        assertEquals("", state.amaInput)
        assertTrue(state.amaLoading)
        assertEquals(1, state.amaMessages.size)
        assertEquals(AmaMessageRole.User, state.amaMessages.first().role)
        assertEquals("What did I do today?", state.amaMessages.first().content)
    }

    @Test
    fun amaAnswerSuccessAddsAssistantMessageWithSources() {
        val submitted = reduceTimeboxxingState(
            reduceTimeboxxingState(createAmaConfiguredState(), TimeboxxingAction.UpdateAmaInput("Question")),
            TimeboxxingAction.SubmitAmaQuestion,
        )

        val state = reduceTimeboxxingState(
            submitted,
            TimeboxxingAction.AmaAnswerSucceeded(
                AmaAnswer(
                    answer = "You used Chrome.",
                    model = "google/gemma-4-31b-it:free",
                    sources = listOf(
                        AmaSource(
                            transitionEventId = 42,
                            documentKey = "event:42",
                            documentType = "event",
                            startedAtEpochMillis = null,
                            endedAtEpochMillis = null,
                            content = "Application: Chrome",
                            distance = 0.1,
                        ),
                    ),
                    artifacts = listOf(
                        AmaAppUsageChart(
                            periodLabel = "Today",
                            startedAtEpochMillis = null,
                            endedAtEpochMillis = null,
                            timeZone = "UTC",
                            totalDurationSeconds = 3600,
                            buckets = listOf(
                                AmaAppUsageBucket(
                                    name = "Chrome",
                                    sourceType = "browser",
                                    durationSeconds = 3600,
                                    sessionCount = 1,
                                    applicationIdentifier = "com.google.Chrome",
                                    applicationPath = "/Applications/Google Chrome.app",
                                ),
                            ),
                        ),
                    ),
                ),
            ),
        )

        assertFalse(state.amaLoading)
        assertEquals(null, state.amaError)
        assertEquals(2, state.amaMessages.size)
        assertEquals(AmaMessageRole.Assistant, state.amaMessages.last().role)
        assertEquals("You used Chrome.", state.amaMessages.last().content)
        assertEquals(42, state.amaMessages.last().sources.first().transitionEventId)
        assertEquals(1, state.amaMessages.last().artifacts.size)
    }

    @Test
    fun amaAnswerFailureKeepsQuestionAndShowsError() {
        val submitted = reduceTimeboxxingState(
            reduceTimeboxxingState(createAmaConfiguredState(), TimeboxxingAction.UpdateAmaInput("Question")),
            TimeboxxingAction.SubmitAmaQuestion,
        )

        val state = reduceTimeboxxingState(submitted, TimeboxxingAction.AmaAnswerFailed("Sidecar unavailable"))

        assertFalse(state.amaLoading)
        assertEquals("Sidecar unavailable", state.amaError)
        assertEquals(1, state.amaMessages.size)
    }

    @Test
    fun amaIndexStatusFailureShowsUnavailableBadgeWithoutAnswerError() {
        val ready = reduceTimeboxxingState(
            createAmaConfiguredState(),
            TimeboxxingAction.AmaIndexStatusSucceeded(
                AmaIndexStatus(
                    state = AmaIndexState.Ready,
                    completedEventCount = 10L,
                    indexedEventCount = 8L,
                    pendingEventCount = 2L,
                    backfillRunning = false,
                    message = "Semantic index is ready.",
                ),
            ),
        )

        val state = reduceTimeboxxingState(
            ready,
            TimeboxxingAction.AmaIndexStatusFailed("Semantic index status is unavailable."),
        )

        assertEquals(null, state.amaError)
        assertEquals(AmaIndexState.Unavailable, state.amaIndexStatus?.state)
        assertEquals("Semantic index status is unavailable.", state.amaIndexStatus?.message)
        assertEquals(10L, state.amaIndexStatus?.completedEventCount)
        assertEquals(8L, state.amaIndexStatus?.indexedEventCount)
        assertEquals(2L, state.amaIndexStatus?.pendingEventCount)
    }

    @Test
    fun clearingAmaChatResetsChatState() {
        val submitted = reduceTimeboxxingState(
            reduceTimeboxxingState(createAmaConfiguredState(), TimeboxxingAction.UpdateAmaInput("Question")),
            TimeboxxingAction.SubmitAmaQuestion,
        )

        val state = reduceTimeboxxingState(submitted, TimeboxxingAction.ClearAmaChat)

        assertTrue(state.amaMessages.isEmpty())
        assertFalse(state.amaLoading)
        assertEquals(null, state.amaError)
        assertEquals("", state.amaInput)
    }

    @Test
    fun selectedUsageMinutesCountsSelectedEventsOnly() {
        val selectedOne = reduceTimeboxxingState(
            createInitialTimeboxxingState(),
            TimeboxxingAction.ToggleUsageSelection("usage-intro-email"),
        )
        val selectedTwo = reduceTimeboxxingState(
            selectedOne,
            TimeboxxingAction.ToggleUsageSelection("usage-daven-chat"),
        )

        assertEquals(20, selectedOne.selectedUsageMinutes)
        assertEquals(35, selectedTwo.selectedUsageMinutes)
    }

    @Test
    fun addingDraftCreatesEntryAndClearsSelection() {
        val selected = reduceTimeboxxingState(
            createInitialTimeboxxingState(),
            TimeboxxingAction.ToggleUsageSelection("usage-daven-chat"),
        )
        val beforeCount = selected.entries.size

        val state = reduceTimeboxxingState(selected, TimeboxxingAction.AddDraftEntry)

        assertEquals(beforeCount + 1, state.entries.size)
        assertTrue(state.selectedUsageIds.isEmpty())
        assertEquals("daven", state.entries.last().projectId)
        assertEquals(setOf("usage-daven-chat"), state.entries.last().sourceUsageIds)
        assertEquals(state.entries.last().id, state.scheduleFocusEntryId)
    }

    @Test
    fun duplicatingEntryFocusesDuplicateOnSchedule() {
        val initial = createInitialTimeboxxingState()
        val source = initial.entries.first()

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.DuplicateEntry(source.id))

        assertEquals(initial.entries.size + 1, state.entries.size)
        assertEquals(state.entries.last().id, state.scheduleFocusEntryId)
        assertEquals(source.startMinute + source.durationMinutes, state.entries.last().startMinute)
    }

    @Test
    fun deletingFocusedEntryClearsScheduleFocus() {
        val added = reduceTimeboxxingState(createInitialTimeboxxingState(), TimeboxxingAction.AddDraftEntry)

        val state = reduceTimeboxxingState(
            added,
            TimeboxxingAction.DeleteEntry(added.scheduleFocusEntryId.orEmpty()),
        )

        assertEquals(null, state.scheduleFocusEntryId)
    }

    @Test
    fun movingDateClearsScheduleFocus() {
        val added = reduceTimeboxxingState(createInitialTimeboxxingState(), TimeboxxingAction.AddDraftEntry)

        val state = reduceTimeboxxingState(added, TimeboxxingAction.MoveDate(1))

        assertEquals(null, state.scheduleFocusEntryId)
    }

    @Test
    fun selectingArbitraryDateAddsAndSelectsUsageDay() {
        val initial = createInitialTimeboxxingState()
        val selectedDate = CalendarDate(2026, 7, 4)

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.SelectDate(selectedDate))

        assertEquals(selectedDate, state.selectedCalendarDate)
        assertEquals("Saturday, July 4, 2026", state.dateLabel)
        assertTrue(state.usageDays.any { it.calendarDate == selectedDate })
        assertEquals(
            state.usageDays.sortedBy { it.startedAtEpochMillis },
            state.usageDays,
        )
    }

    @Test
    fun movingDateCanGoPastOriginalRange() {
        val initial = createInitialTimeboxxingState()

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.MoveDate(2))

        assertEquals(CalendarDate(2025, 5, 3), state.selectedCalendarDate)
        assertEquals("Saturday, May 3, 2025", state.dateLabel)
        assertTrue(state.usageDays.any { it.calendarDate == CalendarDate(2025, 5, 3) })
    }

    @Test
    fun dateSelectionClearsLoadedDayState() {
        val withSelection = reduceTimeboxxingState(
            reduceTimeboxxingState(createInitialTimeboxxingState(), TimeboxxingAction.AddDraftEntry),
            TimeboxxingAction.ToggleUsageSelection("usage-intro-email"),
        ).copy(notice = "Ready")

        val state = reduceTimeboxxingState(
            withSelection,
            TimeboxxingAction.SelectDate(CalendarDate(2026, 7, 4)),
        )

        assertTrue(state.usageEvents.isEmpty())
        assertTrue(state.usageLoading)
        assertEquals(null, state.scheduleFocusEntryId)
        assertTrue(state.selectedUsageIds.isEmpty())
        assertEquals("New time entry", state.draft.title)
        assertEquals(null, state.notice)
    }

    @Test
    fun selectingCurrentDateIsNoOp() {
        val initial = createInitialTimeboxxingState()

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.SelectDate(initial.selectedCalendarDate ?: error("Expected a selected date")),
        )

        assertEquals(initial, state)
    }

    @Test
    fun deletingEntryUpdatesTotals() {
        val initial = createInitialTimeboxxingState()
        val entry = initial.entries.first()

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.DeleteEntry(entry.id),
        )

        assertFalse(state.entries.any { it.id == entry.id })
        assertEquals(initial.billableMinutes - entry.durationMinutes, state.billableMinutes)
    }

    @Test
    fun modeChangesPrimaryActionLabel() {
        val state = reduceTimeboxxingState(
            createInitialTimeboxxingState(),
            TimeboxxingAction.ChangeMode(EntryMode.Invoice),
        )

        assertEquals("Create invoice", state.primaryActionLabel)
    }

    @Test
    fun zoomCanChangeToHourly() {
        val state = reduceTimeboxxingState(
            createInitialTimeboxxingState(),
            TimeboxxingAction.ChangeZoom(60),
        )

        assertEquals(60, state.zoomMinutes)
    }

    @Test
    fun usageLoadReplacesEventsForSelectedDay() {
        val initial = createInitialTimeboxxingState()
        val day = initial.selectedDay
        val sidecarEvent = UsageEvent(
            id = "sidecar-1",
            title = "Client dashboard",
            sourceName = "Google Chrome",
            sourceType = UsageSourceType.Browser,
            startMinute = 9 * 60,
            durationMinutes = 25,
            projectHintId = null,
        )

        val state = reduceTimeboxxingState(
            reduceTimeboxxingState(initial, TimeboxxingAction.LoadUsage),
            TimeboxxingAction.UsageLoadSucceeded(day.startedAtEpochMillis, listOf(sidecarEvent)),
        )

        assertFalse(state.usageLoading)
        assertEquals(listOf(sidecarEvent), state.usageEvents)
        assertTrue(state.selectedUsageIds.isEmpty())
    }

    @Test
    fun staleUsageLoadDoesNotReplaceCurrentDay() {
        val initial = createInitialTimeboxxingState()
        val moved = reduceTimeboxxingState(initial, TimeboxxingAction.MoveDate(1))

        val state = reduceTimeboxxingState(
            moved,
            TimeboxxingAction.UsageLoadSucceeded(initial.selectedDay.startedAtEpochMillis, emptyList()),
        )

        assertEquals(moved, state)
    }

    @Test
    fun streamedUsageEventsAreMergedById() {
        val initial = createInitialTimeboxxingState()
        val day = initial.selectedDay
        val first = UsageEvent(
            id = "sidecar-1",
            title = "Editor",
            sourceName = "VS Code",
            sourceType = UsageSourceType.Application,
            startMinute = 60,
            durationMinutes = 10,
            projectHintId = null,
        )
        val updated = first.copy(durationMinutes = 15)

        val state = listOf(first, updated).fold(initial.copy(usageEvents = emptyList())) { current, event ->
            reduceTimeboxxingState(
                current,
                TimeboxxingAction.MergeUsageEvent(day.startedAtEpochMillis, event),
            )
        }

        assertEquals(listOf(updated), state.usageEvents)
        assertFalse(state.usageLoading)
    }

    @Test
    fun activeUsageEventsReplacePreviousActiveSnapshot() {
        val initial = createInitialTimeboxxingState().copy(usageEvents = emptyList())
        val day = initial.selectedDay
        val first = activeUsageEvent(durationMinutes = 2)
        val updated = activeUsageEvent(durationMinutes = 4)

        val state = listOf(first, updated).fold(initial) { current, event ->
            reduceTimeboxxingState(
                current,
                TimeboxxingAction.MergeUsageEvent(day.startedAtEpochMillis, event),
            )
        }

        assertEquals(listOf(updated), state.usageEvents)
        assertEquals(4, state.capturedMinutes)
    }

    @Test
    fun completedUsageEventRemovesMatchingActiveSnapshot() {
        val initial = createInitialTimeboxxingState().copy(usageEvents = emptyList())
        val day = initial.selectedDay
        val active = activeUsageEvent(durationMinutes = 4)
        val completed = active.copy(
            id = "sidecar-42",
            durationMinutes = 5,
            isActive = false,
        )

        val withActive = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.MergeUsageEvent(day.startedAtEpochMillis, active),
        )
        val state = reduceTimeboxxingState(
            withActive,
            TimeboxxingAction.MergeUsageEvent(day.startedAtEpochMillis, completed),
        )

        assertEquals(listOf(completed), state.usageEvents)
        assertFalse(state.usageEvents.any { it.isActive })
    }

    @Test
    fun activeUsageCannotBeSelected() {
        val initial = createInitialTimeboxxingState().copy(usageEvents = listOf(activeUsageEvent()))

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.ToggleUsageSelection("sidecar-active"))

        assertTrue(state.selectedUsageIds.isEmpty())
        assertTrue(state.selectedUsageEvents.isEmpty())
    }

    private fun createAmaConfiguredState() =
        createInitialTimeboxxingState().copy(
            aiSettings = AiSettings(openRouterSecretExists = true),
        )

    private fun timeEntry(
        startMinute: Int,
        sourceUsageIds: Set<String> = emptySet(),
    ): TimeEntry =
        TimeEntry(
            id = "entry",
            projectId = "project",
            title = "Entry",
            notes = "",
            startMinute = startMinute,
            durationMinutes = 30,
            billable = true,
            sourceUsageIds = sourceUsageIds,
        )

    private fun usageEvent(
        durationMinutes: Int,
        isActive: Boolean = false,
        id: String = if (isActive) "sidecar-active" else "usage",
        startMinute: Int = 10 * 60,
        sourceType: UsageSourceType = UsageSourceType.Application,
    ): UsageEvent =
        UsageEvent(
            id = id,
            title = "Usage",
            sourceName = "App",
            sourceType = sourceType,
            startMinute = startMinute,
            durationMinutes = durationMinutes,
            projectHintId = null,
            isActive = isActive,
        )

    private fun activeUsageEvent(durationMinutes: Int = 1): UsageEvent =
        UsageEvent(
            id = "sidecar-active",
            title = "Client dashboard",
            sourceName = "Google Chrome",
            sourceType = UsageSourceType.Browser,
            startMinute = 10 * 60,
            durationMinutes = durationMinutes,
            projectHintId = null,
            isActive = true,
        )
}
