package com.timeboxxing.app.ui

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
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.requiredWidth
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowLeft
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.rounded.Send
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.Dashboard
import androidx.compose.material.icons.rounded.Edit
import androidx.compose.material.icons.rounded.ExpandMore
import androidx.compose.material.icons.rounded.Folder
import androidx.compose.material.icons.rounded.QuestionAnswer
import androidx.compose.material.icons.rounded.Schedule
import com.timeboxxing.app.data.currentCalendarDate
import com.timeboxxing.app.model.AmaMessage
import com.timeboxxing.app.model.AmaMessageRole
import com.timeboxxing.app.model.AmaIndexState
import com.timeboxxing.app.model.AmaIndexStatus
import com.timeboxxing.app.model.AmaSource
import com.timeboxxing.app.model.CalendarDate
import com.timeboxxing.app.model.EntryMode
import com.timeboxxing.app.model.WeekdayShortLabels
import com.timeboxxing.app.model.calendarMonthGrid
import com.timeboxxing.app.model.monthYearLabel
import com.timeboxxing.app.model.plusMonths
import com.timeboxxing.app.model.startOfMonth
import com.timeboxxing.app.state.TimeboxxingAction
import com.timeboxxing.app.state.TimeboxxingScreenState
import com.timeboxxing.app.state.TimeboxxingSection
import com.timeboxxing.app.state.WorkspacePane
import kotlin.math.roundToInt

private enum class WorkspaceLayout {
    Wide,
    Medium,
    Compact,
}

private enum class PaneCollapseEdge {
    Left,
    Right,
}

private val CollapsedPaneWidth = 46.dp
private val CollapsedPaneLabelWidth = 180.dp

