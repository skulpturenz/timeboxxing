package com.timeboxxing.app

import androidx.compose.ui.graphics.Color
import com.timeboxxing.domain.model.AmaAnswer
import com.timeboxxing.domain.model.AmaAppUsageBucket
import com.timeboxxing.domain.model.AmaAppUsageChart
import com.timeboxxing.domain.model.AmaIndexState
import com.timeboxxing.domain.model.AmaIndexStatus
import com.timeboxxing.domain.model.AmaMessageRole
import com.timeboxxing.domain.model.AmaQueryKind
import com.timeboxxing.domain.model.AmaSource
import com.timeboxxing.domain.model.AmaStructuredQuery
import com.timeboxxing.domain.model.AmaTimeWindow
import com.timeboxxing.data.mock.mockTimeboxxingData
import com.timeboxxing.data.time.usageDayForCalendarDate
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.DatabaseMaintenanceStatus
import com.timeboxxing.domain.model.DatabasePruneCounts
import com.timeboxxing.domain.model.DatabasePruneResult
import com.timeboxxing.domain.model.DatabaseVacuumResult
import com.timeboxxing.domain.model.Project
import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.TimesheetExportFormat
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
import com.timeboxxing.app.ui.DefaultProjectColorArgb
import com.timeboxxing.app.ui.ProjectPaletteColorOptions
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
    fun databasePruneRangeRequiresValidDateRange() {
        val initial = createInitialTimeboxxingState()
        val withStart = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.UpdateDatabasePruneStartDate(CalendarDate(2025, 5, 2)),
        )
        val invalid = reduceTimeboxxingState(
            withStart,
            TimeboxxingAction.UpdateDatabasePruneEndDate(CalendarDate(2025, 5, 1)),
        )

        assertFalse(invalid.canPruneDatabaseRange)

        val valid = reduceTimeboxxingState(
            invalid,
            TimeboxxingAction.UpdateDatabasePruneEndDate(CalendarDate(2025, 5, 2)),
        )

        assertTrue(valid.canPruneDatabaseRange)
    }

    @Test
    fun databasePruneSuccessWithRowsWaitsForVacuumMessage() {
        val pruning = createInitialTimeboxxingState().copy(
            databasePruneStartDate = CalendarDate(2025, 5, 1),
            databasePruneEndDate = CalendarDate(2025, 5, 1),
        ).let { state ->
            reduceTimeboxxingState(state, TimeboxxingAction.PruneDatabaseRange)
        }

        val state = reduceTimeboxxingState(
            pruning,
            TimeboxxingAction.DatabasePruneSucceeded(
                DatabasePruneResult(
                    status = DatabaseMaintenanceStatus(sizeBytes = 1024),
                    counts = DatabasePruneCounts(
                        timesheetEntriesDeleted = 1,
                        usageLinksDeleted = 2,
                    ),
                ),
            ),
        )

        assertFalse(state.databasePruning)
        assertEquals(1024, state.databaseMaintenanceStatus.sizeBytes)
        assertEquals(null, state.databaseMaintenanceMessage)
    }

    @Test
    fun databaseVacuumStateUpdatesStatusAndMessage() {
        val vacuuming = reduceTimeboxxingState(
            createInitialTimeboxxingState(),
            TimeboxxingAction.VacuumDatabase,
        )

        assertTrue(vacuuming.databaseVacuuming)
        assertFalse(vacuuming.canPruneDatabaseRange)

        val state = reduceTimeboxxingState(
            vacuuming,
            TimeboxxingAction.DatabaseVacuumSucceeded(
                result = DatabaseVacuumResult(sizeBeforeBytes = 4096, sizeAfterBytes = 1024),
                prunedCounts = DatabasePruneCounts(timesheetEntriesDeleted = 1, usageLinksDeleted = 2),
            ),
        )

        assertFalse(state.databaseVacuuming)
        assertEquals(1024, state.databaseMaintenanceStatus.sizeBytes)
        assertEquals("Pruned 3 database rows and compacted the database.", state.databaseMaintenanceMessage)
    }

    @Test
    fun selectingUsageBuildsDraftFromUsageHistory() {
        val initial = createInitialTimeboxxingState()

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.ToggleUsageSelection("usage-intro-email"),
        )

        assertEquals(setOf("usage-intro-email"), state.selectedUsageIds)
        assertEquals("", state.draft.projectId)
        assertEquals("Re: Intro Axion Ltd.", state.draft.title)
        assertEquals(20, state.draft.durationMinutes)
        assertTrue(state.draft.notes.contains("Gmail"))
    }

    @Test
    fun selectingUsageIgnoresProjectHintWhenProjectDoesNotExist() {
        val usage = usageEvent(
            id = "hinted",
            durationMinutes = 25,
            projectHintId = "missing-project",
        )
        val initial = createInitialTimeboxxingState().copy(
            usageEvents = listOf(usage),
            draft = createInitialTimeboxxingState().draft.copy(projectId = ""),
        )

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.ToggleUsageSelection("hinted"))

        assertEquals("", state.draft.projectId)
        assertEquals("Usage", state.draft.title)
    }

    @Test
    fun selectingUsageUsesProjectHintWhenProjectExists() {
        val usage = usageEvent(
            id = "hinted",
            durationMinutes = 25,
            projectHintId = "client-project",
        )
        val initial = createInitialTimeboxxingState().copy(
            projects = listOf(project(id = "client-project", name = "Client Project")),
            usageEvents = listOf(usage),
            draft = createInitialTimeboxxingState().draft.copy(projectId = ""),
        )

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.ToggleUsageSelection("hinted"))

        assertEquals("client-project", state.draft.projectId)
        assertEquals("Usage", state.draft.title)
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
    fun submittingStructuredAmaQueriesAddsKindSpecificUserMessages() {
        val window = AmaTimeWindow(
            startedAtEpochMillis = 1_000L,
            endedAtEpochMillis = 2_000L,
        )
        val baselineWindow = AmaTimeWindow(
            startedAtEpochMillis = 0L,
            endedAtEpochMillis = 1_000L,
        )
        val cases = listOf(
            AmaStructuredQuery(
                kind = AmaQueryKind.AppTotals,
                window = window,
                limit = 5,
                periodLabel = "Today",
            ) to "Show app totals for Today",
            AmaStructuredQuery(
                kind = AmaQueryKind.Timeline,
                window = window,
                limit = 20,
                periodLabel = "Today",
            ) to "Show usage timeline for Today",
            AmaStructuredQuery(
                kind = AmaQueryKind.Habits,
                window = window,
                limit = 5,
                periodLabel = "Today",
            ) to "Summarize habits for Today",
            AmaStructuredQuery(
                kind = AmaQueryKind.ComparePeriods,
                window = window,
                baselineWindow = baselineWindow,
                limit = 5,
                periodLabel = "Today",
                baselinePeriodLabel = "Yesterday",
            ) to "Compare Today with Yesterday",
        )

        cases.forEach { (query, expectedMessage) ->
            val state = reduceTimeboxxingState(
                createAmaConfiguredState(),
                TimeboxxingAction.SubmitAmaStructuredQuery(query),
            )

            assertEquals(TimeboxxingSection.Ama, state.selectedSection)
            assertEquals("", state.amaInput)
            assertTrue(state.amaLoading)
            assertEquals(1, state.amaMessages.size)
            assertEquals(AmaMessageRole.User, state.amaMessages.first().role)
            assertEquals(expectedMessage, state.amaMessages.first().content)
        }
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
    fun projectCreatePaletteHasDefaultAndFixedOptionsWithoutProjects() {
        val initial = createInitialTimeboxxingState()

        assertTrue(initial.projects.isEmpty())
        assertEquals(0xFF00FFEEL, DefaultProjectColorArgb)
        assertTrue(ProjectPaletteColorOptions.isNotEmpty())
        assertFalse(DefaultProjectColorArgb in ProjectPaletteColorOptions)
        assertEquals(ProjectPaletteColorOptions.distinct(), ProjectPaletteColorOptions)
    }

    @Test
    fun loadingProjectsReplacesProjectsAndSelectsFallbackDraftProject() {
        val initial = createInitialTimeboxxingState().copy(
            projects = emptyList(),
            draft = createInitialTimeboxxingState().draft.copy(projectId = "missing"),
        )
        val loadedProject = project(id = "lunar-lab", name = "Lunar Lab")

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.ProjectsLoadSucceeded(listOf(loadedProject)),
        )

        assertEquals(listOf(loadedProject), state.projects)
        assertEquals("", state.draft.projectId)
        assertEquals(null, state.notice)
    }

    @Test
    fun loadingNoProjectsLeavesEmptyDraftProject() {
        val initial = stateWithLegacyProjects()

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.ProjectsLoadSucceeded(emptyList()))

        assertTrue(state.projects.isEmpty())
        assertEquals("", state.draft.projectId)
    }

    @Test
    fun projectLoadFailureShowsNoticeWithoutReplacingProjects() {
        val initial = stateWithLegacyProjects()

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.ProjectsLoadFailed("Projects failed."))

        assertEquals(initial.projects, state.projects)
        assertEquals("Projects failed.", state.notice)
    }

    @Test
    fun createProjectIntentDoesNotMutateUntilBackendSuccess() {
        val initial = createInitialTimeboxxingState()

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.CreateProject("  Lunar Lab  ", 0xFF00FFEE),
        )

        assertEquals(initial, state)
    }

    @Test
    fun createProjectSuccessAddsProjectSelectsDraftAndUsesHiddenDefaults() {
        val initial = createInitialTimeboxxingState()
        val createdProject = Project(
            id = "lunar-lab",
            name = "Lunar Lab",
            client = "",
            colorArgb = 0xFF00FFEE,
            hourlyRateCents = 0,
        )

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.CreateProjectSucceeded(createdProject),
        )
        val project = state.projects.last()

        assertEquals(initial.projects.size + 1, state.projects.size)
        assertEquals("lunar-lab", project.id)
        assertEquals("Lunar Lab", project.name)
        assertEquals("", project.client)
        assertEquals(0xFF00FFEE, project.colorArgb)
        assertEquals(0, project.hourlyRateCents)
        assertEquals(project.id, state.draft.projectId)
        assertEquals(0, state.minutesForProject(project.id))
        assertEquals("Created Lunar Lab.", state.notice)
    }

    @Test
    fun createProjectFailureLeavesStateUnchangedExceptNotice() {
        val initial = createInitialTimeboxxingState()

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.CreateProjectFailed("Project failed."),
        )

        assertEquals(initial.copy(notice = "Project failed."), state)
    }

    @Test
    fun deleteProjectFailureLeavesStateUnchangedExceptNotice() {
        val initial = stateWithLegacyProjects()

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.DeleteProjectFailed("Delete failed."),
        )

        assertEquals(initial.copy(notice = "Delete failed."), state)
    }

    @Test
    fun creatingProjectPreservesSelectedUsageAndAssignsEntryToNewProject() {
        val selected = reduceTimeboxxingState(
            createInitialTimeboxxingState(),
            TimeboxxingAction.ToggleUsageSelection("usage-daven-chat"),
        )

        val withProject = reduceTimeboxxingState(
            selected,
            TimeboxxingAction.CreateProjectSucceeded(project(id = "launch-build", name = "Launch Build")),
        )
        val saving = reduceTimeboxxingState(withProject, TimeboxxingAction.AddDraftEntry)
        val state = reduceTimeboxxingState(
            saving,
            TimeboxxingAction.AddDraftEntrySucceeded(
                saving.selectedDay.startedAtEpochMillis,
                TimeEntry(
                    id = "entry-created",
                    projectId = "launch-build",
                    title = saving.draft.title,
                    notes = saving.draft.notes,
                    startMinute = saving.draft.startMinute,
                    durationMinutes = saving.draft.durationMinutes,
                    billable = saving.draft.billable,
                    sourceUsageIds = saving.selectedUsageIds,
                ),
            ),
        )

        assertEquals(setOf("usage-daven-chat"), withProject.selectedUsageIds)
        assertEquals("John (DM) - Daven Ltd.", withProject.draft.title)
        assertEquals(selected.draft.notes, withProject.draft.notes)
        assertEquals(selected.draft.startMinute, withProject.draft.startMinute)
        assertEquals(selected.draft.durationMinutes, withProject.draft.durationMinutes)
        assertEquals(selected.draft.billable, withProject.draft.billable)
        assertEquals("launch-build", state.entries.last().projectId)
        assertEquals(setOf("usage-daven-chat"), state.entries.last().sourceUsageIds)
        assertEquals(15, state.minutesForProject("launch-build"))
    }

    @Test
    fun initialStateCanBeCreatedWithoutProjects() {
        val state = createInitialTimeboxxingState(
            data = mockTimeboxxingData().copy(projects = emptyList()),
        )

        assertTrue(state.projects.isEmpty())
        assertEquals("", state.draft.projectId)
    }

    @Test
    fun deleteProjectIntentDoesNotMutateUntilBackendSuccess() {
        val initial = stateWithLegacyProjects()

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.DeleteProject("daven"))

        assertEquals(initial, state)
    }

    @Test
    fun deletingProjectRemovesProject() {
        val initial = stateWithLegacyProjects()

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.DeleteProjectSucceeded("daven"))

        assertEquals(initial.projects.size - 1, state.projects.size)
        assertFalse(state.projects.any { it.id == "daven" })
        assertEquals("Deleted project.", state.notice)
    }

    @Test
    fun deletingFinalProjectLeavesNoProjectsAndEmptyDraftProject() {
        val initial = stateWithLegacyProjects()
        val onlyProject = initial.projects.first()
        val singleProjectState = initial.copy(
            projects = listOf(onlyProject),
            entries = emptyList(),
            draft = initial.draft.copy(projectId = onlyProject.id),
        )

        val state = reduceTimeboxxingState(singleProjectState, TimeboxxingAction.DeleteProjectSucceeded(onlyProject.id))

        assertTrue(state.projects.isEmpty())
        assertEquals("", state.draft.projectId)
        assertEquals("Deleted project.", state.notice)
    }

    @Test
    fun deletingDraftProjectSelectsFallbackProject() {
        val initial = stateWithLegacyProjects()

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.DeleteProjectSucceeded(initial.draft.projectId))

        assertFalse(state.projects.any { it.id == "morgan" })
        assertEquals("", state.draft.projectId)
    }

    @Test
    fun deletingProjectDoesNotDeleteEntriesAndUnlinksThem() {
        val initial = stateWithLegacyProjects().copy(
            entries = listOf(timeEntry(projectId = "morgan", startMinute = 12 * 60)),
        )

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.DeleteProjectSucceeded("morgan"))

        assertEquals(initial.entries.size, state.entries.size)
        assertTrue(state.entries.any { it.projectId == "" })
    }

    @Test
    fun deletingUnknownProjectIsNoOp() {
        val initial = createInitialTimeboxxingState()

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.DeleteProjectSucceeded("missing"))

        assertEquals(initial, state)
    }

    @Test
    fun missingProjectReferencesUseDeletedProjectPlaceholderAndZeroRate() {
        val initial = stateWithLegacyProjects().copy(
            entries = listOf(timeEntry(projectId = "morgan", startMinute = 12 * 60)),
        )
        val state = reduceTimeboxxingState(initial, TimeboxxingAction.DeleteProjectSucceeded("morgan"))
        val morganInvoiceTotalCents = 18_000 * 30 / 60

        assertEquals("No project", state.projectFor("").name)
        assertEquals("Deleted project", state.projectFor("missing").name)
        assertEquals(0, state.projectFor("").hourlyRateCents)
        assertEquals(initial.invoiceTotalCents() - morganInvoiceTotalCents, state.invoiceTotalCents())
    }

    @Test
    fun addingDraftWithoutProjectsStartsSavingNoProjectEntry() {
        val initial = createInitialTimeboxxingState().copy(
            projects = emptyList(),
            entries = emptyList(),
            draft = createInitialTimeboxxingState().draft.copy(projectId = ""),
        )

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.AddDraftEntry)

        assertTrue(state.entries.isEmpty())
        assertTrue(state.entrySaving)
        assertEquals(null, state.notice)
    }

    @Test
    fun addingDraftCreatesEntryAndClearsSelection() {
        val withProject = reduceTimeboxxingState(
            createInitialTimeboxxingState(),
            TimeboxxingAction.CreateProjectSucceeded(project(id = "daven", name = "Daven")),
        )
        val selected = reduceTimeboxxingState(
            withProject,
            TimeboxxingAction.ToggleUsageSelection("usage-daven-chat"),
        )
        val beforeCount = selected.entries.size

        val saving = reduceTimeboxxingState(selected, TimeboxxingAction.AddDraftEntry)
        val state = reduceTimeboxxingState(
            saving,
            TimeboxxingAction.AddDraftEntrySucceeded(
                saving.selectedDay.startedAtEpochMillis,
                TimeEntry(
                    id = "entry-created",
                    projectId = "daven",
                    title = saving.draft.title,
                    notes = saving.draft.notes,
                    startMinute = saving.draft.startMinute,
                    durationMinutes = saving.draft.durationMinutes,
                    billable = saving.draft.billable,
                    sourceUsageIds = saving.selectedUsageIds,
                ),
            ),
        )

        assertEquals(beforeCount + 1, state.entries.size)
        assertFalse(state.entrySaving)
        assertTrue(state.selectedUsageIds.isEmpty())
        assertEquals("daven", state.entries.last().projectId)
        assertEquals(setOf("usage-daven-chat"), state.entries.last().sourceUsageIds)
        assertEquals(state.entries.last().id, state.scheduleFocusEntryId)
    }

    @Test
    fun duplicatingEntryFocusesDuplicateOnSchedule() {
        val initial = stateWithLegacyProjects().copy(
            entries = listOf(timeEntry(projectId = "morgan", startMinute = 12 * 60)),
        )
        val source = initial.entries.first()

        val saving = reduceTimeboxxingState(initial, TimeboxxingAction.DuplicateEntry(source.id))
        val state = reduceTimeboxxingState(
            saving,
            TimeboxxingAction.DuplicateEntrySucceeded(
                saving.selectedDay.startedAtEpochMillis,
                source.title,
                source.copy(
                    id = "entry-copy",
                    title = "Copy of ${source.title}",
                    startMinute = source.startMinute + source.durationMinutes,
                    sourceUsageIds = emptySet(),
                ),
            ),
        )

        assertEquals(initial.entries.size + 1, state.entries.size)
        assertFalse(state.entrySaving)
        assertEquals(state.entries.last().id, state.scheduleFocusEntryId)
        assertEquals(source.startMinute + source.durationMinutes, state.entries.last().startMinute)
    }

    @Test
    fun deletingFocusedEntryClearsScheduleFocus() {
        val entry = timeEntry(projectId = "morgan", startMinute = 12 * 60)
        val added = stateWithLegacyProjects().copy(
            entries = listOf(entry),
            scheduleFocusEntryId = entry.id,
        )

        val state = reduceTimeboxxingState(
            added,
            TimeboxxingAction.DeleteEntry(added.scheduleFocusEntryId.orEmpty()),
        )

        assertEquals(null, state.scheduleFocusEntryId)
    }

    @Test
    fun movingDateClearsScheduleFocus() {
        val entry = timeEntry(projectId = "morgan", startMinute = 12 * 60)
        val added = stateWithLegacyProjects().copy(
            entries = listOf(entry),
            scheduleFocusEntryId = entry.id,
        )

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
            reduceTimeboxxingState(stateWithLegacyProjects(), TimeboxxingAction.AddDraftEntry),
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
    fun openingAmaUsageSourceFocusesLoadedScheduleEvent() {
        val usage = usageEvent(
            id = "sidecar-42",
            startMinute = 10 * 60,
            durationMinutes = 15,
        )
        val initial = createAmaConfiguredState().copy(usageEvents = listOf(usage))
        val selectedDate = initial.selectedCalendarDate ?: error("Expected a selected date")
        val selectedDay = usageDayForCalendarDate(selectedDate)
        val startedAt = selectedDay.startedAtEpochMillis + usage.startMinute * 60_000L

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.OpenAmaUsageSource(
                startedAtEpochMillis = startedAt,
                transitionEventId = 42,
            ),
        )

        assertEquals(TimeboxxingSection.Overview, state.selectedSection)
        assertEquals(setOf("sidecar-42"), state.selectedUsageIds)
        assertEquals(usage.startMinute, state.scheduleFocusTarget?.minute)
        assertEquals("sidecar-42", state.scheduleFocusTarget?.usageId)
    }

    @Test
    fun openingAmaUsageSourcePreservesFocusUntilTargetDayLoads() {
        val targetDate = CalendarDate(2026, 7, 4)
        val targetDay = usageDayForCalendarDate(targetDate)
        val startedAt = targetDay.startedAtEpochMillis + 14 * 60 * 60_000L
        val initial = createAmaConfiguredState()

        val focusing = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.OpenAmaUsageSource(
                startedAtEpochMillis = startedAt,
                transitionEventId = 9,
            ),
        )

        assertEquals(targetDate, focusing.selectedCalendarDate)
        assertTrue(focusing.usageLoading)
        assertEquals("sidecar-9", focusing.scheduleFocusTarget?.usageId)
        assertTrue(focusing.selectedUsageIds.isEmpty())

        val loaded = reduceTimeboxxingState(
            focusing,
            TimeboxxingAction.UsageLoadSucceeded(
                dayStartedAtEpochMillis = targetDay.startedAtEpochMillis,
                events = listOf(usageEvent(id = "sidecar-9", startMinute = 14 * 60, durationMinutes = 20)),
            ),
        )

        assertEquals(setOf("sidecar-9"), loaded.selectedUsageIds)
        assertEquals(14 * 60, loaded.scheduleFocusTarget?.minute)
    }

    @Test
    fun openingTimestampOnlyAmaSourceScrollsWithoutSelectingUsage() {
        val initial = createAmaConfiguredState()
        val selectedDate = initial.selectedCalendarDate ?: error("Expected a selected date")
        val selectedDay = usageDayForCalendarDate(selectedDate)
        val startedAt = selectedDay.startedAtEpochMillis + 9 * 60 * 60_000L

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.OpenAmaUsageSource(
                startedAtEpochMillis = startedAt,
                transitionEventId = 0,
            ),
        )

        assertEquals(TimeboxxingSection.Overview, state.selectedSection)
        assertEquals(9 * 60, state.scheduleFocusTarget?.minute)
        assertEquals(null, state.scheduleFocusTarget?.usageId)
        assertTrue(state.selectedUsageIds.isEmpty())
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
        val initial = stateWithLegacyProjects().copy(
            entries = listOf(timeEntry(projectId = "morgan", startMinute = 12 * 60)),
        )
        val entry = initial.entries.first()

        val deleting = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.DeleteEntry(entry.id),
        )
        val state = reduceTimeboxxingState(deleting, TimeboxxingAction.DeleteEntrySucceeded(entry.id))

        assertFalse(state.entries.any { it.id == entry.id })
        assertEquals(initial.billableMinutes - entry.durationMinutes, state.billableMinutes)
    }

    @Test
    fun exportTimesheetStartsWhenEntriesExist() {
        val initial = createInitialTimeboxxingState().copy(
            entries = listOf(timeEntry(projectId = "morgan", startMinute = 12 * 60)),
        )
        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.ExportTimesheet(TimesheetExportFormat.Json),
        )

        assertTrue(state.timesheetExporting)
        assertEquals(null, state.notice)
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
    fun usageLoadLinearizesOverlappingSidecarEventsByReportOrder() {
        val initial = createInitialTimeboxxingState()
        val day = initial.selectedDay
        val older = usageEvent(
            id = "sidecar-10",
            startMinute = 10 * 60,
            durationMinutes = 10,
        )
        val newer = usageEvent(
            id = "sidecar-11",
            startMinute = 10 * 60 + 3,
            durationMinutes = 2,
        )

        val loaded = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.UsageLoadSucceeded(day.startedAtEpochMillis, listOf(newer, older)),
        )
        val selected = reduceTimeboxxingState(loaded, TimeboxxingAction.ToggleUsageSelection("sidecar-10"))

        assertEquals(listOf("sidecar-10", "sidecar-11"), loaded.usageEvents.map { it.id })
        assertEquals(10 * 60, loaded.usageEvents[0].startMinute)
        assertEquals(3, loaded.usageEvents[0].durationMinutes)
        assertEquals(10 * 60 + 3, loaded.usageEvents[1].startMinute)
        assertEquals(2, loaded.usageEvents[1].durationMinutes)
        assertEquals(5, loaded.capturedMinutes)
        assertEquals(1, loaded.usageEvents.count { it.id == "sidecar-10" })
        assertEquals(3, selected.selectedUsageMinutes)
        assertEquals(3, selected.draft.durationMinutes)
    }

    @Test
    fun usageLoadUsesInputOrderForNonSidecarOverlapResolution() {
        val initial = createInitialTimeboxxingState()
        val day = initial.selectedDay
        val first = usageEvent(
            id = "first",
            startMinute = 11 * 60,
            durationMinutes = 8,
        )
        val second = usageEvent(
            id = "second",
            startMinute = 11 * 60 + 5,
            durationMinutes = 4,
        )

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.UsageLoadSucceeded(day.startedAtEpochMillis, listOf(first, second)),
        )

        assertEquals(listOf("first", "second"), state.usageEvents.map { it.id })
        assertEquals(5, state.usageEvents.first { it.id == "first" }.durationMinutes)
        assertEquals(11 * 60 + 5, state.usageEvents.first { it.id == "second" }.startMinute)
        assertEquals(4, state.usageEvents.first { it.id == "second" }.durationMinutes)
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
        assertEquals(0, state.capturedMinutes)
        assertEquals(0, state.unassignedUsageMinutes)
    }

    @Test
    fun usageLoadPrefersActiveSnapshotOverMatchingCompletedUsage() {
        val initial = createInitialTimeboxxingState()
        val day = initial.selectedDay
        val active = activeUsageEvent(durationMinutes = 4)
        val matchingCompleted = active.copy(
            id = "sidecar-41",
            durationMinutes = 1,
            isActive = false,
        )

        val state = reduceTimeboxxingState(
            initial,
            TimeboxxingAction.UsageLoadSucceeded(day.startedAtEpochMillis, listOf(matchingCompleted, active)),
        )

        assertEquals(listOf(active), state.usageEvents)
        assertEquals(0, state.capturedMinutes)
        assertEquals(0, state.unassignedUsageMinutes)
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
        assertEquals(5, state.capturedMinutes)
        assertEquals(5, state.unassignedUsageMinutes)
    }

    @Test
    fun activeUsageCannotBeSelected() {
        val initial = createInitialTimeboxxingState().copy(usageEvents = listOf(activeUsageEvent()))

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.ToggleUsageSelection("sidecar-active"))

        assertTrue(state.selectedUsageIds.isEmpty())
        assertTrue(state.selectedUsageEvents.isEmpty())
    }

    @Test
    fun oneMinuteUsageCanBeSelected() {
        val oneMinute = usageEvent(
            id = "one-minute",
            durationMinutes = 1,
        )
        val initial = createInitialTimeboxxingState().copy(usageEvents = listOf(oneMinute))

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.ToggleUsageSelection("one-minute"))

        assertEquals(setOf("one-minute"), state.selectedUsageIds)
        assertEquals(listOf(oneMinute), state.selectedUsageEvents)
        assertEquals(1, state.selectedUsageMinutes)
        assertEquals(1, state.draft.durationMinutes)
    }

    @Test
    fun subMinuteUsageCannotBeSelected() {
        val subMinute = usageEvent(
            id = "sub-minute",
            durationMinutes = 0,
        )
        val initial = createInitialTimeboxxingState().copy(usageEvents = listOf(subMinute))

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.ToggleUsageSelection("sub-minute"))

        assertTrue(state.selectedUsageIds.isEmpty())
        assertTrue(state.selectedUsageEvents.isEmpty())
    }

    @Test
    fun selectedUsageIdsArePrunedWhenMergedUsageBecomesSubMinute() {
        val usage = usageEvent(
            id = "usage",
            durationMinutes = 2,
        )
        val initial = createInitialTimeboxxingState().copy(usageEvents = listOf(usage))
        val selected = reduceTimeboxxingState(initial, TimeboxxingAction.ToggleUsageSelection("usage"))
        val subMinuteUpdate = usage.copy(durationMinutes = 0)

        val state = reduceTimeboxxingState(
            selected,
            TimeboxxingAction.MergeUsageEvent(selected.selectedDay.startedAtEpochMillis, subMinuteUpdate),
        )

        assertTrue(state.selectedUsageIds.isEmpty())
        assertTrue(state.selectedUsageEvents.isEmpty())
        assertTrue(state.usageEvents.none { it.id == "usage" })
    }

    @Test
    fun staleSubMinuteSelectionIsIgnoredBySelectedUsageEvents() {
        val subMinute = usageEvent(
            id = "sub-minute",
            durationMinutes = 0,
        )
        val oneMinute = usageEvent(
            id = "one-minute",
            durationMinutes = 1,
            startMinute = subMinute.startMinute + 1,
        )
        val state = createInitialTimeboxxingState().copy(
            usageEvents = listOf(subMinute, oneMinute),
            selectedUsageIds = setOf("sub-minute", "one-minute"),
        )

        assertEquals(listOf("one-minute"), state.selectedUsageEvents.map { it.id })
        assertEquals(1, state.selectedUsageMinutes)
    }

    @Test
    fun idleUsageCannotBeSelected() {
        val idle = usageEvent(
            id = "idle",
            durationMinutes = 20,
            sourceType = UsageSourceType.Idle,
        )
        val initial = createInitialTimeboxxingState().copy(usageEvents = listOf(idle))

        val state = reduceTimeboxxingState(initial, TimeboxxingAction.ToggleUsageSelection("idle"))

        assertTrue(state.selectedUsageIds.isEmpty())
        assertTrue(state.selectedUsageEvents.isEmpty())
    }

    @Test
    fun idleUsageDoesNotAffectCapturedOrUnassignedMetrics() {
        val work = usageEvent(
            id = "work",
            durationMinutes = 30,
        )
        val idle = usageEvent(
            id = "idle",
            durationMinutes = 20,
            sourceType = UsageSourceType.Idle,
        )
        val initial = createInitialTimeboxxingState().copy(
            usageEvents = listOf(work, idle),
            entries = listOf(timeEntry(startMinute = work.startMinute, sourceUsageIds = setOf(work.id))),
        )

        assertEquals(30, initial.capturedMinutes)
        assertEquals(0, initial.unassignedUsageMinutes)
    }

    private fun createAmaConfiguredState() =
        createInitialTimeboxxingState().copy(
            aiSettings = AiSettings(openRouterSecretExists = true),
        )

    private fun stateWithLegacyProjects() =
        createInitialTimeboxxingState().let { state ->
            state.copy(
                projects = listOf(
                    project(id = "morgan", name = "Morgan Project", hourlyRateCents = 18_000),
                    project(id = "axion", name = "Axion Ltd. Project", hourlyRateCents = 16_500),
                    project(id = "daven", name = "Daven Retainer", hourlyRateCents = 15_000),
                ),
                draft = state.draft.copy(projectId = "morgan"),
            )
        }

    private fun project(
        id: String,
        name: String,
        hourlyRateCents: Int = 0,
    ): Project =
        Project(
            id = id,
            name = name,
            client = "",
            colorArgb = 0xFF00FFEE,
            hourlyRateCents = hourlyRateCents,
        )

    private fun timeEntry(
        startMinute: Int,
        projectId: String = "project",
        sourceUsageIds: Set<String> = emptySet(),
    ): TimeEntry =
        TimeEntry(
            id = "entry",
            projectId = projectId,
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
        projectHintId: String? = null,
    ): UsageEvent =
        UsageEvent(
            id = id,
            title = "Usage",
            sourceName = "App",
            sourceType = sourceType,
            startMinute = startMinute,
            durationMinutes = durationMinutes,
            projectHintId = projectHintId,
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
