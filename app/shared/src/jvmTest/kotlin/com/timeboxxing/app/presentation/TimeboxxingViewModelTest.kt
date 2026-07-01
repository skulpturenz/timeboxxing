package com.timeboxxing.app.presentation

import com.timeboxxing.data.mock.mockTimeboxxingData
import com.timeboxxing.domain.model.AmaAnswer
import com.timeboxxing.domain.model.AmaIndexState
import com.timeboxxing.domain.model.AmaIndexStatus
import com.timeboxxing.domain.model.AiModelOptions
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.DatabaseMaintenanceStatus
import com.timeboxxing.domain.model.DatabasePruneCounts
import com.timeboxxing.domain.model.DatabasePruneRange
import com.timeboxxing.domain.model.DatabasePruneResult
import com.timeboxxing.domain.model.DatabaseVacuumResult
import com.timeboxxing.domain.model.DiagnosticsLogLine
import com.timeboxxing.domain.model.Project
import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.TimesheetEntryDraft
import com.timeboxxing.domain.model.TimesheetExport
import com.timeboxxing.domain.model.TimesheetExportFormat
import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.model.UsageEvent
import com.timeboxxing.domain.repository.AmaRepository
import com.timeboxxing.domain.repository.ProjectRepository
import com.timeboxxing.domain.repository.SettingsRepository
import com.timeboxxing.domain.repository.TimesheetRepository
import com.timeboxxing.domain.repository.UsageHistoryRepository
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse

@OptIn(ExperimentalCoroutinesApi::class)
class TimeboxxingViewModelTest {
    private val dispatcher = StandardTestDispatcher()

    @BeforeTest
    fun setUp() {
        Dispatchers.setMain(dispatcher)
    }

    @AfterTest
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun readyRuntimeLoadsUsageForInitialDay() = runTest {
        val usageRepository = FakeUsageHistoryRepository(
            loadedEvents = listOf(mockTimeboxxingData().usageEvents.first()),
        )
        val viewModel = TimeboxxingViewModel(fakeRuntime(usageRepository = usageRepository))

        advanceUntilIdle()

        assertEquals(1, usageRepository.getCalls)
        assertEquals("usage-proposal", viewModel.state.value.usageEvents.single().id)
        assertFalse(viewModel.state.value.usageLoading)
    }

    @Test
    fun initialStateIncludesRuntimeDataDirectory() = runTest {
        val viewModel = TimeboxxingViewModel(
            fakeRuntime(dataDirectory = "/Users/tester/Library/Application Support/Timeboxxing"),
        )

        assertEquals(
            "/Users/tester/Library/Application Support/Timeboxxing",
            viewModel.state.value.dataDirectory,
        )
    }

    @Test
    fun streamedUsageEventsAreMergedIntoState() = runTest {
        val usageRepository = FakeUsageHistoryRepository()
        val viewModel = TimeboxxingViewModel(fakeRuntime(usageRepository = usageRepository))
        advanceUntilIdle()

        usageRepository.stream.emit(mockTimeboxxingData().usageEvents.first())
        runCurrent()

        assertEquals("usage-proposal", viewModel.state.value.usageEvents.single().id)
    }

    @Test
    fun amaSubmitAddsAssistantMessageOnSuccess() = runTest {
        val amaRepository = FakeAmaRepository(
            answer = AmaAnswer(
                answer = "You worked in Chrome.",
                model = "test-model",
                sources = emptyList(),
            ),
        )
        val viewModel = TimeboxxingViewModel(
            fakeRuntime(
                amaRepository = amaRepository,
                sidecarStatus = TimeboxxingSidecarStatus.Starting,
            ),
        )

        viewModel.dispatch(TimeboxxingAction.UpdateAmaInput("What did I do?"))
        viewModel.dispatch(TimeboxxingAction.SubmitAmaQuestion)
        runCurrent()

        assertEquals(listOf("What did I do?", "You worked in Chrome."), viewModel.state.value.amaMessages.map { it.content })
        assertFalse(viewModel.state.value.amaLoading)
        assertEquals(null, viewModel.state.value.amaError)
    }

