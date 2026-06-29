package com.timeboxxing.app.ui

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.animateContentSize
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.animation.expandHorizontally
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.shrinkHorizontally
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
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
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.TransformOrigin
import androidx.compose.ui.graphics.drawscope.clipPath
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.isShiftPressed
import androidx.compose.ui.input.key.key
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.boundsInRoot
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextLinkStyles
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.zIndex
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowLeft
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.rounded.Send
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.Dashboard
import androidx.compose.material.icons.rounded.BugReport
import androidx.compose.material.icons.rounded.Edit
import androidx.compose.material.icons.rounded.ExpandMore
import androidx.compose.material.icons.rounded.Folder
import androidx.compose.material.icons.rounded.Minimize
import androidx.compose.material.icons.rounded.QuestionAnswer
import androidx.compose.material.icons.rounded.Schedule
import androidx.compose.material.icons.rounded.Settings
import com.timeboxxing.app.data.currentCalendarDate
import com.timeboxxing.app.model.AmaAppUsageChart
import com.timeboxxing.app.model.AmaMessage
import com.timeboxxing.app.model.AmaMessageRole
import com.timeboxxing.app.model.AmaIndexState
import com.timeboxxing.app.model.AmaIndexStatus
import com.timeboxxing.app.model.AmaSource
import com.timeboxxing.app.model.CalendarDate
import com.timeboxxing.app.model.DiagnosticsLogLine
import com.timeboxxing.app.model.EntryMode
import com.timeboxxing.app.model.WeekdayShortLabels
import com.timeboxxing.app.model.calendarMonthGrid
import com.timeboxxing.app.model.monthYearLabel
import com.timeboxxing.app.model.plusMonths
import com.timeboxxing.app.model.startOfMonth
import com.timeboxxing.app.state.TimeboxxingAction
import com.timeboxxing.app.state.TimeboxxingScreenState
import com.timeboxxing.app.state.TimeboxxingSection
import com.mikepenz.markdown.compose.Markdown
import com.mikepenz.markdown.model.DefaultMarkdownColors
import com.mikepenz.markdown.model.DefaultMarkdownTypography
import com.mikepenz.markdown.model.markdownDimens
import com.mikepenz.markdown.model.markdownPadding
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

private enum class OverviewPane {
    UsageSchedule,
    TimeEntries,
    Projects,
}

private enum class OverviewPaneMotionDirection {
    Minimize,
    Restore,
}

private data class PendingOverviewPaneGenieCue(
    val id: Int,
    val pane: OverviewPane,
    val direction: OverviewPaneMotionDirection,
    val sourceBounds: Rect,
)

private data class OverviewPaneGenieCue(
    val id: Int,
    val pane: OverviewPane,
    val direction: OverviewPaneMotionDirection,
    val startBounds: Rect,
    val endBounds: Rect,
)

