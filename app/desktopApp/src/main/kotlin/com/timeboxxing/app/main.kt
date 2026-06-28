package com.timeboxxing.app

import androidx.compose.foundation.background
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.awt.ComposeWindow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.input.key.KeyEvent
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.KeyShortcut
import androidx.compose.ui.input.key.isAltPressed
import androidx.compose.ui.input.key.isCtrlPressed
import androidx.compose.ui.input.key.isMetaPressed
import androidx.compose.ui.input.key.isShiftPressed
import androidx.compose.ui.input.key.key
import androidx.compose.ui.input.key.type
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.FrameWindowScope
import androidx.compose.ui.window.MenuBar
import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import androidx.compose.ui.window.rememberWindowState
import com.timeboxxing.app.data.EmptyUsageHistoryRepository
import com.timeboxxing.app.data.AmaRepository
import com.timeboxxing.app.data.UnavailableAmaRepository
import com.timeboxxing.app.data.UnavailableUsageHistoryRepository
import com.timeboxxing.app.data.UsageHistoryRepository
import com.timeboxxing.app.data.recentUsageDays
import com.timeboxxing.app.sidecar.SidecarConnection
import com.timeboxxing.app.sidecar.SidecarProcessManager
import com.timeboxxing.app.sidecar.SidecarStartResult
import com.timeboxxing.app.state.createSidecarTimeboxxingState
import com.timeboxxing.app.ui.TbDarkColors
import com.timeboxxing.app.ui.TbLightColors
import com.timeboxxing.app.ui.fonts.InterFontLoader
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import java.awt.Dimension
import java.awt.Color as AwtColor

private const val AppName = "Timeboxxing"
private const val MinWindowWidth = 760
private const val MinWindowHeight = 640
private val MacWindowControlsInset = 28.dp

@Suppress("UNUSED_PARAMETER")
fun main(args: Array<String>) {
    configureDesktopSystemProperties()

    application {
        Window(
            onCloseRequest = ::exitApplication,
            title = AppName,
            state = rememberWindowState(width = 1280.dp, height = 860.dp),
            onPreviewKeyEvent = { event ->
                if (event.isPlatformCloseShortcut()) {
                    exitApplication()
                    true
                } else {
                    false
                }
            },
        ) {
            val darkTheme = isSystemInDarkTheme()
            val isMacOs = remember { isMacOs() }
            val windowBackground = if (darkTheme) TbDarkColors.appBackground else TbLightColors.appBackground
            var interFontFamily by remember { mutableStateOf<FontFamily?>(null) }
            val usageDays = remember { recentUsageDays() }
            val sidecarManager = remember { SidecarProcessManager() }
            val usageIconLoader = remember { NativeUsageIconLoader() }
            val workspacePanePreferences = remember { WorkspacePanePreferences() }
            val initialCollapsedPanes = remember { workspacePanePreferences.load() }
            val setupScope = rememberCoroutineScope()
            var usageHistoryRepository by remember { mutableStateOf<UsageHistoryRepository>(EmptyUsageHistoryRepository()) }
            var amaRepository by remember { mutableStateOf<AmaRepository>(UnavailableAmaRepository("Starting usage sidecar...")) }
            var sidecarConnection by remember { mutableStateOf<SidecarConnection?>(null) }
            var savedCollapsedPanes by remember { mutableStateOf(initialCollapsedPanes) }
            val latestSidecarConnection by rememberUpdatedState(sidecarConnection)

            LaunchedEffect(Unit) {
                window.minimumSize = Dimension(MinWindowWidth, MinWindowHeight)
            }

            LaunchedEffect(windowBackground) {
                window.applyDesktopChrome(windowBackground)
            }

            LaunchedEffect(Unit) {
                interFontFamily = InterFontLoader.loadFontFamily()
            }

            LaunchedEffect(Unit) {
                sidecarConnection?.close()
                sidecarConnection = null
                usageHistoryRepository = EmptyUsageHistoryRepository()
                amaRepository = UnavailableAmaRepository("Starting usage sidecar...")

                when (val result = sidecarManager.start(usageDays[usageDays.size / 2])) {
                    is SidecarStartResult.Started -> {
                        sidecarConnection = result.connection
                        usageHistoryRepository = result.connection.repository
                        amaRepository = result.connection.amaRepository
                    }

                    is SidecarStartResult.Failed -> {
                        usageHistoryRepository = UnavailableUsageHistoryRepository(result.message)
                        amaRepository = UnavailableAmaRepository(result.message)
                    }
                }
            }

            DisposableEffect(Unit) {
                onDispose {
                    latestSidecarConnection?.close()
                }
            }

            PlatformMenu(onClose = ::exitApplication)

            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .background(windowBackground),
            ) {
                if (isMacOs) {
                    Spacer(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(MacWindowControlsInset),
                    )
                }
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f),
                ) {
                    App(
                        fontFamily = interFontFamily,
                        darkTheme = darkTheme,
                        usageHistoryRepository = usageHistoryRepository,
                        amaRepository = amaRepository,
                        usageIconLoader = usageIconLoader,
                        initialState = createSidecarTimeboxxingState(
                            usageDays = usageDays,
                            initialNotice = "Starting usage sidecar...",
                            collapsedPanes = initialCollapsedPanes,
                        ),
                        onStateChange = { nextState ->
                            if (nextState.collapsedPanes != savedCollapsedPanes) {
                                val panesToSave = nextState.collapsedPanes
                                savedCollapsedPanes = panesToSave
                                setupScope.launch(Dispatchers.IO) {
                                    workspacePanePreferences.save(panesToSave)
                                }
                            }
                        }
                    )
                }
            }
        }
    }
}