    @Test
    fun amaSubmitShowsErrorOnFailure() = runTest {
        val viewModel = TimeboxxingViewModel(
            fakeRuntime(
                amaRepository = FakeAmaRepository(error = IllegalStateException("AMA failed.")),
                sidecarStatus = TimeboxxingSidecarStatus.Starting,
            ),
        )

        viewModel.dispatch(TimeboxxingAction.UpdateAmaInput("Question"))
        viewModel.dispatch(TimeboxxingAction.SubmitAmaQuestion)
        runCurrent()

        assertEquals("AMA failed.", viewModel.state.value.amaError)
        assertFalse(viewModel.state.value.amaLoading)
    }

    @Test
    fun saveSettingsUsesRepositoryAndClearsSavingState() = runTest {
        val settingsRepository = FakeSettingsRepository()
        val viewModel = TimeboxxingViewModel(
            fakeRuntime(
                settingsRepository = settingsRepository,
                sidecarStatus = TimeboxxingSidecarStatus.Starting,
            ),
        )

        viewModel.dispatch(TimeboxxingAction.SaveSettings)
        runCurrent()

        assertEquals(1, settingsRepository.savedSettings.size)
        assertFalse(viewModel.state.value.settingsSaving)
        assertEquals("Settings saved. Restarting sidecar...", viewModel.state.value.settingsSavedMessage)
    }

    @Test
    fun readyRuntimeLoadsDatabaseMaintenanceStatus() = runTest {
        val settingsRepository = FakeSettingsRepository(
            status = DatabaseMaintenanceStatus(sizeBytes = 4096),
        )
        val viewModel = TimeboxxingViewModel(fakeRuntime(settingsRepository = settingsRepository))

        advanceUntilIdle()

        assertEquals(1, settingsRepository.statusCalls)
        assertEquals(4096, viewModel.state.value.databaseMaintenanceStatus.sizeBytes)
        assertFalse(viewModel.state.value.databaseMaintenanceLoading)
    }

    @Test
    fun pruneDatabaseRangeUsesRepositoryAndRefreshesCurrentData() = runTest {
        val settingsRepository = FakeSettingsRepository(
            status = DatabaseMaintenanceStatus(sizeBytes = 4096),
            pruneResult = DatabasePruneResult(
                status = DatabaseMaintenanceStatus(sizeBytes = 2048),
                counts = DatabasePruneCounts(
                    timesheetEntriesDeleted = 1,
                    transitionEventsDeleted = 1,
                ),
            ),
            vacuumResult = DatabaseVacuumResult(
                sizeBeforeBytes = 2048,
                sizeAfterBytes = 1024,
            ),
        )
        val usageRepository = FakeUsageHistoryRepository()
        val timesheetRepository = FakeTimesheetRepository()
        val viewModel = TimeboxxingViewModel(
            fakeRuntime(
                usageRepository = usageRepository,
                settingsRepository = settingsRepository,
                timesheetRepository = timesheetRepository,
            ),
        )
        advanceUntilIdle()

        viewModel.dispatch(TimeboxxingAction.UpdateDatabasePruneStartDate(CalendarDate(2025, 5, 1)))
        viewModel.dispatch(TimeboxxingAction.UpdateDatabasePruneEndDate(CalendarDate(2025, 5, 1)))
        viewModel.dispatch(TimeboxxingAction.PruneDatabaseRange)
        advanceUntilIdle()

        assertEquals(1, settingsRepository.pruneRanges.size)
        assertEquals(1, settingsRepository.vacuumCalls)
        assertEquals(2, settingsRepository.statusCalls)
        assertEquals(2, usageRepository.getCalls)
        assertEquals(2, timesheetRepository.listCalls)
        assertFalse(viewModel.state.value.databasePruning)
        assertFalse(viewModel.state.value.databaseVacuuming)
        assertEquals(4096, viewModel.state.value.databaseMaintenanceStatus.sizeBytes)
        assertEquals("Pruned 2 database rows and compacted the database.", viewModel.state.value.databaseMaintenanceMessage)
    }