private val WideWorkspaceMinContentWidth = 1120.dp
private val ProjectPaneWidth = 280.dp
private const val OverviewPaneAnimationMillis = 220
private const val OverviewPaneGenieMillis = 820
private const val OverviewPaneTrayPulseMillis = 620L
private const val MinimizedPaneWeight = 0.0001f
private const val OverviewPaneGenieTrayScale = 0.86f
private val OverviewPaneGenieEasing = CubicBezierEasing(0.2f, 0f, 0f, 1f)
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

                        state.notice?.let { notice ->
                            NoticeBanner(
                                notice = notice,
                                actionLabel = noticeActionLabel,
                                onAction = onNoticeAction,
                                onDismiss = { onAction(TimeboxxingAction.DismissNotice) },
                            )
                        }

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
            TbText(
                modifier = Modifier.padding(horizontal = 8.dp, vertical = 8.dp),
                text = "Timeboxxing",
                style = TbTheme.typography.title2,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
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
    val selectedDate = state.selectedCalendarDate
    val todayDate = currentCalendarDate()
    var expanded by remember { mutableStateOf(false) }
    var visibleMonth by remember(selectedDate) {
        mutableStateOf((selectedDate ?: CalendarDate(1970, 1, 1)).startOfMonth())
    }

    TbMenu(
        expanded = expanded,
        onExpandedChange = { expanded = it },
        anchor = {
            TbButton(
                modifier = Modifier.widthIn(min = 300.dp, max = 360.dp),
                onClick = { expanded = true },
                variant = TbButtonVariant.Secondary,
                contentPadding = PaddingValues(horizontal = 14.dp, vertical = 9.dp),
            ) {
                TbText(
                    text = state.dateLabel,
                    style = TbTheme.typography.title2,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbIcon(
                    imageVector = Icons.Rounded.ExpandMore,
                    contentDescription = null,
                    modifier = Modifier
                        .padding(start = 8.dp)
                        .size(16.dp),
                )
            }
        },
        panelContent = {
            CalendarPanel(
                visibleMonth = visibleMonth,
                selectedDate = selectedDate,
                todayDate = todayDate,
                onPreviousMonth = { visibleMonth = visibleMonth.plusMonths(-1) },
                onNextMonth = { visibleMonth = visibleMonth.plusMonths(1) },
                onDateSelected = { date ->
                    expanded = false
                    onDateSelected(date)
                },
            )
        },
    )
}

@Composable
private fun CalendarPanel(
    visibleMonth: CalendarDate,
    selectedDate: CalendarDate?,
    todayDate: CalendarDate,
    onPreviousMonth: () -> Unit,
    onNextMonth: () -> Unit,
    onDateSelected: (CalendarDate) -> Unit,
) {
    Column(
        modifier = Modifier
            .width(286.dp)
            .padding(6.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TbIconButton(
                icon = Icons.AutoMirrored.Rounded.KeyboardArrowLeft,
                contentDescription = "Previous month",
                onClick = onPreviousMonth,
                variant = TbButtonVariant.Ghost,
            )
            TbText(
                text = visibleMonth.monthYearLabel(),
                style = TbTheme.typography.headline,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            TbIconButton(
                icon = Icons.AutoMirrored.Rounded.KeyboardArrowRight,
                contentDescription = "Next month",
                onClick = onNextMonth,
                variant = TbButtonVariant.Ghost,
            )
        }

        CalendarWeekdayRow()

        calendarMonthGrid(visibleMonth).chunked(7).forEach { week ->
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                week.forEach { date ->
                    CalendarDayButton(
                        date = date,
                        visibleMonth = visibleMonth,
                        selected = date == selectedDate,
                        today = date == todayDate,
                        onClick = { onDateSelected(date) },
                    )
                }
            }
        }
    }
}

@Composable
private fun CalendarWeekdayRow() {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        WeekdayShortLabels.forEach { label ->
            Box(
                modifier = Modifier.size(34.dp),
                contentAlignment = Alignment.Center,
            ) {
                TbText(
                    text = label,
                    style = TbTheme.typography.caption.copy(textAlign = TextAlign.Center),
                    color = TbTheme.colors.tertiaryText,
                    maxLines = 1,
                )
            }
        }
    }
}