@Composable
fun TimeboxxingScreen(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    modifier: Modifier = Modifier,
    usageIconLoader: UsageIconLoader = NoOpUsageIconLoader,
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
            contentWidth >= 1240.dp -> WorkspaceLayout.Wide
            contentWidth >= 860.dp -> WorkspaceLayout.Medium
            else -> WorkspaceLayout.Compact
        }

        Row(
            modifier = Modifier.fillMaxSize(),
        ) {
            AppNavigationPane(
                selectedSection = state.selectedSection,
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
                            WorkspaceLayout.Wide -> WideWorkspace(state, onAction, usageIconLoader)
                            WorkspaceLayout.Medium -> MediumWorkspace(state, onAction, usageIconLoader)
                            WorkspaceLayout.Compact -> CompactWorkspace(state, onAction, usageIconLoader)
                        }
                    }

                    TimeboxxingSection.Ama -> AmaPane(
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
private fun AppNavigationPane(
    selectedSection: TimeboxxingSection,
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

        TimeboxxingSection.entries.forEach { section ->
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
) {
    val onlyProjectsExpanded = WorkspacePane.UsageSchedule in state.collapsedPanes &&
        WorkspacePane.TimeEntries in state.collapsedPanes &&
        WorkspacePane.Projects !in state.collapsedPanes

    Row(
        modifier = Modifier.fillMaxSize(),
    ) {
        CollapsibleWorkspacePane(
            pane = WorkspacePane.UsageSchedule,
            collapsed = WorkspacePane.UsageSchedule in state.collapsedPanes,
            expandedModifier = Modifier.weight(0.98f),
            collapseEdge = PaneCollapseEdge.Left,
            collapseEnabled = canCollapseExpandedPane(state, WorkspacePane.UsageSchedule),
            onToggle = { onAction(TimeboxxingAction.ToggleWorkspacePaneCollapsed(WorkspacePane.UsageSchedule)) },
        ) { paneModifier, headerAction ->
            SchedulePane(
                state = state,
                usageIconLoader = usageIconLoader,
                onUsageClick = { onAction(TimeboxxingAction.ToggleUsageSelection(it)) },
                onClearSelection = { onAction(TimeboxxingAction.ClearUsageSelection) },
                onCreateEntry = { onAction(TimeboxxingAction.AddDraftEntry) },
                modifier = paneModifier,
                headerAction = headerAction,
            )
        }
        TbVerticalDivider()
        CollapsibleWorkspacePane(
            pane = WorkspacePane.TimeEntries,
            collapsed = WorkspacePane.TimeEntries in state.collapsedPanes,
            expandedModifier = Modifier.weight(1.05f),
            collapseEdge = PaneCollapseEdge.Right,
            collapseEnabled = canCollapseExpandedPane(state, WorkspacePane.TimeEntries),
            onToggle = { onAction(TimeboxxingAction.ToggleWorkspacePaneCollapsed(WorkspacePane.TimeEntries)) },
        ) { paneModifier, headerAction ->
            EntryBuilderPane(
                state = state,
                onAction = onAction,
                showInlineSummary = false,
                modifier = paneModifier,
                headerAction = headerAction,
            )
        }
        TbVerticalDivider()
        CollapsibleWorkspacePane(
            pane = WorkspacePane.Projects,
            collapsed = WorkspacePane.Projects in state.collapsedPanes,
            expandedModifier = if (onlyProjectsExpanded) Modifier.weight(1f) else Modifier.width(280.dp),
            collapseEdge = PaneCollapseEdge.Right,
            collapseEnabled = canCollapseExpandedPane(state, WorkspacePane.Projects),
            onToggle = { onAction(TimeboxxingAction.ToggleWorkspacePaneCollapsed(WorkspacePane.Projects)) },
        ) { paneModifier, headerAction ->
            ProjectSummaryRail(
                state = state,
                onAction = onAction,
                modifier = paneModifier,
                headerAction = headerAction,
            )
        }
    }
}

@Composable
private fun MediumWorkspace(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    usageIconLoader: UsageIconLoader,
) {
    val autoExpandedUsage = (
        WorkspacePane.UsageSchedule in state.collapsedPanes &&
        WorkspacePane.TimeEntries in state.collapsedPanes
    )
    val collapsedPanes = if (autoExpandedUsage) {
        state.collapsedPanes - WorkspacePane.UsageSchedule
    } else {
        state.collapsedPanes
    }

    Row(
        modifier = Modifier.fillMaxSize(),
    ) {
        CollapsibleWorkspacePane(
            pane = WorkspacePane.UsageSchedule,
            collapsed = WorkspacePane.UsageSchedule in collapsedPanes,
            expandedModifier = Modifier.weight(0.92f),
            collapseEdge = PaneCollapseEdge.Left,
            collapseEnabled = !autoExpandedUsage && canCollapseExpandedPane(state, WorkspacePane.UsageSchedule),
            onToggle = { onAction(TimeboxxingAction.ToggleWorkspacePaneCollapsed(WorkspacePane.UsageSchedule)) },
        ) { paneModifier, headerAction ->
            SchedulePane(
                state = state,
                usageIconLoader = usageIconLoader,
                onUsageClick = { onAction(TimeboxxingAction.ToggleUsageSelection(it)) },
                onClearSelection = { onAction(TimeboxxingAction.ClearUsageSelection) },
                onCreateEntry = { onAction(TimeboxxingAction.AddDraftEntry) },
                modifier = paneModifier,
                headerAction = headerAction,
            )
        }
        TbVerticalDivider()
        CollapsibleWorkspacePane(
            pane = WorkspacePane.TimeEntries,
            collapsed = WorkspacePane.TimeEntries in collapsedPanes,
            expandedModifier = Modifier.weight(1.08f),
            collapseEdge = PaneCollapseEdge.Right,
            collapseEnabled = canCollapseExpandedPane(state, WorkspacePane.TimeEntries),
            onToggle = { onAction(TimeboxxingAction.ToggleWorkspacePaneCollapsed(WorkspacePane.TimeEntries)) },
        ) { paneModifier, headerAction ->
            EntryBuilderPane(
                state = state,
                onAction = onAction,
                showInlineSummary = true,
                modifier = paneModifier,
                headerAction = headerAction,
            )
        }
    }
}

@Composable
private fun CompactWorkspace(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    usageIconLoader: UsageIconLoader,
) {
    var selectedTab by remember { mutableStateOf(0) }
    val tabs = listOf("Usage", "Entries")

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
                modifier = Modifier.fillMaxSize(),
            )

            else -> EntryBuilderPane(
                state = state,
                onAction = onAction,
                showInlineSummary = true,
                modifier = Modifier
                    .fillMaxSize()
                    .clip(RoundedCornerShape(0.dp)),
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
                TbText(
                    text = message.content,
                    style = TbTheme.typography.body,
                    color = if (isUser) TbTheme.colors.accentText else TbTheme.colors.text,
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
            singleLine = true,
        )
        TbIconButton(
            icon = Icons.AutoMirrored.Rounded.Send,
            contentDescription = "Send AMA question",
            onClick = onSubmit,
            enabled = value.isNotBlank() && !loading,
            variant = TbButtonVariant.Primary,
        )
    }
}

@Composable
private fun CollapsibleWorkspacePane(
    pane: WorkspacePane,
    collapsed: Boolean,
    expandedModifier: Modifier,
    collapseEdge: PaneCollapseEdge,
    collapseEnabled: Boolean,
    onToggle: () -> Unit,
    content: @Composable (Modifier, @Composable () -> Unit) -> Unit,
) {
    if (collapsed) {
        CollapsedWorkspacePaneStrip(
            pane = pane,
            onClick = onToggle,
            modifier = Modifier
                .width(CollapsedPaneWidth)
                .fillMaxHeight(),
        )
    } else {
        content(expandedModifier) {
            PaneCollapseButton(
                pane = pane,
                collapseEdge = collapseEdge,
                enabled = collapseEnabled,
                onClick = onToggle,
            )
        }
    }
}

@Composable
private fun PaneCollapseButton(
    pane: WorkspacePane,
    collapseEdge: PaneCollapseEdge,
    enabled: Boolean,
    onClick: () -> Unit,
) {
    TbIconButton(
        icon = if (collapseEdge == PaneCollapseEdge.Left) {
            Icons.AutoMirrored.Rounded.KeyboardArrowLeft
        } else {
            Icons.AutoMirrored.Rounded.KeyboardArrowRight
        },
        contentDescription = "Collapse ${pane.label}",
        onClick = onClick,
        enabled = enabled,
        variant = TbButtonVariant.Ghost,
    )
}

@Composable
private fun CollapsedWorkspacePaneStrip(
    pane: WorkspacePane,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = TbTheme.colors
    val interactionSource = remember { MutableInteractionSource() }
    val hovered by interactionSource.collectIsHoveredAsState()
    val pressed by interactionSource.collectIsPressedAsState()
    val background = when {
        pressed -> colors.controlFillHover
        hovered -> colors.elevatedSurface
        else -> colors.groupedSurface
    }

    TbTooltip(text = "Expand ${pane.label}") {
        TbSurface(
            modifier = modifier.clickable(
                interactionSource = interactionSource,
                indication = null,
                onClick = onClick,
            ),
            color = background,
            contentColor = colors.secondaryText,
            contentAlignment = Alignment.Center,
        ) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(vertical = 14.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center,
            ) {
                TbIcon(
                    imageVector = pane.icon,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp),
                    tint = if (hovered || pressed) colors.accent else colors.secondaryText,
                )
                Box(
                    modifier = Modifier
                        .height(CollapsedPaneLabelWidth)
                        .width(28.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    TbText(
                        modifier = Modifier
                            .rotate(-90f)
                            .requiredWidth(CollapsedPaneLabelWidth),
                        text = pane.label,
                        style = TbTheme.typography.label.copy(textAlign = TextAlign.Center),
                        color = if (hovered || pressed) colors.text else colors.secondaryText,
                        maxLines = 1,
                        overflow = TextOverflow.Clip,
                    )
                }
            }
        }
    }
}