    @Test
    fun pruneDatabaseRangeSkipsVacuumWhenNoRowsDeleted() = runTest {
        val settingsRepository = FakeSettingsRepository(
            status = DatabaseMaintenanceStatus(sizeBytes = 4096),
            pruneResult = DatabasePruneResult(
                status = DatabaseMaintenanceStatus(sizeBytes = 4096),
                counts = DatabasePruneCounts(),
            ),
        )
        val viewModel = TimeboxxingViewModel(fakeRuntime(settingsRepository = settingsRepository))
        advanceUntilIdle()

        viewModel.dispatch(TimeboxxingAction.UpdateDatabasePruneStartDate(CalendarDate(2025, 5, 1)))
        viewModel.dispatch(TimeboxxingAction.UpdateDatabasePruneEndDate(CalendarDate(2025, 5, 1)))
        viewModel.dispatch(TimeboxxingAction.PruneDatabaseRange)
        advanceUntilIdle()

        assertEquals(1, settingsRepository.pruneRanges.size)
        assertEquals(0, settingsRepository.vacuumCalls)
        assertEquals("No database rows matched that range.", viewModel.state.value.databaseMaintenanceMessage)
    }

    @Test
    fun pruneDatabaseRangeReportsVacuumFailureAfterPrune() = runTest {
        val settingsRepository = FakeSettingsRepository(
            status = DatabaseMaintenanceStatus(sizeBytes = 4096),
            pruneResult = DatabasePruneResult(
                status = DatabaseMaintenanceStatus(sizeBytes = 2048),
                counts = DatabasePruneCounts(timesheetEntriesDeleted = 1),
            ),
            vacuumError = IllegalStateException("disk is full"),
        )
        val usageRepository = FakeUsageHistoryRepository()
        val timesheetRepository = FakeTimesheetRepository()
        val viewModel = TimeboxxingViewModel(
            fakeRuntime(
                usageRepository = usageRepository,
                settingsRepository = settingsRepository,
                timesheetRepository = timesheetRepository,
            ),
        )
        advanceUntilIdle()

        viewModel.dispatch(TimeboxxingAction.UpdateDatabasePruneStartDate(CalendarDate(2025, 5, 1)))
        viewModel.dispatch(TimeboxxingAction.UpdateDatabasePruneEndDate(CalendarDate(2025, 5, 1)))
        viewModel.dispatch(TimeboxxingAction.PruneDatabaseRange)
        advanceUntilIdle()

        assertEquals(1, settingsRepository.pruneRanges.size)
        assertEquals(1, settingsRepository.vacuumCalls)
        assertFalse(viewModel.state.value.databasePruning)
        assertFalse(viewModel.state.value.databaseVacuuming)
        assertEquals(
            "Database rows were pruned, but compaction failed: disk is full",
            viewModel.state.value.databaseMaintenanceError,
        )
        assertEquals(2, settingsRepository.statusCalls)
        assertEquals(2, usageRepository.getCalls)
        assertEquals(2, timesheetRepository.listCalls)
    }

    @Test
    fun readinessTransitionTriggersInitialUsageLoad() = runTest {
        val usageRepository = FakeUsageHistoryRepository(
            loadedEvents = listOf(mockTimeboxxingData().usageEvents.first()),
        )
        val runtime = fakeRuntime(
            usageRepository = usageRepository,
            sidecarStatus = TimeboxxingSidecarStatus.Starting,
        )
        val viewModel = TimeboxxingViewModel(runtime)
        advanceUntilIdle()

        runtime.sidecarStatus.value = TimeboxxingSidecarStatus.Ready
        runCurrent()

        assertEquals(1, usageRepository.getCalls)
        assertEquals("usage-proposal", viewModel.state.value.usageEvents.single().id)
    }