@Composable
private fun CalendarDayButton(
    date: CalendarDate,
    visibleMonth: CalendarDate,
    selected: Boolean,
    today: Boolean,
    onClick: () -> Unit,
) {
    val colors = TbTheme.colors
    val inVisibleMonth = date.year == visibleMonth.year && date.month == visibleMonth.month
    val interactionSource = remember { MutableInteractionSource() }
    val hovered by interactionSource.collectIsHoveredAsState()
    val pressed by interactionSource.collectIsPressedAsState()
    val shape = RoundedCornerShape(TbTheme.radii.control)
    val backgroundColor = when {
        selected -> colors.accent
        today -> colors.accentSubtle
        pressed -> colors.controlFillHover
        hovered -> colors.controlFill
        else -> Color.Transparent
    }
    val border = if (today && !selected) BorderStroke(1.dp, colors.accent) else null
    val textColor = when {
        selected -> colors.accentText
        today -> colors.text
        inVisibleMonth -> colors.text
        else -> colors.tertiaryText
    }

    TbSurface(
        modifier = Modifier
            .size(34.dp)
            .clickable(
                interactionSource = interactionSource,
                indication = null,
                onClick = onClick,
            ),
        color = backgroundColor,
        shape = shape,
        border = border,
        contentColor = textColor,
        contentAlignment = Alignment.Center,
    ) {
        TbText(
            text = date.dayOfMonth.toString(),
            style = TbTheme.typography.label.copy(
                textAlign = TextAlign.Center,
                fontWeight = if (today || selected) FontWeight.SemiBold else FontWeight.Medium,
            ),
            color = textColor,
            maxLines = 1,
        )
    }
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
private fun NoticeBanner(
    notice: String,
    actionLabel: String?,
    onAction: (() -> Unit)?,
    onDismiss: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(TbTheme.colors.accentSubtle)
            .padding(horizontal = 24.dp, vertical = 10.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
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
    var pendingGenieCue by remember { mutableStateOf<PendingOverviewPaneGenieCue?>(null) }
    var activeGenieCue by remember { mutableStateOf<OverviewPaneGenieCue?>(null) }
    var pulsingTrayPane by remember { mutableStateOf<OverviewPane?>(null) }
    var nextGenieCueId by remember { mutableStateOf(0) }
    var workspaceBounds by remember { mutableStateOf<Rect?>(null) }
    var paneBounds by remember { mutableStateOf(emptyMap<OverviewPane, Rect>()) }
    var trayIconBounds by remember { mutableStateOf(emptyMap<OverviewPane, Rect>()) }
    val visiblePanes = OverviewPane.entries.filterNot { it in minimizedPanes }
    val canMinimize = visiblePanes.size > 1
    val usageVisible = OverviewPane.UsageSchedule !in minimizedPanes
    val entriesVisible = OverviewPane.TimeEntries !in minimizedPanes
    val projectsVisible = OverviewPane.Projects !in minimizedPanes
    val usageWeight by animateFloatAsState(
        targetValue = if (usageVisible) 0.98f else MinimizedPaneWeight,
        animationSpec = tween(OverviewPaneAnimationMillis),
        label = "usage-pane-weight",
    )
    val entriesWeight by animateFloatAsState(
        targetValue = if (entriesVisible) 1.05f else MinimizedPaneWeight,
        animationSpec = tween(OverviewPaneAnimationMillis),
        label = "entries-pane-weight",
    )
    val projectsWidth by animateDpAsState(
        targetValue = if (projectsVisible) ProjectPaneWidth else 0.dp,
        animationSpec = tween(OverviewPaneAnimationMillis),
        label = "projects-pane-width",
    )

    fun nextCueId(): Int {
        val id = nextGenieCueId
        nextGenieCueId += 1
        return id
    }

    fun updatePaneBounds(pane: OverviewPane, bounds: Rect) {
        if (paneBounds[pane] != bounds) {
            paneBounds = paneBounds + (pane to bounds)
        }
    }

    fun updateTrayIconBounds(pane: OverviewPane, bounds: Rect) {
        if (trayIconBounds[pane] != bounds) {
            trayIconBounds = trayIconBounds + (pane to bounds)
        }
    }

    fun minimizePane(pane: OverviewPane) {
        if (!canMinimize) {
            return
        }
        val sourceBounds = paneBounds[pane]
        if (sourceBounds != null) {
            pendingGenieCue = PendingOverviewPaneGenieCue(
                id = nextCueId(),
                pane = pane,
                direction = OverviewPaneMotionDirection.Minimize,
                sourceBounds = sourceBounds,
            )
        }
        onMinimizePane(pane)
        pulsingTrayPane = pane
    }

    fun restorePane(pane: OverviewPane) {
        val sourceBounds = trayIconBounds[pane]
        if (sourceBounds != null) {
            pendingGenieCue = PendingOverviewPaneGenieCue(
                id = nextCueId(),
                pane = pane,
                direction = OverviewPaneMotionDirection.Restore,
                sourceBounds = sourceBounds,
            )
        }
        onRestorePane(pane)
        pulsingTrayPane = pane
    }

    LaunchedEffect(pendingGenieCue, paneBounds, trayIconBounds, workspaceBounds) {
        val pending = pendingGenieCue ?: return@LaunchedEffect
        val workspace = workspaceBounds ?: return@LaunchedEffect
        val targetBounds = when (pending.direction) {
            OverviewPaneMotionDirection.Minimize -> trayIconBounds[pending.pane]
            OverviewPaneMotionDirection.Restore -> paneBounds[pending.pane]
        } ?: return@LaunchedEffect
        val localSourceBounds = pending.sourceBounds.toLocalBounds(workspace)
        val localTargetBounds = targetBounds.toLocalBounds(workspace)

        activeGenieCue = OverviewPaneGenieCue(
            id = pending.id,
            pane = pending.pane,
            direction = pending.direction,
            startBounds = when (pending.direction) {
                OverviewPaneMotionDirection.Minimize -> localSourceBounds
                OverviewPaneMotionDirection.Restore -> localSourceBounds.centeredScale(OverviewPaneGenieTrayScale)
            },
            endBounds = when (pending.direction) {
                OverviewPaneMotionDirection.Minimize -> localTargetBounds.centeredScale(OverviewPaneGenieTrayScale)
                OverviewPaneMotionDirection.Restore -> localTargetBounds
            },
        )
        pendingGenieCue = null
    }

    LaunchedEffect(pendingGenieCue?.id) {
        val pending = pendingGenieCue ?: return@LaunchedEffect
        delay(OverviewPaneAnimationMillis.toLong() + 140L)
        if (pendingGenieCue?.id == pending.id) {
            pendingGenieCue = null
        }
    }

    LaunchedEffect(activeGenieCue?.id) {
        val active = activeGenieCue ?: return@LaunchedEffect
        delay(OverviewPaneGenieMillis.toLong())
        if (activeGenieCue?.id == active.id) {
            activeGenieCue = null
        }
    }

    LaunchedEffect(pulsingTrayPane) {
        if (pulsingTrayPane != null) {
            delay(OverviewPaneTrayPulseMillis)
            pulsingTrayPane = null
        }
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .onGloballyPositioned { coordinates ->
                workspaceBounds = coordinates.boundsInRoot()
            },
    ) {
        Row(
            modifier = Modifier
                .fillMaxSize()
                .animateContentSize(animationSpec = tween(OverviewPaneAnimationMillis)),
        ) {
            AnimatedOverviewPane(
                visible = usageVisible,
                modifier = Modifier
                    .weight(usageWeight)
                    .fillMaxHeight()
                    .recordOverviewBounds { bounds ->
                        updatePaneBounds(OverviewPane.UsageSchedule, bounds)
                    },
            ) {
                SchedulePane(
                    state = state,
                    usageIconLoader = usageIconLoader,
                    onUsageClick = { onAction(TimeboxxingAction.ToggleUsageSelection(it)) },
                    onClearSelection = { onAction(TimeboxxingAction.ClearUsageSelection) },
                    onCreateEntry = { onAction(TimeboxxingAction.AddDraftEntry) },
                    scrollState = scheduleScrollState,
                    modifier = Modifier.fillMaxSize(),
                    headerAction = if (canMinimize) {
                        {
                            OverviewPaneMinimizeButton(
                                pane = OverviewPane.UsageSchedule,
                                onClick = { minimizePane(OverviewPane.UsageSchedule) },
                            )
                        }
                    } else {
                        null
                    },
                )
            }
            if (usageVisible && (entriesVisible || projectsVisible)) {
                TbVerticalDivider()
            }
            AnimatedOverviewPane(
                visible = entriesVisible,
                modifier = Modifier
                    .weight(entriesWeight)
                    .fillMaxHeight()
                    .recordOverviewBounds { bounds ->
                        updatePaneBounds(OverviewPane.TimeEntries, bounds)
                    },
            ) {
                EntryBuilderPane(
                    state = state,
                    onAction = onAction,
                    showInlineSummary = false,
                    modifier = Modifier.fillMaxSize(),
                    headerAction = if (canMinimize) {
                        {
                            OverviewPaneMinimizeButton(
                                pane = OverviewPane.TimeEntries,
                                onClick = { minimizePane(OverviewPane.TimeEntries) },
                            )
                        }
                    } else {
                        null
                    },
                )
            }
            if (entriesVisible && projectsVisible) {
                TbVerticalDivider()
            }
            AnimatedOverviewPane(
                visible = projectsVisible,
                modifier = (if (projectsVisible && visiblePanes.size == 1) {
                    Modifier
                        .weight(1f)
                        .fillMaxHeight()
                } else {
                    Modifier
                        .width(projectsWidth)
                        .fillMaxHeight()
                }).recordOverviewBounds { bounds ->
                    updatePaneBounds(OverviewPane.Projects, bounds)
                },
            ) {
                ProjectSummaryRail(
                    state = state,
                    onAction = onAction,
                    modifier = Modifier.fillMaxSize(),
                    headerAction = if (canMinimize) {
                        {
                            OverviewPaneMinimizeButton(
                                pane = OverviewPane.Projects,
                                onClick = { minimizePane(OverviewPane.Projects) },
                            )
                        }
                    } else {
                        null
                    },
                )
            }
        }

        OverviewPaneGenieOverlay(
            cue = activeGenieCue,
            state = state,
            usageIconLoader = usageIconLoader,
            onFinished = { cue ->
                if (activeGenieCue?.id == cue.id) {
                    activeGenieCue = null
                }
            },
            modifier = Modifier
                .matchParentSize()
                .zIndex(2f),
        )

        MinimizedOverviewPaneTray(
            visiblePanes = minimizedPanes,
            pulsingPane = pulsingTrayPane,
            onTrayIconBoundsChanged = ::updateTrayIconBounds,
            onRestorePane = ::restorePane,
            modifier = Modifier
                .align(Alignment.BottomStart)
                .padding(start = 24.dp, bottom = 24.dp)
                .zIndex(3f),
        )
    }
}

@Composable
private fun AnimatedOverviewPane(
    visible: Boolean,
    modifier: Modifier = Modifier,
    content: @Composable () -> Unit,
) {
    AnimatedVisibility(
        visible = visible,
        modifier = modifier,
        enter = fadeIn(animationSpec = tween(OverviewPaneAnimationMillis)) +
            expandHorizontally(
                animationSpec = tween(OverviewPaneAnimationMillis),
                expandFrom = Alignment.Start,
                clip = true,
            ),
        exit = fadeOut(animationSpec = tween(OverviewPaneAnimationMillis)) +
            shrinkHorizontally(
                animationSpec = tween(OverviewPaneAnimationMillis),
                shrinkTowards = Alignment.Start,
                clip = true,
            ),
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            content()
        }
    }
}

@Composable
private fun OverviewPaneGenieOverlay(
    cue: OverviewPaneGenieCue?,
    state: TimeboxxingScreenState,
    usageIconLoader: UsageIconLoader,
    onFinished: (OverviewPaneGenieCue) -> Unit,
    modifier: Modifier = Modifier,
) {
    val currentCue = cue ?: return
    val progress = remember(currentCue.id) { Animatable(0f) }
    val density = LocalDensity.current
    val baseBounds = when (currentCue.direction) {
        OverviewPaneMotionDirection.Minimize -> currentCue.startBounds
        OverviewPaneMotionDirection.Restore -> currentCue.endBounds
    }

    LaunchedEffect(currentCue.id) {
        progress.snapTo(0f)
        progress.animateTo(
            targetValue = 1f,
            animationSpec = tween(
                durationMillis = OverviewPaneGenieMillis,
                easing = OverviewPaneGenieEasing,
            ),
        )
        onFinished(currentCue)
    }

    Box(modifier = modifier) {
        Box(
            modifier = Modifier
                .offset {
                    IntOffset(
                        x = baseBounds.left.roundToInt(),
                        y = baseBounds.top.roundToInt(),
                    )
                }
                .size(
                    width = with(density) { baseBounds.width.toDp() },
                    height = with(density) { baseBounds.height.toDp() },
                )
                .overviewPaneGenieTransform(
                    cue = currentCue,
                    baseBounds = baseBounds,
                    progress = progress.value,
                    shape = RoundedCornerShape(TbTheme.radii.card),
                    highlightColor = TbTheme.colors.accent,
                )
                .pointerInput(currentCue.id) {
                    awaitPointerEventScope {
                        while (true) {
                            awaitPointerEvent().changes.forEach { it.consume() }
                        }
                    }
                },
        ) {
            OverviewPaneProxyContent(
                pane = currentCue.pane,
                state = state,
                usageIconLoader = usageIconLoader,
                modifier = Modifier.fillMaxSize(),
            )
        }
    }
}

@Composable
private fun OverviewPaneProxyContent(
    pane: OverviewPane,
    state: TimeboxxingScreenState,
    usageIconLoader: UsageIconLoader,
    modifier: Modifier = Modifier,
) {
    when (pane) {
        OverviewPane.UsageSchedule -> SchedulePane(
            state = state,
            usageIconLoader = usageIconLoader,
            onUsageClick = {},
            onClearSelection = {},
            onCreateEntry = {},
            scrollState = rememberSchedulePaneScrollState(),
            modifier = modifier,
        )
        OverviewPane.TimeEntries -> EntryBuilderPane(
            state = state,
            onAction = {},
            showInlineSummary = false,
            modifier = modifier,
        )
        OverviewPane.Projects -> ProjectSummaryRail(
            state = state,
            onAction = {},
            modifier = modifier,
        )
    }
}

private fun Modifier.overviewPaneGenieTransform(
    cue: OverviewPaneGenieCue,
    baseBounds: Rect,
    progress: Float,
    shape: RoundedCornerShape,
    highlightColor: Color,
): Modifier =
    drawWithContent {
        if (baseBounds.width <= 0f || baseBounds.height <= 0f) {
            drawContent()
            return@drawWithContent
        }

        val rawProgress = progress.coerceIn(0f, 1f)
        val genieProgress = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> rawProgress
            OverviewPaneMotionDirection.Restore -> 1f - rawProgress
        }
        val pullPoint = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> cue.endBounds.center
            OverviewPaneMotionDirection.Restore -> cue.startBounds.center
        }
        val localPullX = pullPoint.x - baseBounds.left
        val localPullY = pullPoint.y - baseBounds.top
        val path = overviewPaneGenieClipPath(
            width = size.width,
            height = size.height,
            pullX = localPullX,
            pullY = localPullY,
            progress = genieProgress,
        )
        val highlightAlpha = genieHighlightAlpha(rawProgress)

        clipPath(path) {
            this@drawWithContent.drawContent()
        }
        drawPath(
            path = path,
            color = highlightColor.copy(alpha = highlightAlpha * 0.06f),
        )
        drawPath(
            path = path,
            color = highlightColor.copy(alpha = highlightAlpha * 0.2f),
            style = Stroke(width = 1.4.dp.toPx()),
        )
    }.graphicsLayer {
        if (baseBounds.width <= 0f || baseBounds.height <= 0f) {
            return@graphicsLayer
        }

        val rawProgress = progress.coerceIn(0f, 1f)
        val pullPoint = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> cue.endBounds.center
            OverviewPaneMotionDirection.Restore -> cue.startBounds.center
        }
        val originXFraction = ((pullPoint.x - baseBounds.left) / baseBounds.width).coerceIn(0f, 1f)
        val originYFraction = ((pullPoint.y - baseBounds.top) / baseBounds.height).coerceIn(0f, 1f)
        val originX = baseBounds.left + baseBounds.width * originXFraction
        val originY = baseBounds.top + baseBounds.height * originYFraction
        val originProgress = easeOutCubic(stagedProgress(rawProgress, 0.12f, 1f))
        val widthProgress = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> stagedProgress(rawProgress, 0f, 0.72f)
            OverviewPaneMotionDirection.Restore -> stagedProgress(rawProgress, 0.14f, 1f)
        }
        val heightProgress = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> stagedProgress(rawProgress, 0.36f, 1f)
            OverviewPaneMotionDirection.Restore -> stagedProgress(rawProgress, 0f, 0.72f)
        }
        val targetScaleX = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> (cue.endBounds.width / baseBounds.width).coerceAtLeast(0.02f)
            OverviewPaneMotionDirection.Restore -> 1f
        }
        val targetScaleY = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> (cue.endBounds.height / baseBounds.height).coerceAtLeast(0.02f)
            OverviewPaneMotionDirection.Restore -> 1f
        }
        val startScaleX = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> 1f
            OverviewPaneMotionDirection.Restore -> (cue.startBounds.width / baseBounds.width).coerceAtLeast(0.02f)
        }
        val startScaleY = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> 1f
            OverviewPaneMotionDirection.Restore -> (cue.startBounds.height / baseBounds.height).coerceAtLeast(0.02f)
        }
        val targetOriginX = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> cue.endBounds.center.x
            OverviewPaneMotionDirection.Restore -> originX
        }
        val targetOriginY = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> cue.endBounds.center.y
            OverviewPaneMotionDirection.Restore -> originY
        }
        val startOriginX = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> originX
            OverviewPaneMotionDirection.Restore -> cue.startBounds.center.x
        }
        val startOriginY = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> originY
            OverviewPaneMotionDirection.Restore -> cue.startBounds.center.y
        }

        transformOrigin = TransformOrigin(originXFraction, originYFraction)
        translationX = lerpFloat(startOriginX, targetOriginX, originProgress) - originX
        translationY = lerpFloat(startOriginY, targetOriginY, originProgress) - originY
        scaleX = lerpFloat(startScaleX, targetScaleX, widthProgress)
        scaleY = lerpFloat(startScaleY, targetScaleY, heightProgress)
        alpha = when (cue.direction) {
            OverviewPaneMotionDirection.Minimize -> lerpFloat(
                start = 1f,
                end = 0.64f,
                fraction = smoothStep(((rawProgress - 0.68f) / 0.32f).coerceIn(0f, 1f)),
            )
            OverviewPaneMotionDirection.Restore -> lerpFloat(
                start = 0.64f,
                end = 1f,
                fraction = smoothStep((rawProgress / 0.32f).coerceIn(0f, 1f)),
            )
        }
        this.shape = shape
        clip = true
    }

