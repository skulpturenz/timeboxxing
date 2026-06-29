package com.timeboxxing.app

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.text.font.FontFamily
import com.timeboxxing.app.data.AmaRepository
import com.timeboxxing.app.data.SettingsRepository
import com.timeboxxing.app.data.StaticUsageHistoryRepository
import com.timeboxxing.app.data.StaticAmaRepository
import com.timeboxxing.app.data.StaticSettingsRepository
import com.timeboxxing.app.data.UsageHistoryRepository
import com.timeboxxing.app.data.mockTimeboxxingData
import com.timeboxxing.app.model.AppearanceMode
import com.timeboxxing.app.model.DiagnosticsLogLine
import com.timeboxxing.app.state.TimeboxxingAction
import com.timeboxxing.app.state.TimeboxxingScreenState
import com.timeboxxing.app.state.TimeboxxingSection
import com.timeboxxing.app.state.createInitialTimeboxxingState
import com.timeboxxing.app.state.reduceTimeboxxingState
import com.timeboxxing.app.ui.TbTheme
import com.timeboxxing.app.ui.TimeboxxingScreen
import com.timeboxxing.app.ui.NoOpUsageIconLoader
import com.timeboxxing.app.ui.UsageIconLoader
import kotlinx.coroutines.launch
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.catch

@Composable
@Preview
fun App(
    fontFamily: FontFamily? = null,
    darkTheme: Boolean = isSystemInDarkTheme(),
    usageHistoryRepository: UsageHistoryRepository = StaticUsageHistoryRepository(mockTimeboxxingData().usageEvents),
    amaRepository: AmaRepository = StaticAmaRepository(),
    settingsRepository: SettingsRepository = StaticSettingsRepository(),
    usageIconLoader: UsageIconLoader = NoOpUsageIconLoader,
    initialState: TimeboxxingScreenState = createInitialTimeboxxingState(),
    onStateChange: (TimeboxxingScreenState) -> Unit = {},
    noticeActionLabel: String? = null,
    onNoticeAction: (() -> Unit)? = null,
    overlay: @Composable (() -> Unit)? = null,
    appearanceMode: AppearanceMode = AppearanceMode.System,
    sidecarReady: Boolean = true,
    diagnosticsEnabled: Boolean = false,
    diagnosticsLogs: List<DiagnosticsLogLine> = emptyList(),
) {
    TimeboxxingApp(
        fontFamily = fontFamily,
        darkTheme = darkTheme,
        usageHistoryRepository = usageHistoryRepository,
        amaRepository = amaRepository,
        settingsRepository = settingsRepository,
        usageIconLoader = usageIconLoader,
        initialState = initialState,
        onStateChange = onStateChange,
        noticeActionLabel = noticeActionLabel,
        onNoticeAction = onNoticeAction,
        overlay = overlay,
        appearanceMode = appearanceMode,
        sidecarReady = sidecarReady,
        diagnosticsEnabled = diagnosticsEnabled,
        diagnosticsLogs = diagnosticsLogs,
    )
}

@Composable
@Preview
fun DarkAppPreview() {
    App(darkTheme = true)
}