    @Test
    fun readyRuntimeLoadsProjects() = runTest {
        val projectRepository = FakeProjectRepository(
            loadedProjects = listOf(project(id = "client", name = "Client")),
        )
        val viewModel = TimeboxxingViewModel(fakeRuntime(projectRepository = projectRepository))

        advanceUntilIdle()

        assertEquals(1, projectRepository.listCalls)
        assertEquals(listOf("client"), viewModel.state.value.projects.map { it.id })
        assertEquals("", viewModel.state.value.draft.projectId)
    }

    @Test
    fun createProjectUsesRepositoryAndUpdatesOnSuccess() = runTest {
        val projectRepository = FakeProjectRepository(
            createdProject = project(id = "lunar-lab", name = "Lunar Lab"),
        )
        val viewModel = TimeboxxingViewModel(
            fakeRuntime(
                projectRepository = projectRepository,
                sidecarStatus = TimeboxxingSidecarStatus.Starting,
            ),
        )

        viewModel.dispatch(TimeboxxingAction.CreateProject("  Lunar Lab  ", 0xFF00FFEE))
        runCurrent()

        assertEquals(listOf("  Lunar Lab  " to 0xFF00FFEE), projectRepository.createCalls)
        assertEquals(listOf("lunar-lab"), viewModel.state.value.projects.map { it.id })
        assertEquals("lunar-lab", viewModel.state.value.draft.projectId)
        assertEquals("Created Lunar Lab.", viewModel.state.value.notice)
    }

    @Test
    fun createProjectFailureShowsNoticeWithoutAddingProject() = runTest {
        val viewModel = TimeboxxingViewModel(
            fakeRuntime(
                projectRepository = FakeProjectRepository(error = IllegalStateException("Project failed.")),
                sidecarStatus = TimeboxxingSidecarStatus.Starting,
            ),
        )

        viewModel.dispatch(TimeboxxingAction.CreateProject("Lunar Lab", 0xFF00FFEE))
        runCurrent()

        assertEquals(emptyList(), viewModel.state.value.projects)
        assertEquals("Project failed.", viewModel.state.value.notice)
    }

    @Test
    fun deleteProjectUsesRepositoryAndUpdatesOnSuccess() = runTest {
        val projectRepository = FakeProjectRepository(
            loadedProjects = listOf(
                project(id = "alpha", name = "Alpha"),
                project(id = "beta", name = "Beta"),
            ),
        )
        val viewModel = TimeboxxingViewModel(fakeRuntime(projectRepository = projectRepository))
        advanceUntilIdle()

        viewModel.dispatch(TimeboxxingAction.DeleteProject("alpha"))
        runCurrent()

        assertEquals(listOf("alpha"), projectRepository.deleteCalls)
        assertEquals(listOf("beta"), viewModel.state.value.projects.map { it.id })
        assertEquals("", viewModel.state.value.draft.projectId)
        assertEquals("Deleted project.", viewModel.state.value.notice)
    }

    @Test
    fun addDraftEntryUsesTimesheetRepositoryAndUpdatesOnSuccess() = runTest {
        val timesheetRepository = FakeTimesheetRepository()
        val viewModel = TimeboxxingViewModel(
            fakeRuntime(
                timesheetRepository = timesheetRepository,
                sidecarStatus = TimeboxxingSidecarStatus.Starting,
            ),
        )

        viewModel.dispatch(TimeboxxingAction.AddDraftEntry)
        runCurrent()

        assertEquals(1, timesheetRepository.createdEntries.size)
        assertEquals("", timesheetRepository.createdEntries.single().projectId)
        assertEquals(1, viewModel.state.value.entries.size)
        assertEquals("No project", viewModel.state.value.projectFor(viewModel.state.value.entries.single().projectId).name)
    }