private fun overviewPaneGenieClipPath(
    width: Float,
    height: Float,
    pullX: Float,
    pullY: Float,
    progress: Float,
): Path {
    val bend = stagedProgress(progress, 0.02f, 0.86f)
    val waist = stagedProgress(progress, 0.12f, 1f)
    val pullsTowardBottom = pullY >= height / 2f
    val pullsLeft = pullX < width / 2f
    val targetWidth = (width * lerpFloat(0.84f, 0.08f, waist)).coerceAtLeast(20f.coerceAtMost(width))
    val targetHalfWidth = targetWidth / 2f
    val targetLeft = pullX - targetHalfWidth
    val targetRight = pullX + targetHalfWidth
    val nearEdgeInset = width * 0.012f * bend * bend
    val farEdgeInset = width * 0.075f * bend * bend
    val controlPull = lerpFloat(0.22f, 0.96f, bend)
    val nearControlPull = (controlPull * 0.74f).coerceIn(0f, 1f)
    val farControlPull = (controlPull * 1.08f).coerceIn(0f, 1f)

    return Path().apply {
        if (pullsTowardBottom) {
            val topLeft = if (pullsLeft) nearEdgeInset else farEdgeInset
            val topRight = width - if (pullsLeft) farEdgeInset else nearEdgeInset
            val bottomLeft = lerpFloat(0f, targetLeft, bend)
            val bottomRight = lerpFloat(width, targetRight, bend)
            val leftPull = if (pullsLeft) nearControlPull else farControlPull
            val rightPull = if (pullsLeft) farControlPull else nearControlPull

            moveTo(topLeft, 0f)
            lineTo(topRight, 0f)
            cubicTo(
                lerpFloat(width, bottomRight, rightPull * 0.58f),
                height * 0.28f,
                lerpFloat(width, bottomRight, rightPull),
                height * 0.78f,
                bottomRight,
                height,
            )
            lineTo(bottomLeft, height)
            cubicTo(
                lerpFloat(0f, bottomLeft, leftPull),
                height * 0.78f,
                lerpFloat(0f, bottomLeft, leftPull * 0.58f),
                height * 0.28f,
                topLeft,
                0f,
            )
        } else {
            val topLeft = lerpFloat(0f, targetLeft, bend)
            val topRight = lerpFloat(width, targetRight, bend)
            val bottomLeft = if (pullsLeft) nearEdgeInset else farEdgeInset
            val bottomRight = width - if (pullsLeft) farEdgeInset else nearEdgeInset
            val leftPull = if (pullsLeft) nearControlPull else farControlPull
            val rightPull = if (pullsLeft) farControlPull else nearControlPull

            moveTo(topLeft, 0f)
            lineTo(topRight, 0f)
            cubicTo(
                lerpFloat(width, bottomRight, 1f - rightPull),
                height * 0.22f,
                lerpFloat(width, bottomRight, 0.58f),
                height * 0.72f,
                bottomRight,
                height,
            )
            lineTo(bottomLeft, height)
            cubicTo(
                lerpFloat(0f, bottomLeft, 0.58f),
                height * 0.72f,
                lerpFloat(0f, bottomLeft, 1f - leftPull),
                height * 0.22f,
                topLeft,
                0f,
            )
        }
        close()
    }
}