private val WorkspacePane.label: String
    get() = when (this) {
        WorkspacePane.UsageSchedule -> "Usage schedule"
        WorkspacePane.TimeEntries -> "Time entries"
        WorkspacePane.Projects -> "Projects"
    }

private val WorkspacePane.icon: ImageVector
    get() = when (this) {
        WorkspacePane.UsageSchedule -> Icons.Rounded.Schedule
        WorkspacePane.TimeEntries -> Icons.Rounded.Edit
        WorkspacePane.Projects -> Icons.Rounded.Folder
    }

private val TimeboxxingSection.label: String
    get() = when (this) {
        TimeboxxingSection.Overview -> "Overview"
        TimeboxxingSection.Ama -> "AMA"
    }

private val TimeboxxingSection.icon: ImageVector
    get() = when (this) {
        TimeboxxingSection.Overview -> Icons.Rounded.Dashboard
        TimeboxxingSection.Ama -> Icons.Rounded.QuestionAnswer
    }

private fun Double.formatDistance(): String =
    ((this * 1000.0).roundToInt() / 1000.0).toString()

private fun canCollapseExpandedPane(
    state: TimeboxxingScreenState,
    pane: WorkspacePane,
): Boolean =
    pane in state.collapsedPanes || state.collapsedPanes.size < WorkspacePane.entries.size - 1

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
