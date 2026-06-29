package com.timeboxxing.app

import androidx.compose.ui.graphics.Color
import com.timeboxxing.app.model.AmaAnswer
import com.timeboxxing.app.model.AmaAppUsageBucket
import com.timeboxxing.app.model.AmaAppUsageChart
import com.timeboxxing.app.model.AmaIndexState
import com.timeboxxing.app.model.AmaIndexStatus
import com.timeboxxing.app.model.AmaMessageRole
import com.timeboxxing.app.model.AmaSource
import com.timeboxxing.app.data.mockTimeboxxingData
import com.timeboxxing.app.model.AiSettings
import com.timeboxxing.app.model.AppearanceMode
import com.timeboxxing.app.model.CalendarDate
import com.timeboxxing.app.model.EntryMode
import com.timeboxxing.app.model.TimeEntry
import com.timeboxxing.app.model.UsageEvent
import com.timeboxxing.app.model.UsageSourceType
import com.timeboxxing.app.model.formatClockTime
import com.timeboxxing.app.model.formatDuration
import com.timeboxxing.app.state.TimeboxxingAction
import com.timeboxxing.app.state.TimeboxxingSection
import com.timeboxxing.app.state.createInitialTimeboxxingState
import com.timeboxxing.app.state.reduceTimeboxxingState
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

class TimeboxxingStateTest {

    @Test
    fun lightAndDarkPalettesUseDistinctSurfaces() {
        assertEquals(Color.White, TbLightColors.appBackground)
        assertEquals(Color.Black, TbDarkColors.appBackground)
        assertEquals(Color(0xFF00FFEE), TbLightColors.accent)
        assertEquals(Color(0xFF00FFEE), TbDarkColors.accent)
        assertTrue(TbLightColors.appBackground != TbDarkColors.appBackground)
        assertTrue(TbLightColors.surface != TbDarkColors.surface)
        assertTrue(TbLightColors.elevatedSurface != TbDarkColors.elevatedSurface)
        assertTrue(TbLightColors.text != TbDarkColors.text)
    }

    @Test
    fun darkPaletteTextContrastsAgainstSurfaces() {
        assertContrastAtLeast(7.0, TbDarkColors.text, TbDarkColors.surface)
        assertContrastAtLeast(7.0, TbDarkColors.text, TbDarkColors.groupedSurface)
        assertContrastAtLeast(7.0, TbDarkColors.text, TbDarkColors.elevatedSurface)
        assertContrastAtLeast(4.5, TbDarkColors.secondaryText, TbDarkColors.surface)
    }

    @Test
    fun accentTextContrastsAgainstAccentBackgrounds() {
        assertContrastAtLeast(4.5, TbLightColors.accentText, TbLightColors.accent)
        assertContrastAtLeast(4.5, TbDarkColors.accentText, TbDarkColors.accent)
    }

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

    @Test
    fun formatsDurationAndClockTimes() {
        assertEquals("45m", formatDuration(45))
        assertEquals("2h 5m", formatDuration(125))
        assertEquals("12:00 PM", formatClockTime(12 * 60))
        assertEquals("2:05 PM", formatClockTime(14 * 60 + 5))
    }