private fun Modifier.recordOverviewBounds(onBoundsChanged: (Rect) -> Unit): Modifier =
    onGloballyPositioned { coordinates ->
        if (coordinates.size.width > 0 && coordinates.size.height > 0) {
            onBoundsChanged(coordinates.boundsInRoot())
        }
    }

private fun Rect.toLocalBounds(containerBounds: Rect): Rect =
    Rect(
        left = left - containerBounds.left,
        top = top - containerBounds.top,
        right = right - containerBounds.left,
        bottom = bottom - containerBounds.top,
    )

private fun Rect.centeredScale(scale: Float): Rect {
    val scaledWidth = width * scale
    val scaledHeight = height * scale
    val center = center
    return Rect(
        left = center.x - scaledWidth / 2f,
        top = center.y - scaledHeight / 2f,
        right = center.x + scaledWidth / 2f,
        bottom = center.y + scaledHeight / 2f,
    )
}

private fun lerpFloat(start: Float, end: Float, fraction: Float): Float =
    start + (end - start) * fraction

private fun easeOutCubic(fraction: Float): Float {
    val inverse = 1f - fraction
    return 1f - inverse * inverse * inverse
}

private fun genieHighlightAlpha(fraction: Float): Float =
    stagedProgress(fraction, 0.12f, 0.34f) *
        (1f - stagedProgress(fraction, 0.62f, 0.9f))

