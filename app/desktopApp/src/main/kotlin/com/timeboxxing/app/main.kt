package com.timeboxxing.app

import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.keyframes
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Close
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.awt.ComposeWindow
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
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
import androidx.compose.ui.input.pointer.PointerEventPass
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.FrameWindowScope
import androidx.compose.ui.window.MenuBar
import androidx.compose.ui.window.Window
import androidx.compose.ui.window.WindowPlacement
import androidx.compose.ui.window.application
import androidx.compose.ui.window.rememberWindowState
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.app.presentation.TimeboxxingSidecarStatus
import com.timeboxxing.app.ui.TbDarkColors
import com.timeboxxing.app.ui.TbLightColors
import com.timeboxxing.app.ui.DiagnosticsPane
import com.timeboxxing.app.ui.TbButton
import com.timeboxxing.app.ui.TbButtonVariant
import com.timeboxxing.app.ui.TbCard
import com.timeboxxing.app.ui.TbIcon
import com.timeboxxing.app.ui.TbSurface
import com.timeboxxing.app.ui.TbText
import com.timeboxxing.app.ui.TbTheme
import com.timeboxxing.app.ui.fonts.InterFontLoader
import kotlinx.coroutines.launch
import org.koin.compose.KoinApplication
import org.koin.compose.koinInject
import java.awt.Dimension
import java.awt.Color as AwtColor

private const val AppName = "Timeboxxing"
private const val MinWindowWidth = 760
private const val MinWindowHeight = 640
private val MacWindowControlsInset = 28.dp

@Suppress("UNUSED_PARAMETER")
fun main(args: Array<String>) {
    val javaEnv = JavaEnv.parse(DesktopBuildConfig.JavaEnv)
        ?: error("Generated DesktopBuildConfig.JavaEnv has unsupported value '${DesktopBuildConfig.JavaEnv}'")
    DesktopSentry.init(javaEnv = javaEnv)
    try {
        runDesktopApplication(javaEnv)
    } catch (throwable: Throwable) {
        DesktopSentry.captureException(throwable)
        throw throwable
    } finally {
        DesktopSentry.close()
    }
}