    @Test
    fun scheduleInitialAutoScrollIsOneShotForSameKey() {
        val scrollState = SchedulePaneAutoScrollState()

        assertTrue(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 15, targetDp = 42f))
        assertFalse(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 15, targetDp = 42f))
    }

    @Test
    fun scheduleInitialAutoScrollRearmsForDifferentDay() {
        val scrollState = SchedulePaneAutoScrollState()

        assertTrue(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 15, targetDp = 42f))
        assertTrue(scrollState.shouldHandleInitialScroll(2L, zoomMinutes = 15, targetDp = 42f))
        assertFalse(scrollState.shouldHandleInitialScroll(2L, zoomMinutes = 15, targetDp = 42f))
    }

    @Test
    fun scheduleInitialAutoScrollRearmsForZoomOrTargetChange() {
        val scrollState = SchedulePaneAutoScrollState()

        assertTrue(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 15, targetDp = 42f))
        assertTrue(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 30, targetDp = 42f))
        assertTrue(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 30, targetDp = 84f))
        assertFalse(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 30, targetDp = 84f))
    }

    @Test
    fun scheduleFocusedEntryAutoScrollIsOneShotForSameKey() {
        val scrollState = SchedulePaneAutoScrollState()

        assertTrue(
            scrollState.shouldHandleFocusedEntryScroll(
                entryId = "entry-1",
                startMinute = 10 * 60,
                sourceUsageIds = setOf("usage-1"),
                zoomMinutes = 15,
                targetDp = 120f,
            ),
        )
        assertFalse(
            scrollState.shouldHandleFocusedEntryScroll(
                entryId = "entry-1",
                startMinute = 10 * 60,
                sourceUsageIds = setOf("usage-1"),
                zoomMinutes = 15,
                targetDp = 120f,
            ),
        )
    }

    @Test
    fun scheduleFocusedEntryAutoScrollRearmsForDifferentEntry() {
        val scrollState = SchedulePaneAutoScrollState()

        assertTrue(
            scrollState.shouldHandleFocusedEntryScroll(
                entryId = "entry-1",
                startMinute = 10 * 60,
                sourceUsageIds = setOf("usage-1"),
                zoomMinutes = 15,
                targetDp = 120f,
            ),
        )
        assertTrue(
            scrollState.shouldHandleFocusedEntryScroll(
                entryId = "entry-2",
                startMinute = 11 * 60,
                sourceUsageIds = setOf("usage-2"),
                zoomMinutes = 15,
                targetDp = 240f,
            ),
        )
        assertFalse(
            scrollState.shouldHandleFocusedEntryScroll(
                entryId = "entry-2",
                startMinute = 11 * 60,
                sourceUsageIds = setOf("usage-2"),
                zoomMinutes = 15,
                targetDp = 240f,
            ),
        )
    }

    @Test
    fun timelineGridUsesFullDayRowsForSupportedZoomLevels() {
        val expected = listOf(
            10 to 280f,
            15 to 420f,
            30 to 840f,
            60 to 1680f,
        )

        expected.forEach { (zoomMinutes, rowHeightDp) ->
            val grid = buildTimelineGrid(
                events = emptyList(),
                zoomMinutes = zoomMinutes,
            )

            assertEquals(0, grid.visibleStartMinute)
            assertEquals(24 * 60, grid.visibleEndMinute)
            assertEquals((24 * 60) / zoomMinutes, grid.intervalCount)
            assertWithin(rowHeightDp, grid.rowHeightDp)
            assertWithin(rowHeightDp / zoomMinutes, grid.dpPerMinute)
            assertEquals(listOf(0, zoomMinutes, zoomMinutes * 2), grid.rows.take(3).map { it.minute })
            assertEquals(24 * 60, grid.rows.last().minute)
        }
    }

    @Test
    fun timelineIntervalMarkerOffsetsUseFiveMinuteCadenceWithoutBoundaryDuplicates() {
        val rowStartMinute = 8 * 60
        val cases = listOf(
            60 to listOf(5f, 10f, 15f, 20f, 25f, 30f, 35f, 40f, 45f, 50f, 55f),
            30 to listOf(5f, 10f, 15f, 20f, 25f),
            15 to listOf(5f, 10f),
            10 to listOf(5f),
        )

        cases.forEach { (zoomMinutes, expectedOffsets) ->
            assertEquals(
                expectedOffsets,
                timelineIntervalMarkerOffsets(
                    rowStartMinute = rowStartMinute,
                    rowEndMinute = rowStartMinute + zoomMinutes,
                    dpPerMinute = 1f,
                ),
            )
        }
    }

    @Test
    fun timelineOneMinuteCompletedEventUsesExactScaledHeight() {
        val grid = buildTimelineGrid(
            events = listOf(usageEvent(durationMinutes = 1)),
            zoomMinutes = 15,
        )
        val placement = grid.placements.single()

        assertWithin(28f, timelineDpPerMinuteForZoom(15))
        assertWithin(timelineDpPerMinuteForZoom(15), placement.heightDp)
        assertWithin(10 * 60 * timelineDpPerMinuteForZoom(15), placement.timeTopDp)
        assertWithin(placement.timeTopDp, placement.displayTopDp)
    }

    @Test
    fun timelineTwoOneMinuteEventsStayInsideFifteenMinuteInterval() {
        val events = listOf(
            usageEvent(id = "usage-one", startMinute = 11 * 60, durationMinutes = 1),
            usageEvent(id = "usage-two", startMinute = 11 * 60 + 1, durationMinutes = 1),
        )
        val grid = buildTimelineGrid(
            events = events,
            zoomMinutes = 15,
        )
        val placements = grid.placements
        val firstTopDp = placements.first().displayTopDp
        val lastBottomDp = placements.last().displayTopDp + placements.last().heightDp

        assertWithin(15f * timelineDpPerMinuteForZoom(15), grid.rowHeightDp)
        assertWithin(2f * timelineDpPerMinuteForZoom(15), lastBottomDp - firstTopDp)
        assertTrue(lastBottomDp - firstTopDp < grid.rowHeightDp)
    }

    @Test
    fun timelineTenSequentialSixMinuteEventsOccupyOneScaledHour() {
        val events = (0 until 10).map { index ->
            usageEvent(
                id = "usage-$index",
                startMinute = 10 * 60 + index * 6,
                durationMinutes = 6,
            )
        }
        val grid = buildTimelineGrid(
            events = events,
            zoomMinutes = 15,
        )
        val placements = grid.placements
        val firstTopDp = placements.first().displayTopDp
        val lastBottomDp = placements.last().displayTopDp + placements.last().heightDp

        assertWithin(60f * timelineDpPerMinuteForZoom(15), lastBottomDp - firstTopDp)
    }

    @Test
    fun timelineLongerEventScalesWithSelectedZoomDensity() {
        val grid = buildTimelineGrid(
            events = listOf(usageEvent(durationMinutes = 30)),
            zoomMinutes = 60,
        )

        assertWithin(840f, grid.placements.single().heightDp)
    }

    @Test
    fun timelineOneMinuteEventAtHourlyZoomUsesReadableScale() {
        val grid = buildTimelineGrid(
            events = listOf(usageEvent(durationMinutes = 1)),
            zoomMinutes = 60,
        )

        assertWithin(28f, timelineDpPerMinuteForZoom(60))
        assertWithin(28f, grid.placements.single().heightDp)
    }

    @Test
    fun timelineActiveEventUsesExactScaledHeight() {
        val grid = buildTimelineGrid(
            events = listOf(usageEvent(durationMinutes = 1, isActive = true)),
            zoomMinutes = 15,
        )

        assertWithin(timelineDpPerMinuteForZoom(15), grid.placements.single().heightDp)
    }

    @Test
    fun timelineIdleEventUsesSameScaledPlacementAsApplicationUsage() {
        val grid = buildTimelineGrid(
            events = listOf(
                usageEvent(
                    id = "idle",
                    durationMinutes = 5,
                    sourceType = UsageSourceType.Idle,
                ),
            ),
            zoomMinutes = 10,
        )
        val placement = grid.placements.single()

        assertEquals(UsageSourceType.Idle, placement.event.sourceType)
        assertWithin(5f * timelineDpPerMinuteForZoom(10), placement.heightDp)
    }

    @Test
    fun timelineEventCrossingRowsIsSplitIntoRowSegments() {
        val event = usageEvent(
            id = "cross-row",
            startMinute = 10 * 60 + 50,
            durationMinutes = 20,
        )
        val grid = buildTimelineGrid(
            events = listOf(event),
            zoomMinutes = 15,
        )
        val firstRow = grid.rows.first { it.minute == 10 * 60 + 45 }
        val secondRow = grid.rows.first { it.minute == 11 * 60 }
        val firstSegment = timelineSegmentsForRow(grid, firstRow).single()
        val secondSegment = timelineSegmentsForRow(grid, secondRow).single()

        assertWithin(5f * grid.dpPerMinute, firstSegment.offsetDp)
        assertWithin(10f * grid.dpPerMinute, firstSegment.heightDp)
        assertTrue(firstSegment.startsEvent)
        assertFalse(firstSegment.endsEvent)

        assertWithin(0f, secondSegment.offsetDp)
        assertWithin(10f * grid.dpPerMinute, secondSegment.heightDp)
        assertFalse(secondSegment.startsEvent)
        assertTrue(secondSegment.endsEvent)
    }

    @Test
    fun timelineBackToBackEventsDoNotReceiveArtificialGaps() {
        val events = listOf(
            usageEvent(id = "first", startMinute = 11 * 60, durationMinutes = 1),
            usageEvent(id = "second", startMinute = 11 * 60 + 1, durationMinutes = 1),
        )
        val grid = buildTimelineGrid(
            events = events,
            zoomMinutes = 15,
        )
        val first = grid.placements.first { it.event.id == "first" }
        val second = grid.placements.first { it.event.id == "second" }

        assertWithin(first.displayTopDp + first.heightDp, second.displayTopDp)
        assertEquals(1, first.laneCount)
        assertEquals(1, second.laneCount)
    }

    @Test
    fun timelineOverlappingEventsUseHorizontalLanesWithoutMovingTime() {
        val events = listOf(
            usageEvent(id = "first", startMinute = 11 * 60, durationMinutes = 10),
            usageEvent(id = "second", startMinute = 11 * 60 + 5, durationMinutes = 10),
        )
        val grid = buildTimelineGrid(
            events = events,
            zoomMinutes = 15,
        )
        val first = grid.placements.first { it.event.id == "first" }
        val second = grid.placements.first { it.event.id == "second" }

        assertWithin(first.timeTopDp, first.displayTopDp)
        assertWithin(second.timeTopDp, second.displayTopDp)
        assertEquals(0, first.laneIndex)
        assertEquals(1, second.laneIndex)
        assertEquals(2, first.laneCount)
        assertEquals(2, second.laneCount)
    }

    @Test
    fun timelineRowHeightHelperUsesReadableMinuteScale() {
        assertWithin(280f, timelineRowHeightDpForZoom(10))
        assertWithin(420f, timelineRowHeightDpForZoom(15))
        assertWithin(840f, timelineRowHeightDpForZoom(30))
        assertWithin(1680f, timelineRowHeightDpForZoom(60))
        assertWithin(28f, timelineDpPerMinuteForZoom(10))
        assertWithin(28f, timelineDpPerMinuteForZoom(15))
        assertWithin(28f, timelineDpPerMinuteForZoom(30))
        assertWithin(28f, timelineDpPerMinuteForZoom(60))
    }

    @Test
    fun timelineFirstEventScrollUsesFixedVisualContext() {
        val grid = buildTimelineGrid(
            events = mockTimeboxxingData().usageEvents,
            zoomMinutes = 15,
        )
        val firstPlacement = grid.placements.first()

        assertWithin(firstPlacement.displayTopDp - 48f, timelineFirstEventScrollDp(grid)!!)
    }

    @Test
    fun timelineFirstEventScrollKeepsHourlyZoomFirstEventVisible() {
        val grid = buildTimelineGrid(
            events = mockTimeboxxingData().usageEvents,
            zoomMinutes = 60,
        )
        val firstPlacement = grid.placements.first()
        val target = timelineFirstEventScrollDp(grid)!!

        assertWithin(firstPlacement.displayTopDp - 48f, target)
        assertTrue(firstPlacement.displayTopDp - target < 600f)
    }

    @Test
    fun timelineFirstEventScrollClampsEarlyEventsToTop() {
        val grid = buildTimelineGrid(
            events = listOf(usageEvent(startMinute = 1, durationMinutes = 1)),
            zoomMinutes = 60,
        )

        assertWithin(0f, timelineFirstEventScrollDp(grid)!!)
    }

    @Test
    fun timelineFirstEventScrollReturnsNullWithoutEvents() {
        val grid = buildTimelineGrid(
            events = emptyList(),
            zoomMinutes = 15,
        )

        assertEquals(null, timelineFirstEventScrollDp(grid))
    }

    @Test
    fun timelineNowScrollDirectionPointsDownWhenNowIsBelowViewport() {
        assertEquals(
            TimelineScrollDirection.Down,
            timelineNowScrollDirection(
                nowOffsetDp = 680f,
                viewportTopDp = 100f,
                viewportHeightDp = 600f,
            ),
        )
    }

    @Test
    fun timelineNowScrollDirectionPointsUpWhenNowIsAboveViewport() {
        assertEquals(
            TimelineScrollDirection.Up,
            timelineNowScrollDirection(
                nowOffsetDp = 90f,
                viewportTopDp = 100f,
                viewportHeightDp = 600f,
            ),
        )
    }

    @Test
    fun timelineNowScrollDirectionIsNullWhenNowIsVisible() {
        assertEquals(
            null,
            timelineNowScrollDirection(
                nowOffsetDp = 500f,
                viewportTopDp = 100f,
                viewportHeightDp = 600f,
            ),
        )
    }

    @Test
    fun timelineScrollToMinuteCentersTargetAndClampsToDayBounds() {
        val grid = buildTimelineGrid(
            events = emptyList(),
            zoomMinutes = 60,
        )

        assertWithin(12 * 60 * grid.dpPerMinute, timelineMinuteOffsetDp(grid, 12 * 60))
        assertWithin(
            timelineMinuteOffsetDp(grid, 12 * 60) - 300f,
            timelineScrollToMinuteDp(grid, 12 * 60, viewportHeightDp = 600f),
        )
        assertWithin(0f, timelineScrollToMinuteDp(grid, 1, viewportHeightDp = 600f))
        assertWithin(
            (grid.contentHeightDp + 24f - 600f).coerceAtLeast(0f),
            timelineScrollToMinuteDp(grid, 24 * 60, viewportHeightDp = 600f),
        )
    }

    @Test
    fun timelineEntryScrollStartsNearEntryTimeWithContext() {
        val grid = buildTimelineGrid(
            events = emptyList(),
            zoomMinutes = 15,
        )
        val entry = timeEntry(startMinute = 9 * 60)

        assertWithin(
            9 * 60 * grid.dpPerMinute - 2 * grid.rowHeightDp,
            timelineEntryScrollDp(grid, entry, viewportHeightDp = 600f),
        )
    }

    @Test
    fun timelineEntryScrollPrefersAssignedUsagePlacement() {
        val grid = buildTimelineGrid(
            events = mockTimeboxxingData().usageEvents,
            zoomMinutes = 15,
        )
        val sourcePlacement = grid.placements.first { it.event.id == "usage-daven-chat" }
        val entry = timeEntry(
            startMinute = 9 * 60,
            sourceUsageIds = setOf("usage-daven-chat"),
        )

        assertWithin(
            sourcePlacement.displayTopDp - 2 * grid.rowHeightDp,
            timelineEntryScrollDp(grid, entry, viewportHeightDp = 600f),
        )
    }

    @Test
    fun timelineEntryScrollFallsBackToStartTimeForUnknownSourceUsage() {
        val grid = buildTimelineGrid(
            events = mockTimeboxxingData().usageEvents,
            zoomMinutes = 15,
        )
        val entry = timeEntry(
            startMinute = 9 * 60,
            sourceUsageIds = setOf("missing-usage"),
        )

        assertWithin(
            9 * 60 * grid.dpPerMinute - 2 * grid.rowHeightDp,
            timelineEntryScrollDp(grid, entry, viewportHeightDp = 600f),
        )
    }

    @Test
    fun timelineEntryScrollClampsEarlyEntriesToTop() {
        val grid = buildTimelineGrid(
            events = emptyList(),
            zoomMinutes = 15,
        )
        val entry = timeEntry(startMinute = 1)

        assertWithin(0f, timelineEntryScrollDp(grid, entry, viewportHeightDp = 600f))
    }

    @Test
    fun timelineEntryScrollClampsLateEntriesToMaximumScroll() {
        val grid = buildTimelineGrid(
            events = emptyList(),
            zoomMinutes = 15,
        )
        val entry = timeEntry(startMinute = 23 * 60 + 59)

        assertWithin(
            (grid.contentHeightDp + 24f - 600f).coerceAtLeast(0f),
            timelineEntryScrollDp(grid, entry, viewportHeightDp = 600f, contextRows = 0),
        )
    }

    @Test
    fun timelineEntryScrollClampsLateEntriesToProvidedMaximumScroll() {
        val grid = buildTimelineGrid(
            events = emptyList(),
            zoomMinutes = 15,
        )
        val entry = timeEntry(startMinute = 23 * 60 + 50)

        assertWithin(500f, timelineEntryScrollDp(grid, entry, viewportHeightDp = 600f, maxScrollDp = 500f))
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

    private fun assertWithin(
        expected: Float,
        actual: Float,
    ) {
        assertTrue(abs(expected - actual) < 0.001f, "Expected $actual to be within 0.001 of $expected")
    }

    private fun assertContrastAtLeast(
        minimum: Double,
        foreground: Color,
        background: Color,
    ) {
        val actual = contrastRatio(foreground, background)
        assertTrue(
            actual >= minimum,
            "Expected contrast ratio $actual to be at least $minimum",
        )
    }

    private fun contrastRatio(
        foreground: Color,
        background: Color,
    ): Double {
        val foregroundLuminance = relativeLuminance(foreground)
        val backgroundLuminance = relativeLuminance(background)
        return (max(foregroundLuminance, backgroundLuminance) + 0.05) /
            (min(foregroundLuminance, backgroundLuminance) + 0.05)
    }

    private fun relativeLuminance(color: Color): Double =
        0.2126 * linearizedColorComponent(color.red) +
            0.7152 * linearizedColorComponent(color.green) +
            0.0722 * linearizedColorComponent(color.blue)

    private fun linearizedColorComponent(component: Float): Double {
        val value = component.toDouble()
        return if (value <= 0.03928) {
            value / 12.92
        } else {
            ((value + 0.055) / 1.055).pow(2.4)
        }
    }
}
