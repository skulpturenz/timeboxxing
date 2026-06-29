package com.timeboxxing.app

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.tooling.preview.Preview
import com.timeboxxing.app.di.timeboxxingPresentationModule
import com.timeboxxing.app.di.timeboxxingPreviewModule
import com.timeboxxing.app.presentation.TimeboxxingRuntime
import com.timeboxxing.app.presentation.TimeboxxingScreenState
import com.timeboxxing.app.presentation.TimeboxxingViewModel
import com.timeboxxing.app.presentation.createInitialTimeboxxingState
import com.timeboxxing.app.ui.NoOpUsageIconLoader
import com.timeboxxing.app.ui.TbTheme
import com.timeboxxing.app.ui.TimeboxxingScreen
import com.timeboxxing.app.ui.UsageIconLoader
import com.timeboxxing.domain.model.DiagnosticsLogLine
import org.koin.compose.KoinApplication
import org.koin.compose.koinInject

@Composable
fun App(
    fontFamily: FontFamily? = null,
    darkTheme: Boolean = isSystemInDarkTheme(),
    noticeActionLabel: String? = null,
    onNoticeAction: (() -> Unit)? = null,
    overlay: @Composable (() -> Unit)? = null,
) {
    val viewModel = koinInject<TimeboxxingViewModel>()
    val runtime = koinInject<TimeboxxingRuntime>()
    val usageIconLoader = koinInject<UsageIconLoader>()
    val state by viewModel.state.collectAsState()
    val diagnosticsLogs by runtime.diagnosticsLogs.collectAsState()

    TimeboxxingApp(
        fontFamily = fontFamily,
        darkTheme = darkTheme,
        state = state,
        onAction = viewModel::dispatch,
        usageIconLoader = usageIconLoader,
        diagnosticsLogs = diagnosticsLogs,
        noticeActionLabel = noticeActionLabel,
        onNoticeAction = onNoticeAction,
        overlay = overlay,
    )
}

@Composable
fun TimeboxxingApp(
    state: TimeboxxingScreenState,
    onAction: (com.timeboxxing.app.presentation.TimeboxxingAction) -> Unit,
    fontFamily: FontFamily? = null,
    darkTheme: Boolean = isSystemInDarkTheme(),
    usageIconLoader: UsageIconLoader = NoOpUsageIconLoader,
    diagnosticsLogs: List<DiagnosticsLogLine> = emptyList(),
    noticeActionLabel: String? = null,
    onNoticeAction: (() -> Unit)? = null,
    overlay: @Composable (() -> Unit)? = null,
) {
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
            onAction = onAction,
        )
        overlay?.invoke()
    }
}

@Composable
@Preview
fun AppPreview() {
    KoinApplication(
        application = {
            modules(timeboxxingPresentationModule, timeboxxingPreviewModule)
        },
    ) {
        App()
    }
}

@Composable
@Preview
fun DarkAppPreview() {
    TimeboxxingApp(
        state = createInitialTimeboxxingState(),
        onAction = {},
        darkTheme = true,
    )
}
