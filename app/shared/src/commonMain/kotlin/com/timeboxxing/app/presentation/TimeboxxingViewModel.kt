package com.timeboxxing.app.presentation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.repository.AmaRepository
import com.timeboxxing.domain.repository.SettingsRepository
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
            appearanceMode = runtime.initialAppearanceMode,
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
        observeSettingsLoads()
        observeAmaIndexStatus()
    }

    fun dispatch(action: TimeboxxingAction) {
        val currentState = _state.value
        val currentRepositories = runtime.repositories.value
        val amaQuestion = currentState.amaQuestionFor(action)
        val settingsToSave = currentState.settingsToSaveFor(action)

        reduce(action)

        if (action is TimeboxxingAction.UpdateAppearanceMode) {
            viewModelScope.launch {
                runtime.setAppearanceMode(action.mode)
            }
        }
        if (amaQuestion != null) {
            askAma(currentRepositories.amaRepository, amaQuestion)
        }
        if (settingsToSave != null) {
            saveSettings(currentRepositories.settingsRepository, settingsToSave)
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

    private fun observeSettingsLoads() {
        viewModelScope.launch {
            combine(ready, runtime.repositories) { isReady, repositories ->
                SettingsLoadRequest(isReady, repositories.settingsRepository, showLoading = false)
            }.collectLatest { request ->
                if (request.isReady) {
                    loadSettings(request.repository, request.showLoading)
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

private fun TimeboxxingScreenState.settingsToSaveFor(action: TimeboxxingAction): AiSettings? =
    if (action == TimeboxxingAction.SaveSettings && !settingsSaving) settingsDraft else null