    @Test
    fun exportTimesheetUsesRepositoryAndWriter() = runTest {
        val entry = TimeEntry(
            id = "entry-1",
            projectId = "",
            title = "Entry",
            notes = "",
            startMinute = 9 * 60,
            durationMinutes = 30,
            billable = true,
            sourceUsageIds = emptySet(),
        )
        val timesheetRepository = FakeTimesheetRepository(loadedEntries = listOf(entry))
        val runtime = fakeRuntime(timesheetRepository = timesheetRepository)
        val viewModel = TimeboxxingViewModel(runtime)
        advanceUntilIdle()

        viewModel.dispatch(TimeboxxingAction.ExportTimesheet(TimesheetExportFormat.Csv))
        runCurrent()

        assertEquals(listOf(TimesheetExportFormat.Csv), timesheetRepository.exportedFormats)
        assertEquals(1, runtime.timesheetExportFileWriter.savedExports.size)
        assertEquals("Exported timesheet-test.csv.", viewModel.state.value.notice)
    }
}

private fun fakeRuntime(
    usageRepository: UsageHistoryRepository = FakeUsageHistoryRepository(),
    amaRepository: AmaRepository = FakeAmaRepository(),
    settingsRepository: SettingsRepository = FakeSettingsRepository(),
    projectRepository: ProjectRepository = FakeProjectRepository(),
    timesheetRepository: TimesheetRepository = FakeTimesheetRepository(),
    sidecarStatus: TimeboxxingSidecarStatus = TimeboxxingSidecarStatus.Ready,
    dataDirectory: String = "",
): FakeTimeboxxingRuntime {
    val data = mockTimeboxxingData()
    return FakeTimeboxxingRuntime(
        usageDays = data.usageDays,
        repositories = TimeboxxingRepositories(
            usageHistoryRepository = usageRepository,
            amaRepository = amaRepository,
            settingsRepository = settingsRepository,
            projectRepository = projectRepository,
            timesheetRepository = timesheetRepository,
        ),
        sidecarStatus = sidecarStatus,
        dataDirectory = dataDirectory,
    )
}

private class FakeTimeboxxingRuntime(
    override val usageDays: List<UsageDay>,
    repositories: TimeboxxingRepositories,
    sidecarStatus: TimeboxxingSidecarStatus,
    override val dataDirectory: String,
) : TimeboxxingRuntime {
    override val initialNotice: String? = null
    override val initialAppearanceMode: AppearanceMode = AppearanceMode.System
    override val diagnosticsEnabled: Boolean = false
    override val appearanceMode = MutableStateFlow(initialAppearanceMode)
    override val repositories = MutableStateFlow(repositories)
    override val sidecarStatus = MutableStateFlow(sidecarStatus)
    override val diagnosticsLogs = MutableStateFlow(emptyList<DiagnosticsLogLine>())
    override val timesheetExportFileWriter = StaticTimesheetExportFileWriter()
    var restartCount = 0

    override suspend fun setAppearanceMode(mode: AppearanceMode) {
        appearanceMode.value = mode
    }

    override suspend fun restartSidecar() {
        restartCount++
    }
}

private class FakeUsageHistoryRepository(
    private val loadedEvents: List<UsageEvent> = emptyList(),
) : UsageHistoryRepository {
    val stream = MutableSharedFlow<UsageEvent>()
    var getCalls = 0

    override suspend fun getUsageEvents(day: UsageDay): List<UsageEvent> {
        getCalls++
        return loadedEvents
    }

    override fun watchUsageEvents(day: UsageDay): Flow<UsageEvent> = stream
}

private class FakeAmaRepository(
    private val answer: AmaAnswer = AmaAnswer("Answer", "test-model", emptyList()),
    private val error: Throwable? = null,
) : AmaRepository {
    override suspend fun ask(question: String, maxSources: Int): AmaAnswer {
        error?.let { throw it }
        return answer
    }

    override suspend fun getSemanticIndexStatus(): AmaIndexStatus =
        AmaIndexStatus(
            state = AmaIndexState.Ready,
            completedEventCount = 1,
            indexedEventCount = 1,
            pendingEventCount = 0,
            backfillRunning = false,
            message = "Ready",
        )
}

