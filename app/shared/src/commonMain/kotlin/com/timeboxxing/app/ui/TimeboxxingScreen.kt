package com.timeboxxing.app.ui

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.MutableTransitionState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.Image
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsHoveredAsState
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.isShiftPressed
import androidx.compose.ui.input.key.key
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.input.pointer.PointerEventPass
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.layout
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextLinkStyles
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.Constraints
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.zIndex
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.CompareArrows
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowLeft
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.rounded.ListAlt
import androidx.compose.material.icons.automirrored.rounded.Send
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.Dashboard
import androidx.compose.material.icons.rounded.BugReport
import androidx.compose.material.icons.rounded.BarChart
import androidx.compose.material.icons.rounded.Edit
import androidx.compose.material.icons.rounded.ExpandMore
import androidx.compose.material.icons.rounded.Folder
import androidx.compose.material.icons.rounded.Insights
import androidx.compose.material.icons.rounded.Minimize
import androidx.compose.material.icons.rounded.QuestionAnswer
import androidx.compose.material.icons.rounded.Schedule
import androidx.compose.material.icons.rounded.Settings
import com.timeboxxing.data.time.calendarDateForEpochMillis
import com.timeboxxing.data.time.usageDayForCalendarDate
import com.timeboxxing.domain.model.AmaAppUsageChart
import com.timeboxxing.domain.model.AmaHabitSummary
import com.timeboxxing.domain.model.AmaMessage
import com.timeboxxing.domain.model.AmaMessageRole
import com.timeboxxing.domain.model.AmaIndexState
import com.timeboxxing.domain.model.AmaIndexStatus
import com.timeboxxing.domain.model.AmaQueryKind
import com.timeboxxing.domain.model.AmaSource
import com.timeboxxing.domain.model.AmaStructuredQuery
import com.timeboxxing.domain.model.AmaTimeWindow
import com.timeboxxing.domain.model.AmaUsageComparison
import com.timeboxxing.domain.model.AmaUsageTimeline
import com.timeboxxing.domain.model.AmaUsageTimelineEvent
import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.DiagnosticsLogLine
import com.timeboxxing.domain.model.formatClockTime
import com.timeboxxing.domain.model.plusDays
import com.timeboxxing.app.presentation.TimeboxxingAction
import com.timeboxxing.app.presentation.TimeboxxingScreenState
import com.timeboxxing.app.presentation.TimeboxxingSection
import app.shared.generated.resources.Res
import app.shared.generated.resources.timeboxxing_logo_dark
import app.shared.generated.resources.timeboxxing_logo_light
import com.mikepenz.markdown.compose.Markdown
import com.mikepenz.markdown.model.DefaultMarkdownColors
import com.mikepenz.markdown.model.DefaultMarkdownTypography
import com.mikepenz.markdown.model.markdownDimens
import com.mikepenz.markdown.model.markdownPadding
import org.jetbrains.compose.resources.painterResource
import kotlin.math.roundToInt
import kotlinx.coroutines.delay
import kotlin.time.Duration.Companion.milliseconds
import kotlin.time.TimeMark
import kotlin.time.TimeSource

private enum class WorkspaceLayout {
    Wide,
    Medium,
    Compact,
}

private enum class AmaPeriodPreset {
    SelectedDay,
    PreviousDay,
    Custom,
}

private data class AmaResolvedWindow(
    val window: AmaTimeWindow,
    val label: String,
    val startDate: CalendarDate,
    val endDate: CalendarDate,
)

internal enum class OverviewPane {
    UsageSchedule,
    TimeEntries,
    Projects,
}

internal enum class OverviewPaneMotionDirection {
    Minimize,
    Restore,
}

private data class OverviewPaneMotion(
    val id: Int,
    val pane: OverviewPane,
    val direction: OverviewPaneMotionDirection,
    val fromSnapshot: OverviewPaneLayoutSnapshot,
    val toSnapshot: OverviewPaneLayoutSnapshot,
)

private val WideWorkspaceMinContentWidth = 1120.dp
private val ProjectPaneWidth = 280.dp
private const val OverviewPaneAnimationMillis = 220
private const val OverviewPaneDockMillis = 220
private const val OverviewPaneTrayPulseMillis = 620L
private const val NoticeToastAnimationMillis = 180
private const val NoticeToastVisibleMillis = 4_000L
private val OverviewPaneTrayIconSize = 32.dp
private val OverviewPaneTraySpacing = 8.dp
private val OverviewPaneTrayPaddingStart = 24.dp
private val OverviewPaneTrayPaddingBottom = 24.dp
private val OverviewPaneDockEasing = CubicBezierEasing(0.18f, 0f, 0.12f, 1f)
private val AmaComposerDoubleEscapeWindow = 700.milliseconds

@Composable
fun TimeboxxingScreen(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    modifier: Modifier = Modifier,
    usageIconLoader: UsageIconLoader = NoOpUsageIconLoader,
    diagnosticsLogs: List<DiagnosticsLogLine> = emptyList(),
    noticeActionLabel: String? = null,
    onNoticeAction: (() -> Unit)? = null,
) {
    BoxWithConstraints(
        modifier = modifier
            .fillMaxSize()
            .background(TbTheme.colors.appBackground),
    ) {
        val navigationWide = maxWidth >= 860.dp
        val navigationWidth = if (navigationWide) 152.dp else 68.dp
        val contentWidth = maxWidth - navigationWidth
        val layout = when {
            contentWidth >= WideWorkspaceMinContentWidth -> WorkspaceLayout.Wide
            contentWidth >= 860.dp -> WorkspaceLayout.Medium
            else -> WorkspaceLayout.Compact
        }
        val scheduleScrollState = rememberSchedulePaneScrollState()
        var minimizedOverviewPanes by remember { mutableStateOf(emptySet<OverviewPane>()) }
        var displayedNotice by remember { mutableStateOf<String?>(null) }
        val noticeVisibilityState = remember { MutableTransitionState(false) }

        LaunchedEffect(state.notice) {
            val notice = state.notice
            if (notice == null) {
                noticeVisibilityState.targetState = false
                delay(NoticeToastAnimationMillis.toLong())
                if (!noticeVisibilityState.targetState) {
                    displayedNotice = null
                }
            } else {
                displayedNotice = notice
                noticeVisibilityState.targetState = true
            }
        }

        LaunchedEffect(state.selectedSection, state.scheduleFocusTarget?.requestId) {
            if (state.selectedSection == TimeboxxingSection.Overview && state.scheduleFocusTarget != null) {
                minimizedOverviewPanes = minimizedOverviewPanes - OverviewPane.UsageSchedule
            }
        }

        Row(
            modifier = Modifier.fillMaxSize(),
        ) {
            AppNavigationPane(
                selectedSection = state.selectedSection,
                sections = state.visibleNavigationSections,
                compact = !navigationWide,
                onAction = onAction,
                modifier = Modifier
                    .width(navigationWidth)
                    .fillMaxHeight(),
            )
            TbVerticalDivider()

            Column(
                modifier = Modifier.fillMaxSize(),
            ) {
                when (state.selectedSection) {
                    TimeboxxingSection.Overview -> {
                        AppHeader(
                            state = state,
                            compact = layout == WorkspaceLayout.Compact,
                            onAction = onAction,
                        )

                        when (layout) {
                            WorkspaceLayout.Wide -> WideWorkspace(
                                state = state,
                                onAction = onAction,
                                usageIconLoader = usageIconLoader,
                                scheduleScrollState = scheduleScrollState,
                                minimizedPanes = minimizedOverviewPanes,
                                onMinimizePane = { pane ->
                                    if (minimizedOverviewPanes.size < OverviewPane.entries.size - 1) {
                                        minimizedOverviewPanes = minimizedOverviewPanes + pane
                                    }
                                },
                                onRestorePane = { pane ->
                                    minimizedOverviewPanes = minimizedOverviewPanes - pane
                                },
                            )
                            WorkspaceLayout.Medium,
                            WorkspaceLayout.Compact -> TabbedWorkspace(state, onAction, usageIconLoader, scheduleScrollState)
                        }
                    }

                    TimeboxxingSection.Ama -> AmaPane(
                        state = state,
                        onAction = onAction,
                        modifier = Modifier.fillMaxSize(),
                    )

                    TimeboxxingSection.Diagnostics -> DiagnosticsPane(
                        logs = diagnosticsLogs,
                        modifier = Modifier.fillMaxSize(),
                    )

                    TimeboxxingSection.Settings -> SettingsPane(
                        state = state,
                        onAction = onAction,
                        modifier = Modifier.fillMaxSize(),
                    )
                }
            }
        }

        state.notice?.let { notice ->
            LaunchedEffect(notice) {
                delay(NoticeToastVisibleMillis)
                onAction(TimeboxxingAction.DismissNotice)
            }
        }

        displayedNotice?.let { notice ->
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(start = navigationWidth)
                    .zIndex(1f),
                contentAlignment = Alignment.BottomCenter,
            ) {
                AnimatedVisibility(
                    visibleState = noticeVisibilityState,
                    enter = fadeIn(animationSpec = tween(NoticeToastAnimationMillis)) +
                        slideInVertically(
                            animationSpec = tween(NoticeToastAnimationMillis),
                            initialOffsetY = { height -> height / 2 },
                        ),
                    exit = fadeOut(animationSpec = tween(NoticeToastAnimationMillis)) +
                        slideOutVertically(
                            animationSpec = tween(NoticeToastAnimationMillis),
                            targetOffsetY = { height -> height / 2 },
                        ),
                ) {
                    NoticeToast(
                        notice = notice,
                        actionLabel = noticeActionLabel,
                        onAction = onNoticeAction,
                        onDismiss = { onAction(TimeboxxingAction.DismissNotice) },
                        modifier = Modifier.padding(24.dp),
                    )
                }
            }
        }

        if (state.databaseVacuuming) {
            DatabaseVacuumOverlay(
                modifier = Modifier
                    .fillMaxSize()
                    .zIndex(2f),
            )
        }
    }
}