private fun stagedProgress(fraction: Float, start: Float, end: Float): Float {
    if (end <= start) {
        return if (fraction >= end) 1f else 0f
    }
    return smoothStep(((fraction - start) / (end - start)).coerceIn(0f, 1f))
}

private fun smoothStep(fraction: Float): Float =
    fraction * fraction * (3f - 2f * fraction)

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
    onTrayIconBoundsChanged: (OverviewPane, Rect) -> Unit,
    onRestorePane: (OverviewPane) -> Unit,
    modifier: Modifier = Modifier,
) {
    if (visiblePanes.isEmpty()) {
        return
    }

    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        OverviewPane.entries.forEach { pane ->
            val isPulsing = pulsingPane == pane
            val pulseScale by animateFloatAsState(
                targetValue = if (isPulsing) 1.1f else 1f,
                animationSpec = tween(OverviewPaneTrayPulseMillis.toInt()),
                label = "${pane.name}-tray-pulse",
            )
            val ringScale by animateFloatAsState(
                targetValue = if (isPulsing) 1.36f else 0.82f,
                animationSpec = tween(OverviewPaneTrayPulseMillis.toInt()),
                label = "${pane.name}-tray-ring-scale",
            )
            val ringAlpha by animateFloatAsState(
                targetValue = if (isPulsing) 0.28f else 0f,
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
                            .size(32.dp)
                            .graphicsLayer {
                                alpha = ringAlpha
                                scaleX = ringScale
                                scaleY = ringScale
                        },
                        shape = RoundedCornerShape(TbTheme.radii.pill),
                        color = TbTheme.colors.accent.copy(alpha = 0.12f),
                        border = BorderStroke(1.dp, TbTheme.colors.accent.copy(alpha = 0.28f)),
                        contentAlignment = Alignment.Center,
                    ) {}
                    Box(
                        modifier = Modifier
                            .recordOverviewBounds { bounds ->
                                onTrayIconBoundsChanged(pane, bounds)
                            }
                            .graphicsLayer {
                                scaleX = pulseScale
                                scaleY = pulseScale
                            },
                    ) {
                        TbIconButton(
                            icon = pane.icon,
                            contentDescription = "Restore ${pane.label}",
                            onClick = { onRestorePane(pane) },
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
            if (state.amaMessages.isEmpty() && !state.amaLoading) {
                item("empty") {
                    AmaEmptyState()
                }
            }
            items(state.amaMessages, key = { it.id }) { message ->
                AmaMessageRow(message)
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

        AmaComposer(
            value = state.amaInput,
            loading = state.amaLoading,
            onValueChange = { onAction(TimeboxxingAction.UpdateAmaInput(it)) },
            onSubmit = { onAction(TimeboxxingAction.SubmitAmaQuestion) },
        )
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
private fun AmaEmptyState() {
    TbCard(
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = 120.dp),
        color = TbTheme.colors.surface,
        borderColor = TbTheme.colors.separator,
    ) {
        Column(
            modifier = Modifier.padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            TbText(
                text = "Ask about your activity",
                style = TbTheme.typography.title2,
            )
            TbText(
                text = "Try: \"What did I spend time on this afternoon?\"",
                style = TbTheme.typography.body,
                color = TbTheme.colors.secondaryText,
            )
        }
    }
}

@Composable
private fun AmaMessageRow(message: AmaMessage) {
    val isUser = message.role == AmaMessageRole.User
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = if (isUser) Arrangement.End else Arrangement.Start,
    ) {
        TbSurface(
            modifier = Modifier.widthIn(max = 760.dp),
            shape = RoundedCornerShape(TbTheme.radii.card),
            color = if (isUser) TbTheme.colors.accent else TbTheme.colors.surface,
            contentColor = if (isUser) TbTheme.colors.accentText else TbTheme.colors.text,
            border = if (isUser) null else BorderStroke(Dp.Hairline, TbTheme.colors.separator),
        ) {
            Column(
                modifier = Modifier.padding(14.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                AmaMarkdownText(
                    content = message.content,
                    isUser = isUser,
                )
                if (!isUser && message.model != null) {
                    TbText(
                        text = message.model,
                        style = TbTheme.typography.caption,
                        color = TbTheme.colors.tertiaryText,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                if (!isUser && message.artifacts.isNotEmpty()) {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        message.artifacts.forEach { artifact ->
                            when (artifact) {
                                is AmaAppUsageChart -> AmaAppUsageChartCard(artifact)
                            }
                        }
                    }
                }
                if (message.sources.isNotEmpty()) {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        message.sources.take(3).forEach { source ->
                            AmaSourceCard(source)
                        }
                    }
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
private fun AmaSourceCard(source: AmaSource) {
    TbSurface(
        modifier = Modifier.fillMaxWidth(),
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
private fun AmaComposer(
    value: String,
    loading: Boolean,
    onValueChange: (String) -> Unit,
    onSubmit: () -> Unit,
) {
    var lastEscapePress by remember { mutableStateOf<TimeMark?>(null) }
    val canSubmit = value.isNotBlank() && !loading

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(TbTheme.colors.surface)
            .padding(16.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.Bottom,
    ) {
        TbTextField(
            modifier = Modifier.weight(1f),
            value = value,
            onValueChange = onValueChange,
            minLines = 2,
            maxLines = 6,
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
        OverviewPane.TimeEntries -> "Time entries"
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

private fun Double.formatDistance(): String =
    ((this * 1000.0).roundToInt() / 1000.0).toString()

@Composable
fun ModeToggle(
    mode: EntryMode,
    onModeChange: (EntryMode) -> Unit,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        EntryMode.entries.forEach { option ->
            TbButton(
                onClick = { onModeChange(option) },
                variant = if (option == mode) TbButtonVariant.Primary else TbButtonVariant.Secondary,
                contentPadding = androidx.compose.foundation.layout.PaddingValues(horizontal = 12.dp, vertical = 7.dp),
            ) {
                TbText(option.label, style = TbTheme.typography.button)
            }
        }
    }
}

private val EntryMode.label: String
    get() = when (this) {
        EntryMode.Timesheet -> "Timesheet"
        EntryMode.Invoice -> "Invoice"
    }