@Composable
fun TimeboxxingApp(
    fontFamily: FontFamily? = null,
    darkTheme: Boolean = isSystemInDarkTheme(),
    usageHistoryRepository: UsageHistoryRepository = StaticUsageHistoryRepository(mockTimeboxxingData().usageEvents),
    amaRepository: AmaRepository = StaticAmaRepository(),
    settingsRepository: SettingsRepository = StaticSettingsRepository(),
    usageIconLoader: UsageIconLoader = NoOpUsageIconLoader,
    initialState: TimeboxxingScreenState = createInitialTimeboxxingState(),
    onStateChange: (TimeboxxingScreenState) -> Unit = {},
    noticeActionLabel: String? = null,
    onNoticeAction: (() -> Unit)? = null,
    overlay: @Composable (() -> Unit)? = null,
    appearanceMode: AppearanceMode = AppearanceMode.System,
    sidecarReady: Boolean = true,
    diagnosticsEnabled: Boolean = false,
    diagnosticsLogs: List<DiagnosticsLogLine> = emptyList(),
) {
    var state by remember { mutableStateOf(initialState.copy(diagnosticsEnabled = diagnosticsEnabled)) }
    LaunchedEffect(diagnosticsEnabled) {
        if (state.diagnosticsEnabled != diagnosticsEnabled) {
            val nextState = state.copy(
                diagnosticsEnabled = diagnosticsEnabled,
                selectedSection = if (!diagnosticsEnabled && state.selectedSection == TimeboxxingSection.Diagnostics) {
                    TimeboxxingSection.Overview
                } else {
                    state.selectedSection
                },
            )
            state = nextState
            onStateChange(nextState)
        }
    }
    LaunchedEffect(appearanceMode) {
        if (state.appearanceMode != appearanceMode) {
            val nextState = state.copy(appearanceMode = appearanceMode)
            state = nextState
            onStateChange(nextState)
        }
    }
    val appScope = rememberCoroutineScope()
    val selectedDay = state.selectedDay

    suspend fun loadSettings(showLoading: Boolean) {
        if (showLoading) {
            state = reduceTimeboxxingState(state, TimeboxxingAction.LoadSettings)
        }
        val loaded = runCatching {
            settingsRepository.listModelOptions() to settingsRepository.getAiSettings()
        }
        state = loaded.fold(
            onSuccess = { (options, settings) ->
                reduceTimeboxxingState(state, TimeboxxingAction.SettingsLoadSucceeded(options, settings))
            },
            onFailure = { error ->
                if (showLoading || state.selectedSection == TimeboxxingSection.Settings) {
                    reduceTimeboxxingState(
                        state,
                        TimeboxxingAction.SettingsLoadFailed(error.message ?: "Settings are unavailable."),
                    )
                } else {
                    state
                }
            },
        )
    }

    LaunchedEffect(sidecarReady, usageHistoryRepository, selectedDay.startedAtEpochMillis) {
        if (!sidecarReady) return@LaunchedEffect
        state = reduceTimeboxxingState(state, TimeboxxingAction.LoadUsage)
        val loaded = runCatching { usageHistoryRepository.getUsageEvents(selectedDay) }
        state = loaded.fold(
            onSuccess = { events ->
                reduceTimeboxxingState(
                    state,
                    TimeboxxingAction.UsageLoadSucceeded(selectedDay.startedAtEpochMillis, events),
                )
            },
            onFailure = { error ->
                reduceTimeboxxingState(
                    state,
                    TimeboxxingAction.UsageLoadFailed(
                        selectedDay.startedAtEpochMillis,
                        error.message ?: "Usage sidecar is unavailable.",
                    ),
                )
            },
        )
    }

    LaunchedEffect(sidecarReady, usageHistoryRepository, selectedDay.startedAtEpochMillis) {
        if (!sidecarReady) return@LaunchedEffect
        usageHistoryRepository.watchUsageEvents(selectedDay)
            .catch { error ->
                state = reduceTimeboxxingState(
                    state,
                    TimeboxxingAction.UsageLoadFailed(
                        selectedDay.startedAtEpochMillis,
                        error.message ?: "Usage sidecar stream stopped.",
                    ),
                )
            }
            .collect { event ->
                state = reduceTimeboxxingState(
                    state,
                    TimeboxxingAction.MergeUsageEvent(selectedDay.startedAtEpochMillis, event),
                )
            }
    }

    LaunchedEffect(sidecarReady, amaRepository, state.selectedSection) {
        if (!sidecarReady) return@LaunchedEffect
        if (state.selectedSection == TimeboxxingSection.Ama) {
            while (true) {
                val status = runCatching { amaRepository.getSemanticIndexStatus() }
                state = status.fold(
                    onSuccess = { reduceTimeboxxingState(state, TimeboxxingAction.AmaIndexStatusSucceeded(it)) },
                    onFailure = { error ->
                        reduceTimeboxxingState(
                            state,
                            TimeboxxingAction.AmaIndexStatusFailed(error.message ?: "AMA status is unavailable."),
                        )
                    },
                )
                delay(5_000)
            }
        }
    }

    LaunchedEffect(sidecarReady, settingsRepository) {
        if (!sidecarReady) return@LaunchedEffect
        loadSettings(showLoading = false)
    }

    LaunchedEffect(sidecarReady, settingsRepository, state.selectedSection) {
        if (!sidecarReady) return@LaunchedEffect
        if (state.selectedSection == TimeboxxingSection.Settings) {
            loadSettings(showLoading = true)
        }
    }

    TbTheme(
        fontFamily = fontFamily,
        darkTheme = darkTheme,
    ) {
        TimeboxxingScreen(
            state = state,
            usageIconLoader = usageIconLoader,
            diagnosticsLogs = diagnosticsLogs,
            noticeActionLabel = noticeActionLabel,
            onNoticeAction = onNoticeAction,
            onAction = { action ->
                val amaQuestion = if (
                    action == TimeboxxingAction.SubmitAmaQuestion &&
                    state.amaInput.trim().isNotEmpty() &&
                    !state.amaLoading
                ) {
                    state.amaInput.trim()
                } else {
                    null
                }
                val settingsToSave = if (
                    action == TimeboxxingAction.SaveSettings &&
                    !state.settingsSaving
                ) {
                    state.settingsDraft
                } else {
                    null
                }
                val nextState = reduceTimeboxxingState(state, action)
                state = nextState
                onStateChange(nextState)
                if (amaQuestion != null) {
                    appScope.launch {
                        val answer = runCatching { amaRepository.ask(amaQuestion) }
                        val resolvedState = answer.fold(
                            onSuccess = { reduceTimeboxxingState(state, TimeboxxingAction.AmaAnswerSucceeded(it)) },
                            onFailure = { error ->
                                reduceTimeboxxingState(
                                    state,
                                    TimeboxxingAction.AmaAnswerFailed(error.message ?: "AMA is unavailable."),
                                )
                            },
                        )
                        state = resolvedState
                        onStateChange(resolvedState)
                    }
                }
                if (settingsToSave != null) {
                    appScope.launch {
                        val saved = runCatching { settingsRepository.saveAiSettings(settingsToSave) }
                        val resolvedState = saved.fold(
                            onSuccess = { reduceTimeboxxingState(state, TimeboxxingAction.SettingsSaveSucceeded(it)) },
                            onFailure = { error ->
                                reduceTimeboxxingState(
                                    state,
                                    TimeboxxingAction.SettingsSaveFailed(error.message ?: "Settings could not be saved."),
                                )
                            },
                        )
                        state = resolvedState
                        onStateChange(resolvedState)
                    }
                }
            },
        )
        overlay?.invoke()
    }
}
