package com.timeboxxing.app.presentation

import com.timeboxxing.data.mock.mockTimeboxxingData
import com.timeboxxing.domain.model.AmaAnswer
import com.timeboxxing.domain.model.AmaIndexState
import com.timeboxxing.domain.model.AmaIndexStatus
import com.timeboxxing.domain.model.AiModelOptions
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.domain.model.DiagnosticsLogLine
import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.model.UsageEvent
import com.timeboxxing.domain.repository.AmaRepository
import com.timeboxxing.domain.repository.SettingsRepository
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
}

private fun fakeRuntime(
    usageRepository: UsageHistoryRepository = FakeUsageHistoryRepository(),
    amaRepository: AmaRepository = FakeAmaRepository(),
    settingsRepository: SettingsRepository = FakeSettingsRepository(),
    sidecarStatus: TimeboxxingSidecarStatus = TimeboxxingSidecarStatus.Ready,
): FakeTimeboxxingRuntime {
    val data = mockTimeboxxingData()
    return FakeTimeboxxingRuntime(
        usageDays = data.usageDays,
        repositories = TimeboxxingRepositories(
            usageHistoryRepository = usageRepository,
            amaRepository = amaRepository,
            settingsRepository = settingsRepository,
        ),
        sidecarStatus = sidecarStatus,
    )
}

private class FakeTimeboxxingRuntime(
    override val usageDays: List<UsageDay>,
    repositories: TimeboxxingRepositories,
    sidecarStatus: TimeboxxingSidecarStatus,
) : TimeboxxingRuntime {
    override val initialNotice: String? = null
    override val initialAppearanceMode: AppearanceMode = AppearanceMode.System
    override val diagnosticsEnabled: Boolean = false
    override val appearanceMode = MutableStateFlow(initialAppearanceMode)
    override val repositories = MutableStateFlow(repositories)
    override val sidecarStatus = MutableStateFlow(sidecarStatus)
    override val diagnosticsLogs = MutableStateFlow(emptyList<DiagnosticsLogLine>())
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

private class FakeSettingsRepository : SettingsRepository {
    val savedSettings = mutableListOf<AiSettings>()

    override suspend fun listModelOptions(): AiModelOptions = AiModelOptions()

    override suspend fun getAiSettings(): AiSettings = AiSettings(openRouterSecretExists = true)

    override suspend fun saveAiSettings(settings: AiSettings): AiSettings {
        savedSettings += settings
        return settings
    }
}