private class FakeSettingsRepository(
    private val status: DatabaseMaintenanceStatus = DatabaseMaintenanceStatus(sizeBytes = 128),
    private val pruneResult: DatabasePruneResult = DatabasePruneResult(
        status = status,
        counts = DatabasePruneCounts(),
    ),
    private val vacuumResult: DatabaseVacuumResult = DatabaseVacuumResult(
        sizeBeforeBytes = status.sizeBytes,
        sizeAfterBytes = status.sizeBytes,
    ),
    private val vacuumError: Throwable? = null,
) : SettingsRepository {
    val savedSettings = mutableListOf<AiSettings>()
    val pruneRanges = mutableListOf<DatabasePruneRange>()
    var statusCalls = 0
    var vacuumCalls = 0

    override suspend fun listModelOptions(): AiModelOptions = AiModelOptions()

    override suspend fun getAiSettings(): AiSettings = AiSettings(openRouterSecretExists = true)

    override suspend fun saveAiSettings(settings: AiSettings): AiSettings {
        savedSettings += settings
        return settings
    }

    override suspend fun getDatabaseMaintenanceStatus(): DatabaseMaintenanceStatus {
        statusCalls++
        return status
    }

    override suspend fun pruneDatabaseRange(range: DatabasePruneRange): DatabasePruneResult {
        pruneRanges += range
        return pruneResult
    }

    override suspend fun vacuumDatabase(): DatabaseVacuumResult {
        vacuumCalls++
        vacuumError?.let { throw it }
        return vacuumResult
    }
}

private class FakeProjectRepository(
    private val loadedProjects: List<Project> = emptyList(),
    private val createdProject: Project = project(id = "created", name = "Created"),
    private val error: Throwable? = null,
) : ProjectRepository {
    val createCalls = mutableListOf<Pair<String, Long>>()
    val deleteCalls = mutableListOf<String>()
    var listCalls = 0

    override suspend fun listProjects(): List<Project> {
        listCalls++
        error?.let { throw it }
        return loadedProjects
    }

    override suspend fun createProject(name: String, colorArgb: Long): Project {
        createCalls += name to colorArgb
        error?.let { throw it }
        return createdProject
    }

    override suspend fun deleteProject(projectId: String) {
        deleteCalls += projectId
        error?.let { throw it }
    }
}

private class FakeTimesheetRepository(
    private val loadedEntries: List<TimeEntry> = emptyList(),
    private val error: Throwable? = null,
) : TimesheetRepository {
    val createdEntries = mutableListOf<TimesheetEntryDraft>()
    val deletedEntries = mutableListOf<String>()
    val exportedFormats = mutableListOf<TimesheetExportFormat>()
    var listCalls = 0
    private var nextEntryNumber = 1

    override suspend fun listEntries(day: UsageDay): List<TimeEntry> {
        listCalls++
        error?.let { throw it }
        return loadedEntries
    }

    override suspend fun createEntry(day: UsageDay, draft: TimesheetEntryDraft): TimeEntry {
        createdEntries += draft
        error?.let { throw it }
        return TimeEntry(
            id = "entry-${nextEntryNumber++}",
            projectId = draft.projectId,
            title = draft.title,
            notes = draft.notes,
            startMinute = draft.startMinute,
            durationMinutes = draft.durationMinutes,
            billable = draft.billable,
            sourceUsageIds = draft.sourceUsageIds.toSet(),
        )
    }

    override suspend fun deleteEntry(entryId: String) {
        deletedEntries += entryId
        error?.let { throw it }
    }

    override suspend fun exportTimesheet(day: UsageDay, format: TimesheetExportFormat): TimesheetExport {
        exportedFormats += format
        error?.let { throw it }
        return TimesheetExport(
            fileName = "timesheet-test.${format.name.lowercase()}",
            contentType = "text/plain",
            content = byteArrayOf(1, 2, 3),
        )
    }
}

private fun project(
    id: String,
    name: String,
): Project =
    Project(
        id = id,
        name = name,
        client = "",
        colorArgb = 0xFF00FFEE,
        hourlyRateCents = 0,
    )