private fun configureDesktopSystemProperties() {
    if (!isMacOs()) return

    System.setProperty("apple.awt.application.appearance", "system")
    System.setProperty("apple.awt.application.name", AppName)
    System.setProperty("apple.laf.useScreenMenuBar", "true")
    System.setProperty("com.apple.mrj.application.apple.menu.about.name", AppName)
}

private fun ComposeWindow.applyDesktopChrome(backgroundColor: Color) {
    val awtBackground = backgroundColor.toAwtColor()
    background = awtBackground
    contentPane.background = awtBackground
    rootPane.background = awtBackground

    if (!isMacOs()) return

    rootPane.putClientProperty("apple.awt.fullWindowContent", true)
    rootPane.putClientProperty("apple.awt.transparentTitleBar", true)
    rootPane.putClientProperty("apple.awt.windowTitleVisible", false)
}

@Composable
private fun FrameWindowScope.PlatformMenu(onClose: () -> Unit) {
    MenuBar {
        Menu("File", mnemonic = 'F') {
            Item(
                text = "Close Window",
                shortcut = platformShortcut(Key.W),
                onClick = onClose,
            )
            Item(
                text = if (isMacOs()) "Quit $AppName" else "Quit",
                shortcut = platformShortcut(Key.Q),
                onClick = onClose,
            )
        }
    }
}

private fun KeyEvent.isPlatformCloseShortcut(): Boolean {
    if (type != KeyEventType.KeyDown) return false
    if (isAltPressed || isShiftPressed) return false

    val shortcutModifierPressed = if (isMacOs()) {
        isMetaPressed && !isCtrlPressed
    } else {
        isCtrlPressed && !isMetaPressed
    }

    return shortcutModifierPressed && (key == Key.W || key == Key.Q)
}

private fun platformShortcut(key: Key): KeyShortcut =
    if (isMacOs()) KeyShortcut(key, meta = true) else KeyShortcut(key, ctrl = true)

private fun isMacOs(): Boolean =
    System.getProperty("os.name").startsWith("Mac", ignoreCase = true)

private fun Color.toAwtColor(): AwtColor =
    AwtColor(red, green, blue, alpha)