private fun runDesktopApplication(javaEnv: JavaEnv) {
    configureDesktopSystemProperties()

    application {
        val windowState = rememberWindowState(width = 1280.dp, height = 860.dp)
        Window(
            onCloseRequest = ::exitApplication,
            title = AppName,
            state = windowState,
            onPreviewKeyEvent = { event ->
                if (event.isPlatformCloseShortcut()) {
                    exitApplication()
                    true
                } else {
                    false
                }
            },
        ) {
            KoinApplication(
                application = {
                    modules(desktopTimeboxxingModule(javaEnv))
                },
            ) {
                val runtime = koinInject<DesktopTimeboxxingRuntime>()
                val isMacOs = remember { isMacOs() }
                var interFontFamily by remember { mutableStateOf<FontFamily?>(null) }
                val setupScope = rememberCoroutineScope()
                var diagnosticsOverlayVisible by remember { mutableStateOf(false) }
                val diagnosticsLogs by runtime.diagnosticsLogs.collectAsState()
                val sidecarStatus by runtime.sidecarStatus.collectAsState()
                val appearanceMode by runtime.appearanceMode.collectAsState()
                val systemDarkTheme = isSystemInDarkTheme()
                val darkTheme = when (appearanceMode) {
                    AppearanceMode.System -> systemDarkTheme
                    AppearanceMode.Light -> false
                    AppearanceMode.Dark -> true
                }
                val windowBackground = if (darkTheme) TbDarkColors.appBackground else TbLightColors.appBackground

                LaunchedEffect(Unit) {
                    window.minimumSize = Dimension(MinWindowWidth, MinWindowHeight)
                }

                LaunchedEffect(windowBackground, darkTheme) {
                    window.applyDesktopChrome(windowBackground, darkTheme)
                }

                LaunchedEffect(Unit) {
                    interFontFamily = InterFontLoader.loadFontFamily()
                }

                LaunchedEffect(runtime) {
                    runtime.start()
                }

                LaunchedEffect(sidecarStatus) {
                    if (sidecarStatus is TimeboxxingSidecarStatus.Starting) {
                        diagnosticsOverlayVisible = false
                    }
                }

                DisposableEffect(runtime) {
                    onDispose {
                        runtime.close()
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
                                .height(MacWindowControlsInset)
                                .pointerInput(windowState) {
                                    detectTapGestures(
                                        onDoubleTap = {
                                            windowState.placement = nextTitleBarDoubleClickPlacement(windowState.placement)
                                        },
                                    )
                                },
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
                            overlay = {
                                when (val state = sidecarStatus) {
                                    TimeboxxingSidecarStatus.Ready -> Unit
                                    TimeboxxingSidecarStatus.Starting -> SidecarStartupOverlay(state = state)
                                    is TimeboxxingSidecarStatus.Failed -> SidecarStartupOverlay(
                                        state = state,
                                        onRetry = {
                                            diagnosticsOverlayVisible = false
                                            setupScope.launch {
                                                runtime.restartSidecar()
                                            }
                                        },
                                        onDiagnostics = if (DesktopBuildConfig.DiagnosticsEnabled) {
                                            { diagnosticsOverlayVisible = true }
                                        } else {
                                            null
                                        },
                                    )
                                }
                                if (diagnosticsOverlayVisible && DesktopBuildConfig.DiagnosticsEnabled) {
                                    DiagnosticsPane(
                                        logs = diagnosticsLogs,
                                        modifier = Modifier.fillMaxSize(),
                                        onClose = { diagnosticsOverlayVisible = false },
                                    )
                                }
                            },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun SidecarStartupOverlay(
    state: TimeboxxingSidecarStatus,
    onRetry: (() -> Unit)? = null,
    onDiagnostics: (() -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    Box(
        modifier = modifier
            .fillMaxSize()
            .background(TbTheme.colors.appBackground),
        contentAlignment = Alignment.Center,
    ) {
        Box(
            modifier = Modifier
                .matchParentSize()
                .pointerInput(Unit) {
                    awaitPointerEventScope {
                        while (true) {
                            val event = awaitPointerEvent(PointerEventPass.Initial)
                            event.changes.forEach { it.consume() }
                        }
                    }
                },
        )
        when (state) {
            TimeboxxingSidecarStatus.Starting -> SidecarLoadingDots()
            is TimeboxxingSidecarStatus.Failed -> {
                TbCard(
                    modifier = Modifier
                        .widthIn(max = 460.dp)
                        .padding(24.dp),
                    color = TbTheme.colors.surface,
                ) {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(22.dp),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        SidecarFailureIcon()
                        TbText(
                            text = "Usage sidecar could not start",
                            style = TbTheme.typography.title.copy(textAlign = TextAlign.Center),
                            maxLines = 2,
                            overflow = TextOverflow.Ellipsis,
                        )
                        TbText(
                            text = state.message,
                            style = TbTheme.typography.body.copy(textAlign = TextAlign.Center),
                            color = TbTheme.colors.secondaryText,
                            maxLines = 6,
                            overflow = TextOverflow.Ellipsis,
                        )
                        if (onRetry != null || onDiagnostics != null) {
                            Row(
                                horizontalArrangement = Arrangement.spacedBy(8.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                if (onRetry != null) {
                                    TbButton(
                                        onClick = onRetry,
                                        variant = TbButtonVariant.Primary,
                                    ) {
                                        TbText("Retry", style = TbTheme.typography.button)
                                    }
                                }
                                if (onDiagnostics != null) {
                                    TbButton(
                                        onClick = onDiagnostics,
                                        variant = TbButtonVariant.Secondary,
                                    ) {
                                        TbText("Diagnostics", style = TbTheme.typography.button)
                                    }
                                }
                            }
                        }
                    }
                }
            }

            TimeboxxingSidecarStatus.Ready -> Unit
        }
    }
}

@Composable
private fun SidecarLoadingDots() {
    val transition = rememberInfiniteTransition(label = "sidecar-loading-dots")
    val dotColor = TbTheme.colors.accent
    val bouncePx = with(LocalDensity.current) { 9.dp.toPx() }

    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.height(34.dp),
    ) {
        repeat(3) { index ->
            val wave by transition.animateFloat(
                initialValue = 0f,
                targetValue = 1f,
                animationSpec = infiniteRepeatable(
                    animation = keyframes {
                        durationMillis = 900
                        0f at 0
                        0f at index * 140
                        1f at index * 140 + 180
                        0f at index * 140 + 360
                        0f at 900
                    },
                ),
                label = "sidecar-loading-dot-$index",
            )
            Canvas(
                modifier = Modifier
                    .size(12.dp)
                    .graphicsLayer {
                        alpha = 0.42f + (wave * 0.58f)
                        translationY = -bouncePx * wave
                    },
            ) {
                drawCircle(color = dotColor)
            }
        }
    }
}

@Composable
private fun SidecarFailureIcon() {
    TbSurface(
        modifier = Modifier
            .size(34.dp)
            .clip(CircleShape),
        shape = CircleShape,
        color = TbTheme.colors.destructiveSubtle,
        contentColor = TbTheme.colors.destructive,
        contentAlignment = Alignment.Center,
    ) {
        TbIcon(
            imageVector = Icons.Rounded.Close,
            contentDescription = "Sidecar startup failed",
            modifier = Modifier.size(18.dp),
        )
    }
}

private fun configureDesktopSystemProperties() {
    if (!isMacOs()) return

    System.setProperty("apple.awt.application.appearance", "system")
    System.setProperty("apple.awt.application.name", AppName)
    System.setProperty("apple.laf.useScreenMenuBar", "true")
    System.setProperty("com.apple.mrj.application.apple.menu.about.name", AppName)
}

private fun ComposeWindow.applyDesktopChrome(
    backgroundColor: Color,
    darkTheme: Boolean,
) {
    val awtBackground = backgroundColor.toAwtColor()
    background = awtBackground
    contentPane.background = awtBackground
    rootPane.background = awtBackground

    if (!isMacOs()) return

    rootPane.putClientProperty("apple.awt.fullWindowContent", true)
    rootPane.putClientProperty("apple.awt.transparentTitleBar", true)
    rootPane.putClientProperty("apple.awt.windowTitleVisible", false)
    rootPane.putClientProperty("apple.awt.windowAppearance", macOsWindowAppearanceName(darkTheme))
}

internal fun macOsWindowAppearanceName(darkTheme: Boolean): String =
    if (darkTheme) "NSAppearanceNameDarkAqua" else "NSAppearanceNameAqua"

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

internal fun nextTitleBarDoubleClickPlacement(current: WindowPlacement): WindowPlacement =
    if (current == WindowPlacement.Maximized) {
        WindowPlacement.Floating
    } else {
        WindowPlacement.Maximized
    }

private fun isMacOs(): Boolean =
    System.getProperty("os.name").startsWith("Mac", ignoreCase = true)

private fun Color.toAwtColor(): AwtColor =
    AwtColor(red, green, blue, alpha)