@Composable
private fun DatabaseVacuumOverlay(
    modifier: Modifier = Modifier,
) {
    var dotCount by remember { mutableStateOf(0) }

    LaunchedEffect(Unit) {
        while (true) {
            delay(360)
            dotCount = (dotCount + 1) % 4
        }
    }

    Box(
        modifier = modifier
            .background(TbTheme.colors.appBackground.copy(alpha = 0.76f))
            .pointerInput(Unit) {
                awaitPointerEventScope {
                    while (true) {
                        val event = awaitPointerEvent(PointerEventPass.Initial)
                        event.changes.forEach { it.consume() }
                    }
                }
            },
        contentAlignment = Alignment.Center,
    ) {
        TbCard(
            modifier = Modifier
                .widthIn(max = 360.dp)
                .padding(24.dp),
            color = TbTheme.colors.surface,
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(22.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                TbText(
                    text = "Compacting database",
                    style = TbTheme.typography.title,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbText(
                    text = "Reducing file size${".".repeat(dotCount)}",
                    style = TbTheme.typography.body,
                    color = TbTheme.colors.secondaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

@Composable
fun DiagnosticsPane(
    logs: List<DiagnosticsLogLine>,
    modifier: Modifier = Modifier,
    onClose: (() -> Unit)? = null,
) {
    val listState = rememberLazyListState()
    var followLogs by remember { mutableStateOf(true) }

    LaunchedEffect(listState) {
        snapshotFlow {
            val layoutInfo = listState.layoutInfo
            val lastVisibleIndex = layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: -1
            layoutInfo.totalItemsCount == 0 || lastVisibleIndex >= layoutInfo.totalItemsCount - 2
        }.collect { atBottom ->
            followLogs = atBottom
        }
    }

    LaunchedEffect(logs.size, followLogs) {
        if (followLogs && logs.isNotEmpty()) {
            listState.animateScrollToItem(logs.lastIndex)
        }
    }

    Column(
        modifier = modifier.background(TbTheme.colors.appBackground),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(TbTheme.colors.surface)
                .padding(horizontal = 24.dp, vertical = 18.dp),
            horizontalArrangement = Arrangement.spacedBy(14.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                TbText(
                    text = "Diagnostics",
                    style = TbTheme.typography.largeTitle,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbText(
                    text = "${logs.size} sidecar log line${if (logs.size == 1) "" else "s"} this session",
                    style = TbTheme.typography.body,
                    color = TbTheme.colors.secondaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            if (onClose != null) {
                TbIconButton(
                    icon = Icons.Rounded.Close,
                    contentDescription = "Close diagnostics",
                    onClick = onClose,
                    variant = TbButtonVariant.Ghost,
                )
            }
        }

        TbHorizontalDivider()

        LazyColumn(
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth(),
            state = listState,
            contentPadding = PaddingValues(horizontal = 24.dp, vertical = 22.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            if (logs.isEmpty()) {
                item("empty") {
                    DiagnosticsEmptyState()
                }
            } else {
                items(logs, key = { it.sequence }) { line ->
                    DiagnosticsLogRow(line)
                }
            }
        }
    }
}

@Composable
private fun DiagnosticsEmptyState() {
    TbCard(
        modifier = Modifier.fillMaxWidth(),
        color = TbTheme.colors.surface,
    ) {
        Column(
            modifier = Modifier.padding(18.dp),
            verticalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            TbText(
                text = "No sidecar logs yet",
                style = TbTheme.typography.headline,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            TbText(
                text = "Logs will appear here after the sidecar process writes output.",
                style = TbTheme.typography.body,
                color = TbTheme.colors.secondaryText,
            )
        }
    }
}

@Composable
private fun DiagnosticsLogRow(line: DiagnosticsLogLine) {
    TbSurface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(TbTheme.radii.control),
        color = TbTheme.colors.elevatedSurface,
        border = BorderStroke(Dp.Hairline, TbTheme.colors.separator),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.Top,
        ) {
            TbText(
                text = line.timestamp,
                style = TbTheme.typography.caption.copy(fontFamily = FontFamily.Monospace),
                color = TbTheme.colors.tertiaryText,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.width(92.dp),
            )
            TbText(
                text = line.message,
                style = TbTheme.typography.caption.copy(fontFamily = FontFamily.Monospace),
                color = TbTheme.colors.text,
                modifier = Modifier.weight(1f),
            )
        }
    }
}

@Composable
private fun AppNavigationPane(
    selectedSection: TimeboxxingSection,
    sections: List<TimeboxxingSection>,
    compact: Boolean,
    onAction: (TimeboxxingAction) -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .background(TbTheme.colors.surface)
            .padding(horizontal = if (compact) 8.dp else 12.dp, vertical = 14.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        if (!compact) {
            SidebarLogo(
                modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            )
        } else {
            TbSurface(
                modifier = Modifier
                    .align(Alignment.CenterHorizontally)
                    .size(36.dp),
                shape = RoundedCornerShape(TbTheme.radii.card),
                color = TbTheme.colors.accent,
                contentColor = TbTheme.colors.accentText,
                contentAlignment = Alignment.Center,
            ) {
                TbText("T", style = TbTheme.typography.title2, color = TbTheme.colors.accentText)
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        sections.forEach { section ->
            AppNavigationItem(
                section = section,
                selected = section == selectedSection,
                compact = compact,
                onClick = { onAction(TimeboxxingAction.SelectSection(section)) },
            )
        }
    }
}

@Composable
private fun SidebarLogo(modifier: Modifier = Modifier) {
    val logo = if (TbTheme.colors == TbDarkColors) {
        Res.drawable.timeboxxing_logo_dark
    } else {
        Res.drawable.timeboxxing_logo_light
    }

    Image(
        painter = painterResource(logo),
        contentDescription = "Timeboxxing",
        contentScale = ContentScale.Fit,
        modifier = modifier.size(width = 42.dp, height = 46.dp),
    )
}

@Composable
private fun AppNavigationItem(
    section: TimeboxxingSection,
    selected: Boolean,
    compact: Boolean,
    onClick: () -> Unit,
) {
    val colors = TbTheme.colors
    val interactionSource = remember { MutableInteractionSource() }
    val hovered by interactionSource.collectIsHoveredAsState()
    val pressed by interactionSource.collectIsPressedAsState()
    val background = when {
        selected -> colors.accentSubtle
        pressed || hovered -> colors.controlFill
        else -> Color.Transparent
    }
    val contentColor = when {
        selected -> colors.text
        hovered || pressed -> colors.text
        else -> colors.secondaryText
    }

    TbTooltip(text = section.label, enabled = compact) {
        TbSurface(
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = if (compact) 46.dp else 40.dp)
                .clip(RoundedCornerShape(TbTheme.radii.control))
                .clickable(
                    interactionSource = interactionSource,
                    indication = null,
                    onClick = onClick,
                ),
            shape = RoundedCornerShape(TbTheme.radii.control),
            color = background,
            contentColor = contentColor,
            contentAlignment = Alignment.CenterStart,
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = if (compact) 0.dp else 10.dp, vertical = 9.dp),
                horizontalArrangement = if (compact) Arrangement.Center else Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                TbIcon(
                    imageVector = section.icon,
                    contentDescription = null,
                    tint = contentColor,
                    modifier = Modifier.size(18.dp),
                )
                if (!compact) {
                    TbText(
                        text = section.label,
                        style = TbTheme.typography.button,
                        color = contentColor,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
    }
}

@Composable
private fun AppHeader(
    state: TimeboxxingScreenState,
    compact: Boolean,
    onAction: (TimeboxxingAction) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(TbTheme.colors.surface)
            .padding(horizontal = if (compact) 16.dp else 24.dp, vertical = 14.dp),
        horizontalArrangement = Arrangement.End,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        DateControls(
            state = state,
            onAction = onAction,
            modifier = if (compact) Modifier.fillMaxWidth() else Modifier,
        )
    }
}

@Composable
private fun DateControls(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(8.dp, Alignment.End),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        TbIconButton(
            icon = Icons.AutoMirrored.Rounded.KeyboardArrowLeft,
            contentDescription = "Previous day",
            onClick = { onAction(TimeboxxingAction.MoveDate(-1)) },
            variant = TbButtonVariant.Ghost,
        )

        DatePickerMenu(
            state = state,
            onDateSelected = { onAction(TimeboxxingAction.SelectDate(it)) },
        )

        TbIconButton(
            icon = Icons.AutoMirrored.Rounded.KeyboardArrowRight,
            contentDescription = "Next day",
            onClick = { onAction(TimeboxxingAction.MoveDate(1)) },
            variant = TbButtonVariant.Ghost,
        )

        ZoomMenu(
            zoomMinutes = state.zoomMinutes,
            onZoomChange = { onAction(TimeboxxingAction.ChangeZoom(it)) },
        )
    }
}

@Composable
private fun DatePickerMenu(
    state: TimeboxxingScreenState,
    onDateSelected: (CalendarDate) -> Unit,
) {
    CalendarDatePickerMenu(
        selectedDate = state.selectedCalendarDate,
        label = state.dateLabel,
        onDateSelected = onDateSelected,
        modifier = Modifier.widthIn(min = 300.dp, max = 360.dp),
        textStyle = TbTheme.typography.title2,
    )
}

@Composable
private fun ZoomMenu(
    zoomMinutes: Int,
    onZoomChange: (Int) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }

    TbMenu(
        expanded = expanded,
        onExpandedChange = { expanded = it },
        anchor = {
            TbButton(
                modifier = Modifier.widthIn(min = 116.dp),
                onClick = { expanded = true },
                variant = TbButtonVariant.Secondary,
            ) {
                TbText("Zoom: ${zoomMinutes}m", style = TbTheme.typography.button)
                TbIcon(
                    imageVector = Icons.Rounded.ExpandMore,
                    contentDescription = null,
                    modifier = Modifier
                        .padding(start = 6.dp)
                        .size(16.dp),
                )
            }
        },
        panelContent = {
            listOf(10, 15, 30, 60).forEach { minutes ->
                TbMenuItem(
                    onClick = {
                        expanded = false
                        onZoomChange(minutes)
                    },
                ) {
                    TbText(
                        text = "${minutes} minutes",
                        style = TbTheme.typography.body,
                    )
                }
            }
        },
    )
}

@Composable
private fun NoticeToast(
    notice: String,
    actionLabel: String?,
    onAction: (() -> Unit)?,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
) {
    TbSurface(
        modifier = modifier
            .widthIn(max = 560.dp)
            .fillMaxWidth(),
        color = TbTheme.colors.surface,
        shape = RoundedCornerShape(TbTheme.radii.panel),
        border = BorderStroke(Dp.Hairline, TbTheme.colors.separator),
        shadowElevation = 12.dp,
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 14.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TbText(
                modifier = Modifier.weight(1f),
                text = notice,
                color = TbTheme.colors.text,
                style = TbTheme.typography.body,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            if (actionLabel != null && onAction != null) {
                TbButton(onClick = onAction, variant = TbButtonVariant.Secondary) {
                    TbText(actionLabel, style = TbTheme.typography.button)
                }
            }
            TbIconButton(
                icon = Icons.Rounded.Close,
                contentDescription = "Dismiss notice",
                onClick = onDismiss,
                variant = TbButtonVariant.Ghost,
            )
        }
    }
}

@Composable
private fun WideWorkspace(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    usageIconLoader: UsageIconLoader,
    scheduleScrollState: SchedulePaneScrollState,
    minimizedPanes: Set<OverviewPane>,
    onMinimizePane: (OverviewPane) -> Unit,
    onRestorePane: (OverviewPane) -> Unit,
) {
    var activeMotion by remember { mutableStateOf<OverviewPaneMotion?>(null) }
    var motionProgress by remember { mutableStateOf(1f) }
    var pulsingTrayPane by remember { mutableStateOf<OverviewPane?>(null) }
    var nextMotionSequence by remember { mutableStateOf(0) }
    val density = LocalDensity.current
    val projectPaneWidthPx = with(density) { ProjectPaneWidth.toPx() }
    val dividerWidthPx = with(density) { Dp.Hairline.toPx().coerceAtLeast(1f) }

    fun nextMotionId(): Int {
        val id = nextMotionSequence
        nextMotionSequence += 1
        return id
    }

    fun snapshotFor(workspaceBounds: Rect, panes: Set<OverviewPane>): OverviewPaneLayoutSnapshot =
        overviewPaneLayoutSnapshot(
            workspaceBounds = workspaceBounds,
            minimizedPanes = panes,
            projectPaneWidthPx = projectPaneWidthPx,
            dividerWidthPx = dividerWidthPx,
        )

    fun finishMotion(motion: OverviewPaneMotion) {
        if (activeMotion?.id != motion.id) {
            return
        }
        motionProgress = 1f
        when (motion.direction) {
            OverviewPaneMotionDirection.Minimize -> {
                activeMotion = null
                pulsingTrayPane = motion.pane
            }
            OverviewPaneMotionDirection.Restore -> activeMotion = null
        }
    }

    LaunchedEffect(activeMotion?.id) {
        val motion = activeMotion ?: return@LaunchedEffect
        val progress = Animatable(0f)
        motionProgress = 0f
        progress.animateTo(
            targetValue = 1f,
            animationSpec = tween(
                durationMillis = OverviewPaneDockMillis,
                easing = OverviewPaneDockEasing,
            ),
        ) {
            motionProgress = value
        }
        if (activeMotion?.id == motion.id) {
            finishMotion(motion)
        }
    }

    LaunchedEffect(pulsingTrayPane) {
        if (pulsingTrayPane != null) {
            delay(OverviewPaneTrayPulseMillis)
            pulsingTrayPane = null
        }
    }

    BoxWithConstraints(
        modifier = Modifier.fillMaxSize(),
    ) {
        val workspaceBounds = Rect(
            left = 0f,
            top = 0f,
            right = with(density) { maxWidth.toPx() },
            bottom = with(density) { maxHeight.toPx() },
        )
        val currentSnapshot = snapshotFor(workspaceBounds, minimizedPanes)
        val active = activeMotion
        val renderedSnapshot = if (active != null) {
            overviewPaneMotionLayoutSnapshot(
                from = active.fromSnapshot,
                to = active.toSnapshot,
                activePane = active.pane,
                direction = active.direction,
                progress = motionProgress,
                dividerWidthPx = dividerWidthPx,
            )
        } else {
            currentSnapshot
        }
        val layoutReady = workspaceBounds.width > 0f && workspaceBounds.height > 0f
        val controlsEnabled = active == null && layoutReady
        val canMinimize = controlsEnabled && currentSnapshot.paneBounds.size > 1

        fun minimizePane(pane: OverviewPane) {
            if (!canMinimize) {
                return
            }
            if (currentSnapshot.paneBounds[pane] == null) {
                return
            }
            val targetMinimizedPanes = minimizedPanes + pane
            val targetSnapshot = snapshotFor(workspaceBounds, targetMinimizedPanes)
            val id = nextMotionId()
            motionProgress = 0f
            activeMotion = OverviewPaneMotion(
                id = id,
                pane = pane,
                direction = OverviewPaneMotionDirection.Minimize,
                fromSnapshot = currentSnapshot,
                toSnapshot = targetSnapshot,
            )
            onMinimizePane(pane)
        }

        fun restorePane(pane: OverviewPane) {
            if (!layoutReady || pane !in minimizedPanes) {
                return
            }
            if (active != null) {
                activeMotion = null
                motionProgress = 1f
                onRestorePane(pane)
                return
            }
            val targetMinimizedPanes = minimizedPanes - pane
            val targetSnapshot = snapshotFor(workspaceBounds, targetMinimizedPanes)
            if (targetSnapshot.paneBounds[pane] == null) {
                return
            }
            val id = nextMotionId()
            if (pulsingTrayPane == pane) {
                pulsingTrayPane = null
            }
            motionProgress = 0f
            activeMotion = OverviewPaneMotion(
                id = id,
                pane = pane,
                direction = OverviewPaneMotionDirection.Restore,
                fromSnapshot = currentSnapshot,
                toSnapshot = targetSnapshot,
            )
            onRestorePane(pane)
        }

        Box(modifier = Modifier.fillMaxSize()) {
            OverviewPane.entries.forEach { pane ->
                val bounds = renderedSnapshot.paneBounds[pane] ?: return@forEach
                PositionedOverviewPane(
                    pane = pane,
                    bounds = bounds,
                    state = state,
                    usageIconLoader = usageIconLoader,
                    scheduleScrollState = scheduleScrollState,
                    onAction = onAction,
                    canMinimize = canMinimize,
                    motionDirection = active?.direction?.takeIf { active.pane == pane },
                    motionProgress = motionProgress,
                    onMinimizePane = { minimizePane(pane) },
                )
            }

            renderedSnapshot.dividerBounds.values.forEach { bounds ->
                TbVerticalDivider(
                    modifier = Modifier
                        .overviewPaneBounds(bounds)
                        .zIndex(1f),
                )
            }

            MinimizedOverviewPaneTray(
                visiblePanes = if (active?.direction == OverviewPaneMotionDirection.Restore) {
                    minimizedPanes + active.pane
                } else {
                    minimizedPanes
                },
                pulsingPane = pulsingTrayPane,
                enabled = layoutReady,
                onRestorePane = ::restorePane,
                modifier = Modifier
                    .align(Alignment.BottomStart)
                    .padding(
                        start = OverviewPaneTrayPaddingStart,
                        bottom = OverviewPaneTrayPaddingBottom,
                    )
                    .zIndex(3f),
            )
        }
    }
}

@Composable
private fun PositionedOverviewPane(
    pane: OverviewPane,
    bounds: Rect,
    state: TimeboxxingScreenState,
    usageIconLoader: UsageIconLoader,
    scheduleScrollState: SchedulePaneScrollState,
    onAction: (TimeboxxingAction) -> Unit,
    canMinimize: Boolean,
    motionDirection: OverviewPaneMotionDirection?,
    motionProgress: Float,
    onMinimizePane: () -> Unit,
) {
    if (bounds.width <= 0f || bounds.height <= 0f) {
        return
    }

    val dockOffsetY = with(LocalDensity.current) { 8.dp.toPx() }
    val dockTransform = motionDirection?.let {
        overviewPaneDockTransformValues(
            direction = it,
            progress = motionProgress,
            offsetPx = dockOffsetY,
        )
    }
    val motionShape = RoundedCornerShape(TbTheme.radii.card)

    Box(
        modifier = Modifier
            .overviewPaneBounds(bounds)
            .zIndex(if (dockTransform != null) 2f else 0f)
            .graphicsLayer {
                if (dockTransform != null) {
                    alpha = dockTransform.alpha
                    scaleX = dockTransform.scale
                    scaleY = dockTransform.scale
                    translationY = dockTransform.translationY
                    shape = motionShape
                    clip = true
                }
            },
    ) {
        val headerAction: (@Composable () -> Unit)? = if (canMinimize) {
            {
                OverviewPaneMinimizeButton(
                    pane = pane,
                    onClick = onMinimizePane,
                )
            }
        } else {
            null
        }

        when (pane) {
            OverviewPane.UsageSchedule -> SchedulePane(
                state = state,
                usageIconLoader = usageIconLoader,
                onUsageClick = { onAction(TimeboxxingAction.ToggleUsageSelection(it)) },
                onClearSelection = { onAction(TimeboxxingAction.ClearUsageSelection) },
                onCreateEntry = { onAction(TimeboxxingAction.AddDraftEntry) },
                scrollState = scheduleScrollState,
                modifier = Modifier.fillMaxSize(),
                headerAction = headerAction,
            )
            OverviewPane.TimeEntries -> EntryBuilderPane(
                state = state,
                onAction = onAction,
                showInlineSummary = false,
                modifier = Modifier.fillMaxSize(),
                headerAction = headerAction,
            )
            OverviewPane.Projects -> ProjectSummaryRail(
                state = state,
                onAction = onAction,
                modifier = Modifier.fillMaxSize(),
                headerAction = headerAction,
            )
        }
    }
}

private fun Modifier.overviewPaneBounds(bounds: Rect): Modifier =
    layout { measurable, _ ->
        val width = bounds.width.roundToInt().coerceAtLeast(0)
        val height = bounds.height.roundToInt().coerceAtLeast(0)
        val placeable = measurable.measure(Constraints.fixed(width, height))
        layout(width, height) {
            placeable.place(0, 0)
        }
    }.offset {
        IntOffset(
            x = bounds.left.roundToInt(),
            y = bounds.top.roundToInt(),
        )
    }

@Composable
private fun OverviewPaneMinimizeButton(
    pane: OverviewPane,
    onClick: () -> Unit,
) {
    TbIconButton(
        icon = Icons.Rounded.Minimize,
        contentDescription = "Minimize ${pane.label}",
        onClick = onClick,
        variant = TbButtonVariant.Ghost,
    )
}

@Composable
private fun MinimizedOverviewPaneTray(
    visiblePanes: Set<OverviewPane>,
    pulsingPane: OverviewPane?,
    enabled: Boolean,
    onRestorePane: (OverviewPane) -> Unit,
    modifier: Modifier = Modifier,
) {
    if (visiblePanes.isEmpty()) {
        return
    }

    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(OverviewPaneTraySpacing),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        OverviewPane.entries.forEach { pane ->
            val isPulsing = pulsingPane == pane
            val pulseScale by animateFloatAsState(
                targetValue = if (isPulsing) 1.14f else 1f,
                animationSpec = tween(OverviewPaneTrayPulseMillis.toInt()),
                label = "${pane.name}-tray-pulse",
            )
            val ringScale by animateFloatAsState(
                targetValue = if (isPulsing) 1.52f else 0.82f,
                animationSpec = tween(OverviewPaneTrayPulseMillis.toInt()),
                label = "${pane.name}-tray-ring-scale",
            )
            val ringAlpha by animateFloatAsState(
                targetValue = if (isPulsing) 0.44f else 0f,
                animationSpec = tween(OverviewPaneTrayPulseMillis.toInt()),
                label = "${pane.name}-tray-ring-alpha",
            )
            AnimatedVisibility(
                visible = pane in visiblePanes,
                enter = fadeIn(animationSpec = tween(OverviewPaneAnimationMillis)) +
                    slideInVertically(
                        animationSpec = tween(OverviewPaneAnimationMillis),
                        initialOffsetY = { it / 2 },
                    ),
                exit = fadeOut(animationSpec = tween(OverviewPaneAnimationMillis)) +
                    slideOutVertically(
                        animationSpec = tween(OverviewPaneAnimationMillis),
                        targetOffsetY = { it / 2 },
                    ),
            ) {
                Box(
                    contentAlignment = Alignment.Center,
                ) {
                    TbSurface(
                        modifier = Modifier
                            .size(OverviewPaneTrayIconSize)
                            .graphicsLayer {
                                alpha = ringAlpha
                                scaleX = ringScale
                                scaleY = ringScale
                        },
                        shape = RoundedCornerShape(TbTheme.radii.pill),
                        color = TbTheme.colors.accent.copy(alpha = 0.22f),
                        border = BorderStroke(1.dp, TbTheme.colors.accent.copy(alpha = 0.58f)),
                        contentAlignment = Alignment.Center,
                    ) {}
                    Box(
                        modifier = Modifier
                            .size(OverviewPaneTrayIconSize)
                            .graphicsLayer {
                                scaleX = pulseScale
                                scaleY = pulseScale
                            },
                    ) {
                        TbIconButton(
                            icon = pane.icon,
                            contentDescription = "Restore ${pane.label}",
                            onClick = { onRestorePane(pane) },
                            enabled = enabled,
                            variant = TbButtonVariant.Secondary,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun TabbedWorkspace(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    usageIconLoader: UsageIconLoader,
    scheduleScrollState: SchedulePaneScrollState,
) {
    var selectedTab by remember { mutableStateOf(0) }
    val tabs = listOf("Usage", "Entries", "Projects")

    LaunchedEffect(state.scheduleFocusTarget?.requestId) {
        if (state.scheduleFocusTarget != null) {
            selectedTab = 0
        }
    }

    Column(
        modifier = Modifier.fillMaxSize(),
    ) {
        TbTabs(
            selectedIndex = selectedTab,
            labels = tabs,
            onSelectedIndexChange = { selectedTab = it },
        )
        TbHorizontalDivider()

        when (selectedTab) {
            0 -> SchedulePane(
                state = state,
                usageIconLoader = usageIconLoader,
                onUsageClick = { onAction(TimeboxxingAction.ToggleUsageSelection(it)) },
                onClearSelection = { onAction(TimeboxxingAction.ClearUsageSelection) },
                onCreateEntry = { onAction(TimeboxxingAction.AddDraftEntry) },
                scrollState = scheduleScrollState,
                modifier = Modifier.fillMaxSize(),
            )

            1 -> EntryBuilderPane(
                state = state,
                onAction = onAction,
                showInlineSummary = true,
                modifier = Modifier
                    .fillMaxSize()
                    .clip(RoundedCornerShape(0.dp)),
            )

            else -> ProjectSummaryRail(
                state = state,
                onAction = onAction,
                modifier = Modifier.fillMaxSize(),
            )
        }
    }
}

@Composable
private fun AmaPane(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    modifier: Modifier = Modifier,
) {
    val listState = rememberLazyListState()
    val messageCount = state.amaMessages.size + if (state.amaLoading) 1 else 0
    val hasMessages = state.amaMessages.isNotEmpty()
    val defaultInsightDate = amaDefaultCalendarDate(state)
    var selectedInsightKind by remember { mutableStateOf(AmaQueryKind.AppTotals) }
    var selectedInsightPreset by remember { mutableStateOf(AmaPeriodPreset.SelectedDay) }
    var customInsightStartDate by remember(defaultInsightDate) { mutableStateOf(defaultInsightDate) }
    var customInsightEndDate by remember(defaultInsightDate) { mutableStateOf(defaultInsightDate) }
    var includeInsightIdle by remember { mutableStateOf(false) }
    var insightLimit by remember { mutableStateOf(5) }
    val insightLimitOptions = amaLimitOptionsFor(selectedInsightKind)
    val selectInsightKind: (AmaQueryKind) -> Unit = { kind ->
        selectedInsightKind = kind
        val nextLimitOptions = amaLimitOptionsFor(kind)
        if (insightLimit !in nextLimitOptions) {
            insightLimit = nextLimitOptions.first()
        }
    }
    val insightQuery = buildAmaStructuredQuery(
        state = state,
        kind = selectedInsightKind,
        preset = selectedInsightPreset,
        customStartDate = customInsightStartDate,
        customEndDate = customInsightEndDate,
        limit = insightLimit,
        includeIdle = includeInsightIdle,
    )
    val submitInsight: () -> Unit = {
        insightQuery?.let { query ->
            onAction(TimeboxxingAction.SubmitAmaStructuredQuery(query))
        }
    }

    LaunchedEffect(selectedInsightKind) {
        if (insightLimit !in insightLimitOptions) {
            insightLimit = insightLimitOptions.first()
        }
    }

    LaunchedEffect(messageCount) {
        if (messageCount > 0) {
            listState.animateScrollToItem(messageCount - 1)
        }
    }

    Column(
        modifier = modifier.background(TbTheme.colors.appBackground),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(TbTheme.colors.surface)
                .padding(horizontal = 24.dp, vertical = 18.dp),
            horizontalArrangement = Arrangement.spacedBy(14.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                TbText(
                    text = "AMA",
                    style = TbTheme.typography.largeTitle,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbText(
                    text = "Ask questions about indexed usage history",
                    style = TbTheme.typography.body,
                    color = TbTheme.colors.secondaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                state.amaIndexStatus?.let { status ->
                    Spacer(modifier = Modifier.height(8.dp))
                    AmaIndexStatusBadge(status)
                }
            }
            TbIconButton(
                icon = Icons.Rounded.Delete,
                contentDescription = "Clear AMA chat",
                onClick = { onAction(TimeboxxingAction.ClearAmaChat) },
                enabled = state.amaMessages.isNotEmpty() || state.amaInput.isNotEmpty() || state.amaError != null,
                variant = TbButtonVariant.Ghost,
            )
        }

        TbHorizontalDivider()

        LazyColumn(
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth(),
            state = listState,
            contentPadding = androidx.compose.foundation.layout.PaddingValues(horizontal = 24.dp, vertical = 22.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            if (!hasMessages && !state.amaLoading) {
                item("empty") {
                    AmaEmptyState(
                        state = state,
                        value = state.amaInput,
                        loading = state.amaLoading,
                        onValueChange = { onAction(TimeboxxingAction.UpdateAmaInput(it)) },
                        onSubmit = { onAction(TimeboxxingAction.SubmitAmaQuestion) },
                        selectedKind = selectedInsightKind,
                        onSelectedKindChange = selectInsightKind,
                        selectedPreset = selectedInsightPreset,
                        onSelectedPresetChange = { selectedInsightPreset = it },
                        customStartDate = customInsightStartDate,
                        customEndDate = customInsightEndDate,
                        onCustomStartDateChange = { customInsightStartDate = it },
                        onCustomEndDateChange = { customInsightEndDate = it },
                        includeIdle = includeInsightIdle,
                        onIncludeIdleChange = { includeInsightIdle = it },
                        limit = insightLimit,
                        limitOptions = insightLimitOptions,
                        onLimitChange = { insightLimit = it },
                        canSubmitInsight = insightQuery != null && !state.amaLoading,
                        onSubmitInsight = submitInsight,
                    )
                }
            }
            items(state.amaMessages, key = { it.id }) { message ->
                AmaMessageRow(
                    message = message,
                    onOpenUsageSource = { startedAtEpochMillis, transitionEventId ->
                        onAction(TimeboxxingAction.OpenAmaUsageSource(startedAtEpochMillis, transitionEventId))
                    },
                )
            }
            if (state.amaLoading) {
                item("loading") {
                    AmaLoadingRow()
                }
            }
        }

        state.amaError?.let { error ->
            AmaErrorBanner(error)
        }

        if (hasMessages || state.amaLoading) {
            AmaInsightDock(
                state = state,
                loading = state.amaLoading,
                selectedKind = selectedInsightKind,
                onSelectedKindChange = selectInsightKind,
                selectedPreset = selectedInsightPreset,
                onSelectedPresetChange = { selectedInsightPreset = it },
                customStartDate = customInsightStartDate,
                customEndDate = customInsightEndDate,
                onCustomStartDateChange = { customInsightStartDate = it },
                onCustomEndDateChange = { customInsightEndDate = it },
                includeIdle = includeInsightIdle,
                onIncludeIdleChange = { includeInsightIdle = it },
                limit = insightLimit,
                limitOptions = insightLimitOptions,
                onLimitChange = { insightLimit = it },
                canSubmit = insightQuery != null && !state.amaLoading,
                onSubmit = submitInsight,
            )
        }

        if (hasMessages) {
            AmaFreeformComposer(
                value = state.amaInput,
                loading = state.amaLoading,
                onValueChange = { onAction(TimeboxxingAction.UpdateAmaInput(it)) },
                onSubmit = { onAction(TimeboxxingAction.SubmitAmaQuestion) },
                framed = true,
                singleLine = true,
            )
        }
    }
}

@Composable
private fun AmaIndexStatusBadge(status: AmaIndexStatus) {
    val background = when (status.state) {
        AmaIndexState.Ready -> TbTheme.colors.successSubtle
        AmaIndexState.Indexing -> TbTheme.colors.warningSubtle
        AmaIndexState.Empty -> TbTheme.colors.controlFill
        AmaIndexState.Unavailable -> TbTheme.colors.destructiveSubtle
        AmaIndexState.Unknown -> TbTheme.colors.controlFill
    }
    val foreground = when (status.state) {
        AmaIndexState.Ready -> TbTheme.colors.success
        AmaIndexState.Indexing -> TbTheme.colors.warning
        AmaIndexState.Empty -> TbTheme.colors.secondaryText
        AmaIndexState.Unavailable -> TbTheme.colors.destructive
        AmaIndexState.Unknown -> TbTheme.colors.secondaryText
    }
    val detail = when (status.state) {
        AmaIndexState.Indexing -> " ${status.indexedEventCount}/${status.completedEventCount} indexed"
        AmaIndexState.Ready -> " ${status.indexedEventCount} indexed"
        else -> ""
    }

    TbBadge(
        label = "${status.message}$detail",
        color = foreground,
        background = background,
    )
}

@Composable
private fun AmaEmptyState(
    state: TimeboxxingScreenState,
    value: String,
    loading: Boolean,
    onValueChange: (String) -> Unit,
    onSubmit: () -> Unit,
    selectedKind: AmaQueryKind,
    onSelectedKindChange: (AmaQueryKind) -> Unit,
    selectedPreset: AmaPeriodPreset,
    onSelectedPresetChange: (AmaPeriodPreset) -> Unit,
    customStartDate: CalendarDate,
    customEndDate: CalendarDate,
    onCustomStartDateChange: (CalendarDate) -> Unit,
    onCustomEndDateChange: (CalendarDate) -> Unit,
    includeIdle: Boolean,
    onIncludeIdleChange: (Boolean) -> Unit,
    limit: Int,
    limitOptions: List<Int>,
    onLimitChange: (Int) -> Unit,
    canSubmitInsight: Boolean,
    onSubmitInsight: () -> Unit,
) {
    var freeformOpen by remember { mutableStateOf(false) }
    TbCard(
        modifier = Modifier
            .fillMaxWidth()
            .widthIn(max = 940.dp),
        color = TbTheme.colors.surface,
        borderColor = TbTheme.colors.separator,
    ) {
        Column(
            modifier = Modifier.padding(18.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            BoxWithConstraints(modifier = Modifier.fillMaxWidth()) {
                val compact = maxWidth < 620.dp
                if (compact) {
                    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                        AmaEmptyStateCopy()
                        AmaEmptyStateActions(
                            loading = loading,
                            selectedKind = selectedKind,
                            canSubmitInsight = canSubmitInsight,
                            freeformOpen = freeformOpen,
                            onSubmitInsight = onSubmitInsight,
                            onToggleFreeform = { freeformOpen = !freeformOpen },
                        )
                    }
                } else {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(16.dp),
                        verticalAlignment = Alignment.Top,
                    ) {
                        AmaEmptyStateCopy(modifier = Modifier.weight(1f))
                        AmaEmptyStateActions(
                            loading = loading,
                            selectedKind = selectedKind,
                            canSubmitInsight = canSubmitInsight,
                            freeformOpen = freeformOpen,
                            onSubmitInsight = onSubmitInsight,
                            onToggleFreeform = { freeformOpen = !freeformOpen },
                        )
                    }
                }
            }

            AnimatedVisibility(visible = freeformOpen) {
                AmaFreeformComposer(
                    value = value,
                    loading = loading,
                    onValueChange = onValueChange,
                    onSubmit = onSubmit,
                    framed = false,
                    singleLine = false,
                )
            }

            TbHorizontalDivider()

            AmaInsightLauncher(
                state = state,
                selectedKind = selectedKind,
                onSelectedKindChange = onSelectedKindChange,
                selectedPreset = selectedPreset,
                onSelectedPresetChange = onSelectedPresetChange,
                customStartDate = customStartDate,
                customEndDate = customEndDate,
                onCustomStartDateChange = onCustomStartDateChange,
                onCustomEndDateChange = onCustomEndDateChange,
                includeIdle = includeIdle,
                onIncludeIdleChange = onIncludeIdleChange,
                limit = limit,
                limitOptions = limitOptions,
                onLimitChange = onLimitChange,
                showRunAction = false,
                canSubmit = canSubmitInsight,
                onSubmit = onSubmitInsight,
            )
        }
    }
}

@Composable
private fun AmaEmptyStateCopy(modifier: Modifier = Modifier) {
    Column(
        modifier = modifier,
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        TbText(
            text = "Ask about your usage history",
            style = TbTheme.typography.headline,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        TbText(
            text = "Summarize apps, timelines, habits, and comparisons from your indexed activity.",
            style = TbTheme.typography.body,
            color = TbTheme.colors.secondaryText,
        )
    }
}

@Composable
private fun AmaEmptyStateActions(
    loading: Boolean,
    selectedKind: AmaQueryKind,
    canSubmitInsight: Boolean,
    freeformOpen: Boolean,
    onSubmitInsight: () -> Unit,
    onToggleFreeform: () -> Unit,
) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        TbButton(
            onClick = onSubmitInsight,
            enabled = !loading && canSubmitInsight,
        ) {
            TbIcon(
                imageVector = selectedKind.icon,
                contentDescription = null,
                modifier = Modifier
                    .padding(end = 6.dp)
                    .size(17.dp),
            )
            TbText(
                text = "Run ${selectedKind.label}",
                style = TbTheme.typography.button,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        TbButton(
            onClick = onToggleFreeform,
            variant = TbButtonVariant.Secondary,
        ) {
            TbIcon(
                imageVector = Icons.Rounded.QuestionAnswer,
                contentDescription = null,
                modifier = Modifier
                    .padding(end = 6.dp)
                    .size(17.dp),
            )
            TbText(
                text = if (freeformOpen) "Hide question" else "Ask anything",
                style = TbTheme.typography.button,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

@Composable
private fun AmaMessageRow(
    message: AmaMessage,
    onOpenUsageSource: (Long?, Long) -> Unit,
) {
    val isUser = message.role == AmaMessageRole.User
    if (isUser) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.End,
        ) {
            TbSurface(
                modifier = Modifier.widthIn(max = 760.dp),
                shape = RoundedCornerShape(TbTheme.radii.card),
                color = TbTheme.colors.accent,
                contentColor = TbTheme.colors.accentText,
            ) {
                AmaMarkdownText(
                    modifier = Modifier.padding(14.dp),
                    content = message.content,
                    isUser = true,
                )
            }
        }
        return
    }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .widthIn(max = 820.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        TbSurface(
            modifier = Modifier.widthIn(max = 760.dp),
            shape = RoundedCornerShape(TbTheme.radii.card),
            color = TbTheme.colors.surface,
            contentColor = TbTheme.colors.text,
            border = BorderStroke(Dp.Hairline, TbTheme.colors.separator),
        ) {
            Column(
                modifier = Modifier.padding(14.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                AmaMarkdownText(
                    content = message.content,
                    isUser = false,
                )
                message.model?.let { model ->
                    TbText(
                        text = model,
                        style = TbTheme.typography.caption,
                        color = TbTheme.colors.tertiaryText,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
        message.artifacts.forEach { artifact ->
            when (artifact) {
                is AmaAppUsageChart -> AmaAppUsageChartCard(artifact)
                is AmaUsageTimeline -> AmaUsageTimelineCard(artifact, onOpenUsageSource)
                is AmaHabitSummary -> AmaHabitSummaryCard(artifact, onOpenUsageSource)
                is AmaUsageComparison -> AmaUsageComparisonCard(artifact)
            }
        }
        if (message.sources.isNotEmpty()) {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                message.sources.take(3).forEach { source ->
                    AmaSourceCard(
                        source = source,
                        onClick = {
                            onOpenUsageSource(source.startedAtEpochMillis, source.transitionEventId)
                        },
                    )
                }
            }
        }
    }
}

@Composable
private fun AmaMarkdownText(
    content: String,
    isUser: Boolean,
    modifier: Modifier = Modifier,
) {
    val colors = TbTheme.colors
    val typography = TbTheme.typography
    val textColor = if (isUser) colors.accentText else colors.text
    val secondaryTextColor = if (isUser) colors.accentText.copy(alpha = 0.74f) else colors.secondaryText
    val codeBackground = if (isUser) colors.accentText.copy(alpha = 0.12f) else colors.controlFill
    val linkColor = if (isUser) colors.accentText else colors.accent
    val body = typography.body.copy(color = textColor)
    val small = typography.bodySmall.copy(color = textColor)

    Markdown(
        content = content,
        modifier = modifier,
        colors = DefaultMarkdownColors(
            text = textColor,
            codeBackground = codeBackground,
            inlineCodeBackground = codeBackground,
            dividerColor = if (isUser) colors.accentText.copy(alpha = 0.18f) else colors.separator,
            tableBackground = if (isUser) colors.accentText.copy(alpha = 0.08f) else colors.groupedSurface,
        ),
        typography = DefaultMarkdownTypography(
            h1 = typography.title.copy(color = textColor),
            h2 = typography.title2.copy(color = textColor),
            h3 = typography.headline.copy(color = textColor),
            h4 = typography.label.copy(color = textColor),
            h5 = typography.label.copy(color = textColor),
            h6 = typography.label.copy(color = textColor),
            text = body,
            code = small.copy(fontFamily = FontFamily.Monospace),
            inlineCode = body.copy(fontFamily = FontFamily.Monospace),
            quote = body.copy(color = secondaryTextColor, fontStyle = FontStyle.Italic),
            paragraph = body,
            ordered = body,
            bullet = body,
            list = body,
            textLink = TextLinkStyles(
                style = SpanStyle(
                    color = linkColor,
                    fontWeight = FontWeight.SemiBold,
                    textDecoration = TextDecoration.Underline,
                ),
            ),
            table = small,
        ),
        padding = markdownPadding(
            block = 2.dp,
            list = 2.dp,
            listItemTop = 2.dp,
            listItemBottom = 2.dp,
            listIndent = 12.dp,
            codeBlock = PaddingValues(8.dp),
            blockQuote = PaddingValues(horizontal = 10.dp),
            blockQuoteText = PaddingValues(vertical = 3.dp),
            blockQuoteBar = PaddingValues.Absolute(left = 0.dp, top = 2.dp, right = 6.dp, bottom = 2.dp),
        ),
        dimens = markdownDimens(
            codeBackgroundCornerSize = TbTheme.radii.control,
            tableCellPadding = 8.dp,
            tableCornerSize = TbTheme.radii.control,
        ),
        retainState = true,
        loading = {
            TbText(
                modifier = it,
                text = content,
                style = body,
                color = textColor,
            )
        },
        error = {
            TbText(
                modifier = it,
                text = content,
                style = body,
                color = textColor,
            )
        },
    )
}

@Composable
private fun AmaAppUsageChartCard(chart: AmaAppUsageChart) {
    val maxSeconds = chart.buckets.maxOfOrNull { it.durationSeconds }?.coerceAtLeast(1) ?: 1
    TbSurface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(TbTheme.radii.card),
        color = TbTheme.colors.groupedSurface,
        border = BorderStroke(Dp.Hairline, TbTheme.colors.separator),
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.Top,
            ) {
                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(3.dp),
                ) {
                    TbText(
                        text = "Most Used Apps",
                        style = TbTheme.typography.label,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    TbText(
                        text = chart.periodLabel.ifBlank { "Selected period" },
                        style = TbTheme.typography.caption,
                        color = TbTheme.colors.secondaryText,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                TbText(
                    text = "${formatUsageChartDuration(chart.totalDurationSeconds)} captured",
                    style = TbTheme.typography.caption,
                    color = TbTheme.colors.tertiaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }

            if (chart.buckets.isEmpty()) {
                TbText(
                    text = "No app usage found",
                    style = TbTheme.typography.bodySmall,
                    color = TbTheme.colors.secondaryText,
                )
            } else {
                Column(verticalArrangement = Arrangement.spacedBy(9.dp)) {
                    chart.buckets.forEach { bucket ->
                        val fraction = (bucket.durationSeconds.toFloat() / maxSeconds.toFloat()).coerceIn(0.04f, 1f)
                        Column(verticalArrangement = Arrangement.spacedBy(5.dp)) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.spacedBy(10.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                TbText(
                                    modifier = Modifier.weight(1f),
                                    text = bucket.name.ifBlank { "Unknown application" },
                                    style = TbTheme.typography.bodySmall,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis,
                                )
                                TbText(
                                    text = "${formatUsageChartDuration(bucket.durationSeconds)} • ${formatUsageChartPercent(bucket.durationSeconds, chart.totalDurationSeconds)}",
                                    style = TbTheme.typography.caption,
                                    color = TbTheme.colors.secondaryText,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            }
                            Box(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .height(8.dp)
                                    .clip(RoundedCornerShape(4.dp))
                                    .background(TbTheme.colors.controlFill),
                            ) {
                                Box(
                                    modifier = Modifier
                                        .fillMaxWidth(fraction)
                                        .height(8.dp)
                                        .clip(RoundedCornerShape(4.dp))
                                        .background(TbTheme.colors.accent),
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun AmaUsageTimelineCard(
    timeline: AmaUsageTimeline,
    onOpenUsageSource: (Long?, Long) -> Unit,
) {
    AmaInsightCard(
        title = "Usage Timeline",
        subtitle = "${timeline.periodLabel.ifBlank { "Selected period" }} · ${timeline.totalEventCount} events",
        trailing = "${formatUsageChartDuration(timeline.totalDurationSeconds)} captured",
    ) {
        if (timeline.events.isEmpty()) {
            TbText(
                text = "No usage events found",
                style = TbTheme.typography.bodySmall,
                color = TbTheme.colors.secondaryText,
            )
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(7.dp)) {
                timeline.events.forEach { event ->
                    AmaTimelineEventRow(
                        event = event,
                        onClick = {
                            onOpenUsageSource(event.startedAtEpochMillis, event.transitionEventId)
                        },
                    )
                }
            }
            if (timeline.truncated) {
                TbText(
                    text = "Showing ${timeline.events.size} of ${timeline.totalEventCount}",
                    style = TbTheme.typography.caption,
                    color = TbTheme.colors.tertiaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

@Composable
private fun AmaHabitSummaryCard(
    summary: AmaHabitSummary,
    onOpenUsageSource: (Long?, Long) -> Unit,
) {
    AmaInsightCard(
        title = "Habit Summary",
        subtitle = summary.periodLabel.ifBlank { "Selected period" },
        trailing = "${summary.sessionCount} sessions",
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            AmaMetricPill("Captured", formatUsageChartDuration(summary.totalDurationSeconds), Modifier.weight(1f))
            AmaMetricPill("Switches", summary.contextSwitchCount.toString(), Modifier.weight(1f))
            AmaMetricPill("Average", formatUsageChartDuration(summary.averageSessionSeconds), Modifier.weight(1f))
        }
        summary.longestSession?.let { longest ->
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                TbText(
                    text = "Longest session",
                    style = TbTheme.typography.label,
                    color = TbTheme.colors.secondaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                AmaTimelineEventRow(
                    event = longest,
                    onClick = {
                        onOpenUsageSource(longest.startedAtEpochMillis, longest.transitionEventId)
                    },
                )
            }
        }
        if (summary.topSources.isNotEmpty()) {
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                TbText(
                    text = "Top sources",
                    style = TbTheme.typography.label,
                    color = TbTheme.colors.secondaryText,
                )
                summary.topSources.take(5).forEach { bucket ->
                    AmaSimpleBarRow(
                        label = bucket.name.ifBlank { "Unknown application" },
                        value = formatUsageChartDuration(bucket.durationSeconds),
                        fraction = bucket.durationSeconds.toFloat() /
                            summary.topSources.maxOf { it.durationSeconds }.coerceAtLeast(1).toFloat(),
                    )
                }
            }
        }
        if (summary.timeBuckets.isNotEmpty()) {
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                TbText(
                    text = "Time of day",
                    style = TbTheme.typography.label,
                    color = TbTheme.colors.secondaryText,
                )
                summary.timeBuckets.forEach { bucket ->
                    AmaSimpleBarRow(
                        label = bucket.label,
                        value = formatUsageChartDuration(bucket.durationSeconds),
                        fraction = bucket.durationSeconds.toFloat() /
                            summary.timeBuckets.maxOf { it.durationSeconds }.coerceAtLeast(1).toFloat(),
                    )
                }
            }
        }
    }
}

@Composable
private fun AmaUsageComparisonCard(comparison: AmaUsageComparison) {
    val direction = when {
        comparison.durationDeltaSeconds > 0 -> "Up"
        comparison.durationDeltaSeconds < 0 -> "Down"
        else -> "Flat"
    }
    AmaInsightCard(
        title = "Period Comparison",
        subtitle = "${comparison.currentPeriodLabel.ifBlank { "Current" }} vs ${comparison.baselinePeriodLabel.ifBlank { "Baseline" }}",
        trailing = "$direction ${formatUsageChartDuration(kotlin.math.abs(comparison.durationDeltaSeconds))}",
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            AmaMetricPill("Current", formatUsageChartDuration(comparison.currentTotalDurationSeconds), Modifier.weight(1f))
            AmaMetricPill("Baseline", formatUsageChartDuration(comparison.baselineTotalDurationSeconds), Modifier.weight(1f))
            AmaMetricPill("Delta", "${comparison.durationDeltaPercent.formatPercentDelta()}%", Modifier.weight(1f))
        }
        if (comparison.buckets.isEmpty()) {
            TbText(
                text = "No comparable usage found",
                style = TbTheme.typography.bodySmall,
                color = TbTheme.colors.secondaryText,
            )
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(7.dp)) {
                comparison.buckets.forEach { bucket ->
                    AmaComparisonRow(bucket)
                }
            }
        }
    }
}

@Composable
private fun AmaInsightCard(
    title: String,
    subtitle: String,
    trailing: String,
    content: @Composable () -> Unit,
) {
    TbSurface(
        modifier = Modifier.widthIn(max = 820.dp),
        shape = RoundedCornerShape(TbTheme.radii.card),
        color = TbTheme.colors.groupedSurface,
        border = BorderStroke(Dp.Hairline, TbTheme.colors.separator),
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.Top,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    TbText(
                        text = title,
                        style = TbTheme.typography.label,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    TbText(
                        text = subtitle,
                        style = TbTheme.typography.caption,
                        color = TbTheme.colors.secondaryText,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                TbText(
                    text = trailing,
                    style = TbTheme.typography.caption,
                    color = TbTheme.colors.tertiaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            content()
        }
    }
}

@Composable
private fun AmaTimelineEventRow(
    event: AmaUsageTimelineEvent,
    onClick: () -> Unit,
) {
    val canOpen = event.startedAtEpochMillis != null
    TbSurface(
        modifier = Modifier
            .fillMaxWidth()
            .then(if (canOpen) Modifier.clickable(onClick = onClick) else Modifier),
        shape = RoundedCornerShape(TbTheme.radii.control),
        color = TbTheme.colors.surface,
        border = BorderStroke(Dp.Hairline, TbTheme.colors.separator),
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TbBadge(
                label = event.sourceType.ifBlank { "usage" },
                color = TbTheme.colors.text,
                background = TbTheme.colors.accentSubtle,
            )
            Column(modifier = Modifier.weight(1f)) {
                TbText(
                    text = event.title.ifBlank { "Usage" },
                    style = TbTheme.typography.bodySmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbText(
                    text = buildList {
                        event.startedAtEpochMillis?.let { add(formatEventEpochTime(it)) }
                        add(event.sourceName.ifBlank { "Application" })
                        if (event.urlHost.isNotBlank()) add(event.urlHost)
                    }.joinToString(" · "),
                    style = TbTheme.typography.caption,
                    color = TbTheme.colors.secondaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            TbText(
                text = formatUsageChartDuration(event.durationSeconds),
                style = TbTheme.typography.caption,
                color = TbTheme.colors.secondaryText,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

@Composable
private fun AmaMetricPill(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
) {
    TbSurface(
        modifier = modifier,
        shape = RoundedCornerShape(TbTheme.radii.control),
        color = TbTheme.colors.surface,
        border = BorderStroke(Dp.Hairline, TbTheme.colors.separator),
    ) {
        Column(modifier = Modifier.padding(horizontal = 10.dp, vertical = 8.dp)) {
            TbText(
                text = label,
                style = TbTheme.typography.caption,
                color = TbTheme.colors.secondaryText,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            TbText(
                text = value,
                style = TbTheme.typography.label,
                color = TbTheme.colors.text,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

@Composable
private fun AmaSimpleBarRow(
    label: String,
    value: String,
    fraction: Float,
) {
    Column(verticalArrangement = Arrangement.spacedBy(5.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TbText(
                modifier = Modifier.weight(1f),
                text = label,
                style = TbTheme.typography.bodySmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            TbText(
                text = value,
                style = TbTheme.typography.caption,
                color = TbTheme.colors.secondaryText,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(6.dp)
                .clip(RoundedCornerShape(3.dp))
                .background(TbTheme.colors.controlFill),
        ) {
            Box(
                modifier = Modifier
                    .fillMaxWidth(fraction.coerceIn(0.04f, 1f))
                    .height(6.dp)
                    .clip(RoundedCornerShape(3.dp))
                    .background(TbTheme.colors.accent),
            )
        }
    }
}

@Composable
private fun AmaComparisonRow(bucket: com.timeboxxing.domain.model.AmaUsageComparisonBucket) {
    val maxSeconds = maxOf(bucket.currentDurationSeconds, bucket.baselineDurationSeconds).coerceAtLeast(1)
    Column(verticalArrangement = Arrangement.spacedBy(5.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TbText(
                modifier = Modifier.weight(1f),
                text = bucket.name.ifBlank { "Unknown application" },
                style = TbTheme.typography.bodySmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            TbText(
                text = "${formatUsageChartDuration(bucket.currentDurationSeconds)} / ${formatUsageChartDuration(bucket.baselineDurationSeconds)}",
                style = TbTheme.typography.caption,
                color = TbTheme.colors.secondaryText,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            AmaTinyBar(
                fraction = bucket.currentDurationSeconds.toFloat() / maxSeconds.toFloat(),
                color = TbTheme.colors.accent,
                modifier = Modifier.weight(1f),
            )
            AmaTinyBar(
                fraction = bucket.baselineDurationSeconds.toFloat() / maxSeconds.toFloat(),
                color = TbTheme.colors.secondaryText,
                modifier = Modifier.weight(1f),
            )
        }
    }
}

@Composable
private fun AmaTinyBar(
    fraction: Float,
    color: Color,
    modifier: Modifier = Modifier,
) {
    Box(
        modifier = modifier
            .height(6.dp)
            .clip(RoundedCornerShape(3.dp))
            .background(TbTheme.colors.controlFill),
    ) {
        Box(
            modifier = Modifier
                .fillMaxWidth(fraction.coerceIn(0.04f, 1f))
                .height(6.dp)
                .clip(RoundedCornerShape(3.dp))
                .background(color),
        )
    }
}

@Composable
private fun AmaSourceCard(
    source: AmaSource,
    onClick: () -> Unit,
) {
    val canOpen = source.startedAtEpochMillis != null
    TbSurface(
        modifier = Modifier
            .fillMaxWidth()
            .then(if (canOpen) Modifier.clickable(onClick = onClick) else Modifier),
        shape = RoundedCornerShape(TbTheme.radii.card),
        color = TbTheme.colors.groupedSurface,
        border = BorderStroke(Dp.Hairline, TbTheme.colors.separator),
    ) {
        Column(
            modifier = Modifier.padding(10.dp),
            verticalArrangement = Arrangement.spacedBy(5.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                TbBadge(
                    label = source.sourceLabel(),
                    color = TbTheme.colors.text,
                    background = TbTheme.colors.accentSubtle,
                )
                TbText(
                    modifier = Modifier.weight(1f),
                    text = "Distance ${source.distance.formatDistance()}",
                    style = TbTheme.typography.caption,
                    color = TbTheme.colors.tertiaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            TbText(
                text = source.content.lineSequence().take(4).joinToString(" · "),
                style = TbTheme.typography.bodySmall,
                color = TbTheme.colors.secondaryText,
                maxLines = 3,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

private fun AmaSource.sourceLabel(): String =
    when {
        transitionEventId != 0L -> "#$transitionEventId"
        documentType.isNotBlank() -> documentType.replace('_', ' ')
        documentKey.isNotBlank() -> documentKey
        else -> "source"
    }

@Composable
private fun AmaLoadingRow() {
    Row(modifier = Modifier.fillMaxWidth()) {
        TbSurface(
            shape = RoundedCornerShape(TbTheme.radii.card),
            color = TbTheme.colors.surface,
            border = BorderStroke(Dp.Hairline, TbTheme.colors.separator),
        ) {
            TbText(
                modifier = Modifier.padding(horizontal = 14.dp, vertical = 11.dp),
                text = "Thinking...",
                style = TbTheme.typography.body,
                color = TbTheme.colors.secondaryText,
            )
        }
    }
}

@Composable
private fun AmaErrorBanner(error: String) {
    TbSurface(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 24.dp, vertical = 8.dp),
        shape = RoundedCornerShape(TbTheme.radii.card),
        color = TbTheme.colors.destructiveSubtle,
        contentColor = TbTheme.colors.destructive,
    ) {
        TbText(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 10.dp),
            text = error,
            style = TbTheme.typography.body,
            color = TbTheme.colors.destructive,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
private fun AmaInsightLauncher(
    state: TimeboxxingScreenState,
    selectedKind: AmaQueryKind,
    onSelectedKindChange: (AmaQueryKind) -> Unit,
    selectedPreset: AmaPeriodPreset,
    onSelectedPresetChange: (AmaPeriodPreset) -> Unit,
    customStartDate: CalendarDate,
    customEndDate: CalendarDate,
    onCustomStartDateChange: (CalendarDate) -> Unit,
    onCustomEndDateChange: (CalendarDate) -> Unit,
    includeIdle: Boolean,
    onIncludeIdleChange: (Boolean) -> Unit,
    limit: Int,
    limitOptions: List<Int>,
    onLimitChange: (Int) -> Unit,
    showRunAction: Boolean,
    canSubmit: Boolean,
    onSubmit: () -> Unit,
) {
    val resolvedWindow = remember(state.selectedDay, state.selectedCalendarDate, selectedPreset, customStartDate, customEndDate) {
        resolveAmaWindow(
            state = state,
            preset = selectedPreset,
            customStartDate = customStartDate,
            customEndDate = customEndDate,
        )
    }

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        AmaInsightHeader(
            selectedKind = selectedKind,
            periodLabel = resolvedWindow?.label ?: "Select a period",
            showRunAction = showRunAction,
            canSubmit = canSubmit,
            onSubmit = onSubmit,
        )

        AmaInsightKindGrid(
            selectedKind = selectedKind,
            onSelectedKindChange = onSelectedKindChange,
        )

        AmaPeriodSelector(
            selectedPreset = selectedPreset,
            onSelectedPresetChange = onSelectedPresetChange,
        )

        if (selectedPreset == AmaPeriodPreset.Custom) {
            AmaCustomDateSelector(
                customStartDate = customStartDate,
                customEndDate = customEndDate,
                onCustomStartDateChange = onCustomStartDateChange,
                onCustomEndDateChange = onCustomEndDateChange,
            )
        }

        AmaInsightControls(
            selectedKind = selectedKind,
            limit = limit,
            limitOptions = limitOptions,
            includeIdle = includeIdle,
            onLimitChange = onLimitChange,
            onIncludeIdleChange = onIncludeIdleChange,
        )
    }
}

@Composable
private fun AmaInsightHeader(
    selectedKind: AmaQueryKind,
    periodLabel: String,
    showRunAction: Boolean,
    canSubmit: Boolean,
    onSubmit: () -> Unit,
) {
    BoxWithConstraints(modifier = Modifier.fillMaxWidth()) {
        val titleContent: @Composable (Modifier) -> Unit = { modifier ->
            Row(
                modifier = modifier,
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                TbText(
                    modifier = Modifier.weight(1f),
                    text = "Insight",
                    style = TbTheme.typography.label,
                    color = TbTheme.colors.secondaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbText(
                    text = periodLabel,
                    style = TbTheme.typography.caption,
                    color = TbTheme.colors.tertiaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }

        when {
            !showRunAction -> {
                titleContent(Modifier.fillMaxWidth())
            }
            maxWidth < 560.dp -> {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    titleContent(Modifier.fillMaxWidth())
                    AmaRunInsightButton(
                        selectedKind = selectedKind,
                        canSubmit = canSubmit,
                        onSubmit = onSubmit,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            }
            else -> {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    titleContent(Modifier.weight(1f))
                    AmaRunInsightButton(
                        selectedKind = selectedKind,
                        canSubmit = canSubmit,
                        onSubmit = onSubmit,
                        modifier = Modifier.widthIn(min = 150.dp),
                    )
                }
            }
        }
    }
}

@Composable
private fun AmaInsightDock(
    state: TimeboxxingScreenState,
    loading: Boolean,
    selectedKind: AmaQueryKind,
    onSelectedKindChange: (AmaQueryKind) -> Unit,
    selectedPreset: AmaPeriodPreset,
    onSelectedPresetChange: (AmaPeriodPreset) -> Unit,
    customStartDate: CalendarDate,
    customEndDate: CalendarDate,
    onCustomStartDateChange: (CalendarDate) -> Unit,
    onCustomEndDateChange: (CalendarDate) -> Unit,
    includeIdle: Boolean,
    onIncludeIdleChange: (Boolean) -> Unit,
    limit: Int,
    limitOptions: List<Int>,
    onLimitChange: (Int) -> Unit,
    canSubmit: Boolean,
    onSubmit: () -> Unit,
) {
    TbSurface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(0.dp),
        color = TbTheme.colors.surface,
        contentColor = TbTheme.colors.text,
        border = BorderStroke(Dp.Hairline, TbTheme.colors.separator),
    ) {
        Column(
            modifier = Modifier.padding(horizontal = 24.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            AmaInsightLauncher(
                state = state,
                selectedKind = selectedKind,
                onSelectedKindChange = onSelectedKindChange,
                selectedPreset = selectedPreset,
                onSelectedPresetChange = onSelectedPresetChange,
                customStartDate = customStartDate,
                customEndDate = customEndDate,
                onCustomStartDateChange = onCustomStartDateChange,
                onCustomEndDateChange = onCustomEndDateChange,
                includeIdle = includeIdle,
                onIncludeIdleChange = onIncludeIdleChange,
                limit = limit,
                limitOptions = limitOptions,
                onLimitChange = onLimitChange,
                showRunAction = true,
                canSubmit = canSubmit && !loading,
                onSubmit = onSubmit,
            )
        }
    }
}

@Composable
private fun AmaInsightKindGrid(
    selectedKind: AmaQueryKind,
    onSelectedKindChange: (AmaQueryKind) -> Unit,
) {
    val kinds = listOf(
        AmaQueryKind.AppTotals,
        AmaQueryKind.Timeline,
        AmaQueryKind.Habits,
        AmaQueryKind.ComparePeriods,
    )
    BoxWithConstraints(modifier = Modifier.fillMaxWidth()) {
        if (maxWidth < 620.dp) {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                kinds.chunked(2).forEach { rowKinds ->
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        rowKinds.forEach { kind ->
                            AmaInsightKindTile(
                                kind = kind,
                                selected = selectedKind == kind,
                                onClick = { onSelectedKindChange(kind) },
                                modifier = Modifier.weight(1f),
                            )
                        }
                    }
                }
            }
        } else {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                kinds.forEach { kind ->
                    AmaInsightKindTile(
                        kind = kind,
                        selected = selectedKind == kind,
                        onClick = { onSelectedKindChange(kind) },
                        modifier = Modifier.weight(1f),
                    )
                }
            }
        }
    }
}

@Composable
private fun AmaInsightKindTile(
    kind: AmaQueryKind,
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    AmaSelectableSurface(
        modifier = modifier,
        selected = selected,
        onClick = onClick,
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 10.dp, vertical = 9.dp),
            horizontalArrangement = Arrangement.spacedBy(9.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TbIcon(
                imageVector = kind.icon,
                contentDescription = null,
                modifier = Modifier.size(17.dp),
            )
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(1.dp),
            ) {
                TbText(
                    text = kind.label,
                    style = TbTheme.typography.button,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbText(
                    text = kind.caption,
                    style = TbTheme.typography.caption,
                    color = if (selected) TbTheme.colors.accent else TbTheme.colors.secondaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

@Composable
private fun AmaPeriodSelector(
    selectedPreset: AmaPeriodPreset,
    onSelectedPresetChange: (AmaPeriodPreset) -> Unit,
) {
    BoxWithConstraints(modifier = Modifier.fillMaxWidth()) {
        val compact = maxWidth < 520.dp
        val content: @Composable (AmaPeriodPreset, String, Modifier) -> Unit = { preset, label, modifier ->
            AmaSelectablePill(
                label = label,
                selected = selectedPreset == preset,
                onClick = { onSelectedPresetChange(preset) },
                modifier = modifier,
            )
        }
        if (compact) {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                content(AmaPeriodPreset.SelectedDay, "Selected day", Modifier.fillMaxWidth())
                content(AmaPeriodPreset.PreviousDay, "Previous day", Modifier.fillMaxWidth())
                content(AmaPeriodPreset.Custom, "Custom", Modifier.fillMaxWidth())
            }
        } else {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                content(AmaPeriodPreset.SelectedDay, "Selected day", Modifier.weight(1f))
                content(AmaPeriodPreset.PreviousDay, "Previous day", Modifier.weight(1f))
                content(AmaPeriodPreset.Custom, "Custom", Modifier.weight(1f))
            }
        }
    }
}

@Composable
private fun AmaCustomDateSelector(
    customStartDate: CalendarDate,
    customEndDate: CalendarDate,
    onCustomStartDateChange: (CalendarDate) -> Unit,
    onCustomEndDateChange: (CalendarDate) -> Unit,
) {
    BoxWithConstraints(modifier = Modifier.fillMaxWidth()) {
        if (maxWidth < 520.dp) {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                CalendarDatePickerMenu(
                    selectedDate = customStartDate,
                    label = "From ${customStartDate.isoLabel()}",
                    onDateSelected = onCustomStartDateChange,
                    modifier = Modifier.fillMaxWidth(),
                )
                CalendarDatePickerMenu(
                    selectedDate = customEndDate,
                    label = "To ${customEndDate.isoLabel()}",
                    onDateSelected = onCustomEndDateChange,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        } else {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                CalendarDatePickerMenu(
                    selectedDate = customStartDate,
                    label = "From ${customStartDate.isoLabel()}",
                    onDateSelected = onCustomStartDateChange,
                    modifier = Modifier.weight(1f),
                )
                CalendarDatePickerMenu(
                    selectedDate = customEndDate,
                    label = "To ${customEndDate.isoLabel()}",
                    onDateSelected = onCustomEndDateChange,
                    modifier = Modifier.weight(1f),
                )
            }
        }
    }
}

@Composable
private fun AmaInsightControls(
    selectedKind: AmaQueryKind,
    limit: Int,
    limitOptions: List<Int>,
    includeIdle: Boolean,
    onLimitChange: (Int) -> Unit,
    onIncludeIdleChange: (Boolean) -> Unit,
) {
    BoxWithConstraints(modifier = Modifier.fillMaxWidth()) {
        if (maxWidth < 720.dp) {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                AmaLimitSelector(
                    label = selectedKind.limitLabel,
                    limit = limit,
                    limitOptions = limitOptions,
                    onLimitChange = onLimitChange,
                )
                AmaIdleToggle(
                    includeIdle = includeIdle,
                    onIncludeIdleChange = onIncludeIdleChange,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        } else {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.Bottom,
            ) {
                AmaLimitSelector(
                    label = selectedKind.limitLabel,
                    limit = limit,
                    limitOptions = limitOptions,
                    onLimitChange = onLimitChange,
                    modifier = Modifier.weight(1f),
                )
                AmaIdleToggle(
                    includeIdle = includeIdle,
                    onIncludeIdleChange = onIncludeIdleChange,
                    modifier = Modifier.widthIn(min = 260.dp),
                )
            }
        }
    }
}

@Composable
private fun AmaLimitSelector(
    label: String,
    limit: Int,
    limitOptions: List<Int>,
    onLimitChange: (Int) -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier,
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        TbText(
            text = label,
            style = TbTheme.typography.caption,
            color = TbTheme.colors.secondaryText,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        Row(horizontalArrangement = Arrangement.spacedBy(7.dp)) {
            limitOptions.forEach { option ->
                AmaSelectablePill(
                    label = option.toString(),
                    selected = limit == option,
                    onClick = { onLimitChange(option) },
                    modifier = Modifier.widthIn(min = 48.dp),
                )
            }
        }
    }
}

@Composable
private fun AmaIdleToggle(
    includeIdle: Boolean,
    onIncludeIdleChange: (Boolean) -> Unit,
    modifier: Modifier = Modifier,
) {
    AmaSelectableSurface(
        modifier = modifier,
        selected = includeIdle,
        onClick = { onIncludeIdleChange(!includeIdle) },
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 10.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(9.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TbCheckbox(
                checked = includeIdle,
                onCheckedChange = onIncludeIdleChange,
                accessibilityLabel = "Include idle time",
            )
            Column(modifier = Modifier.weight(1f)) {
                TbText(
                    text = "Include idle",
                    style = TbTheme.typography.button,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbText(
                    text = "Count idle sessions in totals",
                    style = TbTheme.typography.caption,
                    color = TbTheme.colors.secondaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

@Composable
private fun AmaRunInsightButton(
    selectedKind: AmaQueryKind,
    canSubmit: Boolean,
    onSubmit: () -> Unit,
    modifier: Modifier = Modifier,
) {
    TbButton(
        modifier = modifier,
        onClick = onSubmit,
        enabled = canSubmit,
    ) {
        TbIcon(
            imageVector = selectedKind.icon,
            contentDescription = null,
            modifier = Modifier
                .padding(end = 6.dp)
                .size(17.dp),
        )
        TbText(
            text = "Run ${selectedKind.label}",
            style = TbTheme.typography.button,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
private fun AmaSelectablePill(
    label: String,
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    AmaSelectableSurface(
        modifier = modifier,
        selected = selected,
        onClick = onClick,
    ) {
        TbText(
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 8.dp),
            text = label,
            style = TbTheme.typography.button,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
private fun AmaSelectableSurface(
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    content: @Composable () -> Unit,
) {
    val shape = RoundedCornerShape(TbTheme.radii.control)
    TbSurface(
        modifier = modifier
            .heightIn(min = 34.dp)
            .clip(shape)
            .clickable(onClick = onClick),
        shape = shape,
        color = if (selected) TbTheme.colors.accentSubtle else TbTheme.colors.controlFill,
        contentColor = if (selected) TbTheme.colors.accent else TbTheme.colors.text,
        border = BorderStroke(Dp.Hairline, if (selected) TbTheme.colors.accent else TbTheme.colors.separator),
    ) {
        content()
    }
}

private fun resolveAmaWindow(
    state: TimeboxxingScreenState,
    preset: AmaPeriodPreset,
    customStartDate: CalendarDate,
    customEndDate: CalendarDate,
): AmaResolvedWindow? =
    when (preset) {
        AmaPeriodPreset.SelectedDay -> {
            val date = state.selectedCalendarDate ?: return AmaResolvedWindow(
                window = AmaTimeWindow(state.selectedDay.startedAtEpochMillis, state.selectedDay.endedAtEpochMillis),
                label = state.dateLabel,
                startDate = customStartDate,
                endDate = customStartDate,
            )
            val day = usageDayForCalendarDate(date)
            AmaResolvedWindow(
                window = AmaTimeWindow(day.startedAtEpochMillis, day.endedAtEpochMillis),
                label = date.isoLabel(),
                startDate = date,
                endDate = date,
            )
        }
        AmaPeriodPreset.PreviousDay -> {
            val date = (state.selectedCalendarDate ?: return null).plusDays(-1)
            val day = usageDayForCalendarDate(date)
            AmaResolvedWindow(
                window = AmaTimeWindow(day.startedAtEpochMillis, day.endedAtEpochMillis),
                label = date.isoLabel(),
                startDate = date,
                endDate = date,
            )
        }
        AmaPeriodPreset.Custom -> {
            val start = minCalendarDate(customStartDate, customEndDate)
            val end = maxCalendarDate(customStartDate, customEndDate)
            val startDay = usageDayForCalendarDate(start)
            val endBoundary = usageDayForCalendarDate(end.plusDays(1))
            AmaResolvedWindow(
                window = AmaTimeWindow(startDay.startedAtEpochMillis, endBoundary.startedAtEpochMillis),
                label = if (start == end) start.isoLabel() else "${start.isoLabel()} to ${end.isoLabel()}",
                startDate = start,
                endDate = end,
            )
        }
    }

private fun comparisonBaselineWindow(current: AmaResolvedWindow): AmaResolvedWindow {
    val dayCount = daysBetweenInclusive(current.startDate, current.endDate)
    val baselineEnd = current.startDate.plusDays(-1)
    val baselineStart = baselineEnd.plusDays(-(dayCount - 1))
    val startDay = usageDayForCalendarDate(baselineStart)
    val endBoundary = usageDayForCalendarDate(baselineEnd.plusDays(1))
    return AmaResolvedWindow(
        window = AmaTimeWindow(startDay.startedAtEpochMillis, endBoundary.startedAtEpochMillis),
        label = if (baselineStart == baselineEnd) {
            baselineStart.isoLabel()
        } else {
            "${baselineStart.isoLabel()} to ${baselineEnd.isoLabel()}"
        },
        startDate = baselineStart,
        endDate = baselineEnd,
    )
}

private fun buildAmaStructuredQuery(
    state: TimeboxxingScreenState,
    kind: AmaQueryKind,
    preset: AmaPeriodPreset,
    customStartDate: CalendarDate,
    customEndDate: CalendarDate,
    limit: Int,
    includeIdle: Boolean,
): AmaStructuredQuery? {
    val current = resolveAmaWindow(
        state = state,
        preset = preset,
        customStartDate = customStartDate,
        customEndDate = customEndDate,
    ) ?: return null
    val baseline = current
        .takeIf { kind == AmaQueryKind.ComparePeriods }
        ?.let { comparisonBaselineWindow(it) }
    return AmaStructuredQuery(
        kind = kind,
        window = current.window,
        baselineWindow = baseline?.window,
        limit = limit,
        includeIdle = includeIdle,
        periodLabel = current.label,
        baselinePeriodLabel = baseline?.label.orEmpty(),
    )
}

private fun amaDefaultCalendarDate(state: TimeboxxingScreenState): CalendarDate =
    state.selectedCalendarDate
        ?: state.selectedDay.calendarDate
        ?: calendarDateForEpochMillis(state.selectedDay.startedAtEpochMillis)

private val AmaQueryKind.icon: ImageVector
    get() = when (this) {
        AmaQueryKind.AppTotals -> Icons.Rounded.BarChart
        AmaQueryKind.Timeline -> Icons.AutoMirrored.Rounded.ListAlt
        AmaQueryKind.Habits -> Icons.Rounded.Insights
        AmaQueryKind.ComparePeriods -> Icons.AutoMirrored.Rounded.CompareArrows
    }

private val AmaQueryKind.label: String
    get() = when (this) {
        AmaQueryKind.AppTotals -> "Apps"
        AmaQueryKind.Timeline -> "Timeline"
        AmaQueryKind.Habits -> "Habits"
        AmaQueryKind.ComparePeriods -> "Compare"
    }

private val AmaQueryKind.caption: String
    get() = when (this) {
        AmaQueryKind.AppTotals -> "Totals by app"
        AmaQueryKind.Timeline -> "Exact events"
        AmaQueryKind.Habits -> "Patterns"
        AmaQueryKind.ComparePeriods -> "Period deltas"
    }

private val AmaQueryKind.limitLabel: String
    get() = when (this) {
        AmaQueryKind.Timeline -> "Rows"
        else -> "Apps"
    }

private fun amaLimitOptionsFor(kind: AmaQueryKind): List<Int> =
    if (kind == AmaQueryKind.Timeline) {
        listOf(20, 50, 100)
    } else {
        listOf(5, 10, 20)
    }

private fun CalendarDate.isoLabel(): String =
    "${year.toString().padStart(4, '0')}-${month.toString().padStart(2, '0')}-${dayOfMonth.toString().padStart(2, '0')}"

private fun minCalendarDate(first: CalendarDate, second: CalendarDate): CalendarDate =
    if (first <= second) first else second

private fun maxCalendarDate(first: CalendarDate, second: CalendarDate): CalendarDate =
    if (first >= second) first else second

private fun daysBetweenInclusive(start: CalendarDate, end: CalendarDate): Int {
    var cursor = start
    var count = 1
    while (cursor < end) {
        cursor = cursor.plusDays(1)
        count += 1
    }
    return count
}

@Composable
private fun AmaFreeformComposer(
    value: String,
    loading: Boolean,
    onValueChange: (String) -> Unit,
    onSubmit: () -> Unit,
    framed: Boolean,
    singleLine: Boolean,
) {
    var lastEscapePress by remember { mutableStateOf<TimeMark?>(null) }
    val canSubmit = value.isNotBlank() && !loading
    val rowModifier = if (framed) {
        Modifier
            .fillMaxWidth()
            .background(TbTheme.colors.surface)
            .padding(16.dp)
    } else {
        Modifier.fillMaxWidth()
    }

    Row(
        modifier = rowModifier,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.Bottom,
    ) {
        TbTextField(
            modifier = Modifier.weight(1f),
            value = value,
            onValueChange = onValueChange,
            label = if (framed) null else "Ask anything",
            singleLine = singleLine,
            minLines = if (singleLine) 1 else 2,
            maxLines = if (singleLine) 1 else 6,
            inputModifier = Modifier.onPreviewKeyEvent { event ->
                if (event.type != KeyEventType.KeyDown) {
                    return@onPreviewKeyEvent false
                }

                when (event.key) {
                    Key.Enter -> {
                        lastEscapePress = null
                        if (event.isShiftPressed) {
                            false
                        } else {
                            if (canSubmit) {
                                onSubmit()
                            }
                            true
                        }
                    }

                    Key.Escape -> {
                        val isSecondEscape = lastEscapePress
                            ?.elapsedNow()
                            ?.let { it <= AmaComposerDoubleEscapeWindow }
                            ?: false
                        if (isSecondEscape) {
                            if (value.isNotEmpty()) {
                                onValueChange("")
                            }
                            lastEscapePress = null
                        } else {
                            lastEscapePress = TimeSource.Monotonic.markNow()
                        }
                        true
                    }

                    else -> {
                        lastEscapePress = null
                        false
                    }
                }
            },
        )
        TbIconButton(
            icon = Icons.AutoMirrored.Rounded.Send,
            contentDescription = "Send AMA question",
            onClick = onSubmit,
            enabled = canSubmit,
            variant = TbButtonVariant.Primary,
        )
    }
}

private val OverviewPane.label: String
    get() = when (this) {
        OverviewPane.UsageSchedule -> "Usage schedule"
        OverviewPane.TimeEntries -> "Entries"
        OverviewPane.Projects -> "Projects"
    }

private val OverviewPane.icon: ImageVector
    get() = when (this) {
        OverviewPane.UsageSchedule -> Icons.Rounded.Schedule
        OverviewPane.TimeEntries -> Icons.Rounded.Edit
        OverviewPane.Projects -> Icons.Rounded.Folder
    }

private val TimeboxxingSection.label: String
    get() = when (this) {
        TimeboxxingSection.Overview -> "Overview"
        TimeboxxingSection.Ama -> "AMA"
        TimeboxxingSection.Diagnostics -> "Diagnostics"
        TimeboxxingSection.Settings -> "Settings"
    }

private val TimeboxxingSection.icon: ImageVector
    get() = when (this) {
        TimeboxxingSection.Overview -> Icons.Rounded.Dashboard
        TimeboxxingSection.Ama -> Icons.Rounded.QuestionAnswer
        TimeboxxingSection.Diagnostics -> Icons.Rounded.BugReport
        TimeboxxingSection.Settings -> Icons.Rounded.Settings
    }

private fun formatUsageChartDuration(seconds: Long): String {
    val minutes = (seconds.coerceAtLeast(0) / 60).toInt()
    if (minutes <= 0) {
        return "<1m"
    }
    val hours = minutes / 60
    val remainder = minutes % 60
    return when {
        hours == 0 -> "${remainder}m"
        remainder == 0 -> "${hours}h"
        else -> "${hours}h ${remainder}m"
    }
}

private fun formatUsageChartPercent(seconds: Long, totalSeconds: Long): String {
    if (seconds <= 0 || totalSeconds <= 0) {
        return "0%"
    }
    return "${((seconds.toDouble() / totalSeconds.toDouble()) * 100.0).roundToInt()}%"
}

private fun formatEventEpochTime(epochMillis: Long): String {
    val date = calendarDateForEpochMillis(epochMillis)
    val day = usageDayForCalendarDate(date)
    val minute = ((epochMillis - day.startedAtEpochMillis) / 60_000L)
        .toInt()
        .coerceIn(0, 24 * 60 - 1)
    return formatClockTime(minute)
}

private fun Double.formatPercentDelta(): String {
    val rounded = (this * 10.0).roundToInt() / 10.0
    return if (rounded % 1.0 == 0.0) {
        rounded.toInt().toString()
    } else {
        rounded.toString()
    }
}

private fun Double.formatDistance(): String =
    ((this * 1000.0).roundToInt() / 1000.0).toString()
