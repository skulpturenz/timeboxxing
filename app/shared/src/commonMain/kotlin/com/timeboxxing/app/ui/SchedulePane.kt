package com.timeboxxing.app.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.ArrowDownward
import androidx.compose.material.icons.rounded.ArrowUpward
import androidx.compose.material.icons.rounded.Close
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.zIndex
import com.timeboxxing.domain.model.UsageEvent
import com.timeboxxing.domain.model.UsageSourceType
import com.timeboxxing.domain.model.formatClockTime
import com.timeboxxing.domain.model.formatDuration
import com.timeboxxing.app.presentation.TimeboxxingScreenState
import com.composeunstyled.AnchorAlignment
import com.composeunstyled.AnchorSide
import kotlinx.coroutines.launch
import kotlin.time.Clock
import kotlin.time.ExperimentalTime

private val TimelineTimeLabelWidth = 72.dp
private val TimelineRuleGap = 12.dp
private val TimelineGridLineThickness = 1.dp
internal const val TimelineEventLayerZIndex = 0f
internal const val TimelineNowMarkerLayerZIndex = 1f

class SchedulePaneScrollState internal constructor(
    internal val listState: LazyListState,
    internal val autoScrollState: SchedulePaneAutoScrollState = SchedulePaneAutoScrollState(),
)

@Composable
fun rememberSchedulePaneScrollState(): SchedulePaneScrollState {
    val listState = rememberLazyListState()
    return remember(listState) { SchedulePaneScrollState(listState) }
}

internal class SchedulePaneAutoScrollState {
    private var handledInitialScrollKey: ScheduleInitialScrollKey? = null
    private var handledFocusedEntryScrollKey: ScheduleFocusedEntryScrollKey? = null

    fun shouldHandleInitialScroll(
        dayStartedAtEpochMillis: Long,
        zoomMinutes: Int,
        targetDp: Float,
    ): Boolean {
        val key = ScheduleInitialScrollKey(
            dayStartedAtEpochMillis = dayStartedAtEpochMillis,
            zoomMinutes = zoomMinutes,
            targetDp = targetDp,
        )
        if (handledInitialScrollKey == key) return false

        handledInitialScrollKey = key
        return true
    }

    fun shouldHandleFocusedEntryScroll(
        entryId: String,
        startMinute: Int,
        sourceUsageIds: Set<String>,
        zoomMinutes: Int,
        targetDp: Float,
    ): Boolean {
        val key = ScheduleFocusedEntryScrollKey(
            entryId = entryId,
            startMinute = startMinute,
            sourceUsageIds = sourceUsageIds,
            zoomMinutes = zoomMinutes,
            targetDp = targetDp,
        )
        if (handledFocusedEntryScrollKey == key) return false

        handledFocusedEntryScrollKey = key
        return true
    }
}

internal data class ScheduleInitialScrollKey(
    val dayStartedAtEpochMillis: Long,
    val zoomMinutes: Int,
    val targetDp: Float,
)

internal data class ScheduleFocusedEntryScrollKey(
    val entryId: String,
    val startMinute: Int,
    val sourceUsageIds: Set<String>,
    val zoomMinutes: Int,
    val targetDp: Float,
)

@Composable
fun SchedulePane(
    state: TimeboxxingScreenState,
    usageIconLoader: UsageIconLoader = NoOpUsageIconLoader,
    onUsageClick: (String) -> Unit,
    onClearSelection: () -> Unit,
    onCreateEntry: () -> Unit,
    scrollState: SchedulePaneScrollState = rememberSchedulePaneScrollState(),
    modifier: Modifier = Modifier,
    headerAction: @Composable (() -> Unit)? = null,
) {
    val assignedUsageIds = state.entries.flatMap { it.sourceUsageIds }.toSet()
    val selectedCount = state.selectedUsageIds.size
    val nowMinute = currentMinuteForSelectedDay(state)
    val focusedEntry = remember(state.scheduleFocusEntryId, state.entries) {
        state.scheduleFocusEntryId?.let { entryId ->
            state.entries.firstOrNull { it.id == entryId }
        }
    }
    val visibleUsageEvents = remember(state.usageEvents) {
        scheduleTimelineUsageEvents(state.usageEvents)
    }
    val timelineGrid = remember(visibleUsageEvents, state.zoomMinutes) {
        buildTimelineGrid(
            events = visibleUsageEvents,
            zoomMinutes = state.zoomMinutes,
        )
    }
    val listState = scrollState.listState
    val density = LocalDensity.current
    val coroutineScope = rememberCoroutineScope()
    var scrollViewportHeightPx by remember { mutableIntStateOf(0) }
    val viewportHeightDp = with(density) { scrollViewportHeightPx.toDp().value }
    val viewportTopDp by remember(listState, timelineGrid, density) {
        derivedStateOf {
            listState.firstVisibleItemIndex * timelineGrid.rowHeightDp +
                    with(density) { listState.firstVisibleItemScrollOffset.toDp().value }
        }
    }
    val nowOffsetDp = nowMinute?.let { minute -> timelineMinuteOffsetDp(timelineGrid, minute) }
    val nowScrollDirection = if (nowOffsetDp != null && scrollViewportHeightPx > 0) {
        timelineNowScrollDirection(
            nowOffsetDp = nowOffsetDp,
            viewportTopDp = viewportTopDp,
            viewportHeightDp = viewportHeightDp,
        )
    } else {
        null
    }
    val firstUsagePlacementTopDp = timelineGrid.placements.firstOrNull()?.displayTopDp
    val initialTimelineTargetDp = remember(timelineGrid, firstUsagePlacementTopDp, nowMinute) {
        timelineFirstEventScrollDp(timelineGrid)
            ?: nowMinute?.let { minute ->
                (minute * timelineGrid.dpPerMinute - TimelineInitialScrollContextDp).coerceAtLeast(0f)
            }
    }

    LaunchedEffect(
        scrollState,
        state.selectedDay.startedAtEpochMillis,
        state.zoomMinutes,
        scrollViewportHeightPx,
        initialTimelineTargetDp,
    ) {
        if (
            state.scheduleFocusEntryId == null &&
            scrollViewportHeightPx > 0 &&
            initialTimelineTargetDp != null &&
            scrollState.autoScrollState.shouldHandleInitialScroll(
                dayStartedAtEpochMillis = state.selectedDay.startedAtEpochMillis,
                zoomMinutes = state.zoomMinutes,
                targetDp = initialTimelineTargetDp,
            )
        ) {
            scrollTimelineToDp(
                listState = listState,
                grid = timelineGrid,
                targetDp = initialTimelineTargetDp,
                density = density,
                animated = false,
            )
        }
    }

    Box(
        modifier = modifier
            .fillMaxSize()
            .background(TbTheme.colors.surface),
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(24.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    TbText(
                        text = "Usage schedule",
                        style = TbTheme.typography.largeTitle,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    TbText(
                        text = "Automatically captured apps, tabs, meetings, and documents",
                        style = TbTheme.typography.body,
                        color = TbTheme.colors.secondaryText,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                headerAction?.invoke()
            }

            DayStatusStrip(
                modifier = Modifier.padding(top = 16.dp),
                zoomMinutes = state.zoomMinutes,
                capturedMinutes = state.capturedMinutes,
                unassignedMinutes = state.unassignedUsageMinutes,
                selectedMinutes = state.selectedUsageMinutes,
                selectedCount = selectedCount,
                entryCount = state.entries.size,
                loading = state.usageLoading,
            )

            if (selectedCount > 0) {
                SelectionActionBar(
                    modifier = Modifier.padding(top = 12.dp),
                    selectedCount = selectedCount,
                    selectedMinutes = state.selectedUsageMinutes,
                    onClearSelection = onClearSelection,
                    onCreateEntry = onCreateEntry,
                )
            }

            Box(
                modifier = Modifier
                    .padding(top = 20.dp)
                    .fillMaxWidth()
                    .weight(1f)
                    .onSizeChanged { scrollViewportHeightPx = it.height },
            ) {
                LaunchedEffect(
                    scrollState,
                    focusedEntry?.id,
                    focusedEntry?.startMinute,
                    focusedEntry?.sourceUsageIds,
                    visibleUsageEvents,
                    state.zoomMinutes,
                    scrollViewportHeightPx,
                ) {
                    if (scrollViewportHeightPx > 0) {
                        focusedEntry?.let { entry ->
                            val viewportHeightDp = with(density) { scrollViewportHeightPx.toDp().value }
                            val targetDp = timelineEntryScrollDp(
                                grid = timelineGrid,
                                entry = entry,
                                viewportHeightDp = viewportHeightDp,
                            )
                            if (!scrollState.autoScrollState.shouldHandleFocusedEntryScroll(
                                    entryId = entry.id,
                                    startMinute = entry.startMinute,
                                    sourceUsageIds = entry.sourceUsageIds,
                                    zoomMinutes = state.zoomMinutes,
                                    targetDp = targetDp,
                                )
                            ) {
                                return@let
                            }
                            scrollTimelineToDp(
                                listState = listState,
                                grid = timelineGrid,
                                targetDp = targetDp,
                                density = density,
                                animated = true,
                            )
                        }
                    }
                }

                TimelineGridView(
                    grid = timelineGrid,
                    nowMinute = nowMinute,
                    selectedUsageIds = state.selectedUsageIds,
                    assignedUsageIds = assignedUsageIds,
                    usageIconLoader = usageIconLoader,
                    listState = listState,
                    onUsageClick = onUsageClick,
                )

                if (timelineGrid.placements.isEmpty() && nowMinute == null) {
                    ScheduleStateMessage(
                        loading = state.usageLoading,
                        unavailable = state.notice != null,
                        modifier = Modifier.align(Alignment.Center),
                    )
                }
            }
        }

        nowScrollDirection?.let { direction ->
            Box(
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(24.dp),
            ) {
                TbIconButton(
                    icon = when (direction) {
                        TimelineScrollDirection.Up -> Icons.Rounded.ArrowUpward
                        TimelineScrollDirection.Down -> Icons.Rounded.ArrowDownward
                    },
                    contentDescription = "Scroll to now",
                    onClick = {
                        nowMinute?.let { minute ->
                            coroutineScope.launch {
                                scrollTimelineToDp(
                                    listState = listState,
                                    grid = timelineGrid,
                                    targetDp = timelineScrollToMinuteDp(
                                        grid = timelineGrid,
                                        minute = minute,
                                        viewportHeightDp = viewportHeightDp,
                                    ),
                                    density = density,
                                    animated = true,
                                )
                            }
                        }
                    },
                    variant = TbButtonVariant.Secondary,
                )
            }
        }
    }
}

internal fun scheduleTimelineUsageEvents(events: List<UsageEvent>): List<UsageEvent> =
    events.filterNot { it.isActive }

@Composable
private fun DayStatusStrip(
    zoomMinutes: Int,
    capturedMinutes: Int,
    unassignedMinutes: Int,
    selectedMinutes: Int,
    selectedCount: Int,
    entryCount: Int,
    loading: Boolean,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        TbPill("${zoomMinutes}m grid")
        StatusMetric(label = "Captured", value = formatDuration(capturedMinutes))
        StatusMetric(label = "Unassigned", value = formatDuration(unassignedMinutes))
        StatusMetric(label = "Entries", value = entryCount.toString())
        if (loading) {
            TbPill("Loading", emphasized = true)
        } else if (selectedCount > 0) {
            StatusMetric(
                label = "$selectedCount selected",
                value = formatDuration(selectedMinutes),
                emphasized = true,
            )
        }
    }
}

@Composable
private fun StatusMetric(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
    emphasized: Boolean = false,
) {
    TbSurface(
        modifier = modifier.widthIn(min = 86.dp),
        shape = RoundedCornerShape(TbTheme.radii.pill),
        color = if (emphasized) TbTheme.colors.accentSubtle else TbTheme.colors.controlFill,
        contentColor = if (emphasized) TbTheme.colors.text else TbTheme.colors.secondaryText,
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 5.dp),
            horizontalArrangement = Arrangement.spacedBy(6.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TbText(
                text = label,
                style = TbTheme.typography.label,
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
private fun SelectionActionBar(
    selectedCount: Int,
    selectedMinutes: Int,
    onClearSelection: () -> Unit,
    onCreateEntry: () -> Unit,
    modifier: Modifier = Modifier,
) {
    TbSurface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(TbTheme.radii.card),
        color = TbTheme.colors.accentSubtle,
        border = BorderStroke(Dp.Hairline, TbTheme.colors.separator),
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                TbText(
                    text = "$selectedCount usage item${if (selectedCount == 1) "" else "s"} selected",
                    style = TbTheme.typography.headline,
                    color = TbTheme.colors.text,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbText(
                    text = "${formatDuration(selectedMinutes)} ready to convert",
                    style = TbTheme.typography.bodySmall,
                    color = TbTheme.colors.secondaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            TbIconButton(
                icon = Icons.Rounded.Close,
                contentDescription = "Clear selected usage",
                onClick = onClearSelection,
                variant = TbButtonVariant.Ghost,
            )
            TbButton(onClick = onCreateEntry) {
                TbIcon(
                    imageVector = Icons.Rounded.Add,
                    contentDescription = null,
                    modifier = Modifier
                        .padding(end = 6.dp)
                        .size(17.dp),
                )
                TbText("Add entry", style = TbTheme.typography.button)
            }
        }
    }
}

@Composable
private fun TimelineGridView(
    grid: TimelineGrid,
    nowMinute: Int?,
    selectedUsageIds: Set<String>,
    assignedUsageIds: Set<String>,
    usageIconLoader: UsageIconLoader,
    listState: LazyListState,
    onUsageClick: (String) -> Unit,
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        state = listState,
    ) {
        items(
            items = grid.intervalRows,
            key = { row -> row.minute },
        ) { row ->
            TimelineIntervalRow(
                grid = grid,
                row = row,
                nowMinute = nowMinute,
                selectedUsageIds = selectedUsageIds,
                assignedUsageIds = assignedUsageIds,
                usageIconLoader = usageIconLoader,
                onUsageClick = onUsageClick,
            )
        }
        item(key = "timeline-end") {
            TimelineEndBoundaryRow(row = grid.rows.last())
        }
    }
}

@Composable
private fun TimelineIntervalRow(
    grid: TimelineGrid,
    row: TimelineRow,
    nowMinute: Int?,
    selectedUsageIds: Set<String>,
    assignedUsageIds: Set<String>,
    usageIconLoader: UsageIconLoader,
    onUsageClick: (String) -> Unit,
) {
    val rowEndMinute = row.minute + grid.zoomMinutes
    val segments = remember(grid, row.minute) { timelineSegmentsForRow(grid, row) }

    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(grid.rowHeightDp.dp),
    ) {
        TimelineGridLine(
            minute = row.minute,
            emphasized = row.minute == grid.visibleStartMinute || row.minute % 60 == 0,
        )

        TimelineIntervalMarkers(
            rowStartMinute = row.minute,
            rowEndMinute = rowEndMinute,
            dpPerMinute = grid.dpPerMinute,
        )

        segments.forEach { segment ->
            val event = segment.placement.event
            TimelineEventSegmentRow(
                grid = grid,
                segment = segment,
                selected = event.id in selectedUsageIds,
                assigned = event.id in assignedUsageIds,
                usageIconLoader = usageIconLoader,
                onClick = if (event.isActive || event.sourceType == UsageSourceType.Idle) {
                    null
                } else {
                    { onUsageClick(event.id) }
                },
            )
        }

        nowMinute?.takeIf { it in row.minute until rowEndMinute }?.let { minute ->
            NowMarker(offsetDp = (minute - row.minute) * grid.dpPerMinute)
        }
    }
}

@Composable
private fun TimelineEndBoundaryRow(row: TimelineRow) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(32.dp),
    ) {
        TimelineGridLine(
            minute = row.minute,
            emphasized = true,
        )
    }
}

@Composable
private fun TimelineIntervalMarkers(
    rowStartMinute: Int,
    rowEndMinute: Int,
    dpPerMinute: Float,
) {
    val lineColor = TbTheme.colors.separator
    val offsets = remember(rowStartMinute, rowEndMinute, dpPerMinute) {
        timelineIntervalMarkerOffsets(
            rowStartMinute = rowStartMinute,
            rowEndMinute = rowEndMinute,
            dpPerMinute = dpPerMinute,
        )
    }

    offsets.forEach { offsetDp ->
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .offset(y = offsetDp.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(TimelineRuleGap),
        ) {
            Spacer(modifier = Modifier.width(TimelineTimeLabelWidth))
            Box(
                modifier = Modifier
                    .weight(1f)
                    .height(TimelineGridLineThickness)
                    .background(lineColor),
            )
        }
    }
}

@Composable
private fun TimelineGridLine(
    minute: Int,
    emphasized: Boolean,
) {
    val isHour = minute % 60 == 0
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(TimelineRuleGap),
    ) {
        TimeLabel(formatClockTime(minute), emphasized = emphasized || isHour)
        Box(
            modifier = Modifier
                .height(TimelineGridLineThickness)
                .weight(1f)
                .background(TbTheme.colors.separator),
        )
    }
}

@Composable
private fun NowMarker(offsetDp: Float) {
    val accentColor = TbTheme.colors.accent
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .offset(y = offsetDp.dp)
            .zIndex(TimelineNowMarkerLayerZIndex),
        horizontalArrangement = Arrangement.spacedBy(TimelineRuleGap),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        TbText(
            modifier = Modifier.width(TimelineTimeLabelWidth),
            text = "Now",
            style = TbTheme.typography.label,
            color = TbTheme.colors.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        Row(
            modifier = Modifier.weight(1f),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Canvas(modifier = Modifier.size(7.dp)) {
                drawCircle(color = accentColor, radius = size.minDimension / 2f)
            }
            Box(
                modifier = Modifier
                    .weight(1f)
                    .height(2.dp)
                    .background(TbTheme.colors.accent.copy(alpha = 0.74f)),
            )
        }
    }
}

@Composable
private fun TimelineEventSegmentRow(
    grid: TimelineGrid,
    segment: TimelineEventSegment,
    selected: Boolean,
    assigned: Boolean,
    usageIconLoader: UsageIconLoader,
    onClick: (() -> Unit)?,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .offset(y = segment.offsetDp.dp)
            .height(segment.heightDp.dp)
            .zIndex(TimelineEventLayerZIndex),
        horizontalArrangement = Arrangement.spacedBy(TimelineRuleGap),
        verticalAlignment = Alignment.Top,
    ) {
        Spacer(modifier = Modifier.width(TimelineTimeLabelWidth))
        Row(
            modifier = Modifier
                .weight(1f)
                .fillMaxHeight(),
            horizontalArrangement = Arrangement.spacedBy(grid.laneGapDp.dp),
        ) {
            repeat(segment.placement.laneCount) { laneIndex ->
                if (laneIndex == segment.placement.laneIndex) {
                    UsageEventCard(
                        modifier = Modifier
                            .weight(1f)
                            .fillMaxHeight(),
                        segment = segment,
                        selected = selected,
                        assigned = assigned,
                        usageIconLoader = usageIconLoader,
                        onClick = onClick,
                    )
                } else {
                    Spacer(
                        modifier = Modifier
                            .weight(1f)
                            .fillMaxHeight(),
                    )
                }
            }
        }
    }
}

@Composable
private fun UsageEventCard(
    modifier: Modifier = Modifier,
    segment: TimelineEventSegment,
    selected: Boolean,
    assigned: Boolean,
    usageIconLoader: UsageIconLoader,
    onClick: (() -> Unit)?,
) {
    val event = segment.placement.event
    val sourceColor = colorForSource(event.sourceType)
    val colors = TbTheme.colors
    val dimAssigned = assigned && !selected
    val active = event.isActive
    val height = segment.heightDp.dp
    val density = usageEventCardDensity(height)
    val shape = usageSegmentShape(segment, density)

    TbTooltip(
        text = usageEventTooltip(event),
        side = AnchorSide.Top,
        alignment = AnchorAlignment.Start,
        alignmentOffset = 12.dp,
    ) {
        TbSurface(
            modifier = modifier
                .clipToBounds()
                .alpha(if (dimAssigned) 0.66f else 1f)
                .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier),
            shape = shape,
            color = when {
                selected -> colors.accentSubtle
                active -> colors.warningSubtle
                dimAssigned -> colors.groupedSurface
                density == UsageCardDensity.Tiny -> sourceColor.copy(alpha = 0.13f)
                else -> colors.elevatedSurface
            },
            border = BorderStroke(
                width = if (selected) 1.5.dp else 1.dp,
                color = when {
                    selected -> colors.accent
                    active -> colors.warning.copy(alpha = 0.52f)
                    dimAssigned -> colors.separator.copy(alpha = 0.72f)
                    density == UsageCardDensity.Tiny -> sourceColor.copy(alpha = 0.42f)
                    else -> colors.separator
                },
            ),
            shadowElevation = if (selected || active) 2.dp else 0.dp,
        ) {
            when (density) {
                UsageCardDensity.Full -> FullUsageEventCardContent(
                    event = event,
                    sourceColor = sourceColor,
                    colors = colors,
                    assigned = assigned,
                    active = active,
                    usageIconLoader = usageIconLoader,
                    height = height,
                )

                UsageCardDensity.Medium -> MediumUsageEventCardContent(
                    event = event,
                    sourceColor = sourceColor,
                    colors = colors,
                    active = active,
                    usageIconLoader = usageIconLoader,
                )

                UsageCardDensity.Tiny -> TinyUsageEventCardContent(
                    event = event,
                    sourceColor = sourceColor,
                    colors = colors,
                )
            }
        }
    }
}

private enum class UsageCardDensity {
    Full,
    Medium,
    Tiny,
}

private fun usageEventCardDensity(height: Dp): UsageCardDensity =
    when {
        height >= 56.dp -> UsageCardDensity.Full
        height >= 26.dp -> UsageCardDensity.Medium
        else -> UsageCardDensity.Tiny
    }

@Composable
private fun usageSegmentShape(
    segment: TimelineEventSegment,
    density: UsageCardDensity,
): RoundedCornerShape {
    val radius = if (density == UsageCardDensity.Tiny) 3.dp else TbTheme.radii.card
    val joined = 2.dp
    return RoundedCornerShape(
        topStart = if (segment.startsEvent) radius else joined,
        topEnd = if (segment.startsEvent) radius else joined,
        bottomStart = if (segment.endsEvent) radius else joined,
        bottomEnd = if (segment.endsEvent) radius else joined,
    )
}

@Composable
private fun FullUsageEventCardContent(
    event: UsageEvent,
    sourceColor: Color,
    colors: TbColors,
    assigned: Boolean,
    active: Boolean,
    usageIconLoader: UsageIconLoader,
    height: Dp,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .fillMaxHeight()
            .padding(horizontal = 8.dp, vertical = 5.dp),
        horizontalArrangement = Arrangement.spacedBy(7.dp),
        verticalAlignment = Alignment.Top,
    ) {
        if (event.sourceType == UsageSourceType.Idle) {
            IdleUsageRail(color = sourceColor, height = 22.dp)
        } else {
            SourceIconBadge(
                event = event,
                iconLoader = usageIconLoader,
                name = event.sourceName,
                color = sourceColor,
                size = 24.dp,
                imageSize = 18.dp,
            )
        }

        Column(modifier = Modifier.weight(1f)) {
            OverflowTooltipText(
                text = event.title,
                style = TbTheme.typography.bodySmall.copy(fontWeight = FontWeight.SemiBold),
                maxLines = if (height < 72.dp) 1 else 2,
            )
            Row(
                modifier = Modifier.padding(top = 0.dp),
                horizontalArrangement = Arrangement.spacedBy(6.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                OverflowTooltipText(
                    text = event.sourceName,
                    style = TbTheme.typography.caption,
                    color = colors.secondaryText,
                    maxLines = 1,
                )
                UsageEventBadges(
                    assigned = assigned,
                    active = active,
                    colors = colors,
                )
            }
        }

        TbText(
            text = formatDuration(event.durationMinutes),
            style = TbTheme.typography.label,
            maxLines = 1,
        )
    }
}

@Composable
private fun MediumUsageEventCardContent(
    event: UsageEvent,
    sourceColor: Color,
    colors: TbColors,
    active: Boolean,
    usageIconLoader: UsageIconLoader,
) {
    Row(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 7.dp),
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (event.sourceType == UsageSourceType.Idle) {
            IdleUsageRail(color = sourceColor, height = 14.dp)
        } else {
            SourceIconBadge(
                event = event,
                iconLoader = usageIconLoader,
                name = event.sourceName,
                color = sourceColor,
                size = 18.dp,
                imageSize = 13.dp,
            )
        }
        OverflowTooltipText(
            modifier = Modifier.weight(1f),
            text = event.title,
            style = TbTheme.typography.label,
            maxLines = 1,
        )
        if (active) {
            TbBadge(
                label = "Active",
                color = colors.warning,
                background = colors.surface,
            )
        }
        TbText(
            text = formatDuration(event.durationMinutes),
            style = TbTheme.typography.caption,
            color = colors.secondaryText,
            maxLines = 1,
        )
    }
}

@Composable
private fun TinyUsageEventCardContent(
    event: UsageEvent,
    sourceColor: Color,
    colors: TbColors,
) {
    val tinyTextStyle = TbTheme.typography.caption.copy(
        fontSize = 11.sp,
        lineHeight = 13.sp,
        fontWeight = FontWeight.SemiBold,
    )
    Row(
        modifier = Modifier
            .fillMaxSize()
            .clipToBounds()
            .padding(horizontal = 5.dp),
        horizontalArrangement = Arrangement.spacedBy(5.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        IdleUsageRail(color = sourceColor, height = 10.dp)
        OverflowTooltipText(
            modifier = Modifier.weight(1f),
            text = event.title,
            style = tinyTextStyle,
            color = colors.text,
            maxLines = 1,
        )
        TbText(
            text = formatDuration(event.durationMinutes),
            style = tinyTextStyle,
            color = colors.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
private fun IdleUsageRail(
    color: Color,
    height: Dp,
) {
    Box(
        modifier = Modifier
            .width(3.dp)
            .height(height)
            .background(color.copy(alpha = 0.72f), RoundedCornerShape(TbTheme.radii.pill)),
    )
}

private fun usageEventTooltip(event: UsageEvent): String =
    "${event.title} · ${event.sourceName} · ${formatDuration(event.durationMinutes)}"

@Composable
private fun UsageEventBadges(
    assigned: Boolean,
    active: Boolean,
    colors: TbColors,
) {
    if (active) {
        TbBadge(
            label = "Active",
            color = colors.warning,
            background = colors.surface,
        )
    }
    if (assigned) {
        TbBadge(
            label = "Added",
            color = colors.secondaryText,
            background = colors.controlFillHover,
        )
    }
}

@Composable
private fun TimeLabel(
    label: String,
    emphasized: Boolean,
) {
    TbText(
        modifier = Modifier.width(TimelineTimeLabelWidth),
        text = label,
        style = if (emphasized) TbTheme.typography.label else TbTheme.typography.bodySmall,
        color = if (emphasized) TbTheme.colors.text else TbTheme.colors.tertiaryText,
        maxLines = 1,
        overflow = TextOverflow.Ellipsis,
    )
}

@Composable
private fun ScheduleStateMessage(
    loading: Boolean,
    unavailable: Boolean,
    modifier: Modifier = Modifier,
) {
    val title = when {
        loading -> "Loading usage"
        unavailable -> "Usage unavailable"
        else -> "No usage captured yet"
    }
    val message = when {
        loading -> "Waiting for the sidecar to report today's activity."
        unavailable -> "The schedule will fill in once the sidecar is connected."
        else -> "New app activity will appear here as sessions close."
    }

    TbCard(
        modifier = modifier.widthIn(max = 360.dp),
        color = TbTheme.colors.surface.copy(alpha = 0.94f),
        shadowElevation = 2.dp,
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(6.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            TbText(
                text = title,
                style = TbTheme.typography.title2,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            TbText(
                text = message,
                style = TbTheme.typography.bodySmall,
                color = TbTheme.colors.secondaryText,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

@Composable
private fun SourceIconBadge(
    event: UsageEvent,
    iconLoader: UsageIconLoader,
    name: String,
    color: Color,
    size: Dp,
    imageSize: Dp,
) {
    val icon by produceState<ImageBitmap?>(
        initialValue = null,
        key1 = event.applicationIdentity,
        key2 = iconLoader,
    ) {
        value = null
        val identity = event.applicationIdentity ?: return@produceState
        if (event.sourceType != UsageSourceType.Idle) {
            value = iconLoader.loadIcon(identity)
        }
    }

    TbSurface(
        modifier = Modifier
            .size(size)
            .clip(RoundedCornerShape(TbTheme.radii.card))
            .border(Dp.Hairline, color.copy(alpha = 0.20f), RoundedCornerShape(TbTheme.radii.card)),
        shape = RoundedCornerShape(TbTheme.radii.card),
        color = if (icon != null) TbTheme.colors.surface else color.copy(alpha = 0.12f),
        contentColor = color,
        contentAlignment = Alignment.Center,
    ) {
        if (icon != null) {
            Image(
                bitmap = icon!!,
                contentDescription = "$name icon",
                modifier = Modifier
                    .size(imageSize)
                    .clip(RoundedCornerShape(6.dp)),
            )
        } else {
            TbText(
                text = name.take(1).uppercase(),
                style = TbTheme.typography.label,
                color = color,
            )
        }
    }
}

private suspend fun scrollTimelineToDp(
    listState: LazyListState,
    grid: TimelineGrid,
    targetDp: Float,
    density: Density,
    animated: Boolean,
) {
    if (grid.intervalRows.isEmpty()) return

    val maxTargetDp = (grid.contentHeightDp - 0.001f).coerceAtLeast(0f)
    val clampedTargetDp = targetDp.coerceIn(0f, maxTargetDp)
    val rowIndex = (clampedTargetDp / grid.rowHeightDp)
        .toInt()
        .coerceIn(0, grid.intervalRows.lastIndex)
    val rowOffsetDp = clampedTargetDp - rowIndex * grid.rowHeightDp
    val rowOffsetPx = with(density) { rowOffsetDp.dp.roundToPx() }

    if (animated) {
        listState.animateScrollToItem(rowIndex, rowOffsetPx)
    } else {
        listState.scrollToItem(rowIndex, rowOffsetPx)
    }
}

@OptIn(ExperimentalTime::class)
private fun currentMinuteForSelectedDay(state: TimeboxxingScreenState): Int? {
    val now = Clock.System.now().toEpochMilliseconds()
    if (now !in state.selectedDay.startedAtEpochMillis until state.selectedDay.endedAtEpochMillis) {
        return null
    }
    return ((now - state.selectedDay.startedAtEpochMillis) / 60_000L).toInt().coerceIn(0, 24 * 60)
}

private fun colorForSource(sourceType: UsageSourceType): Color =
    when (sourceType) {
        UsageSourceType.Application -> Color(0xFF64748B)
        UsageSourceType.Document -> Color(0xFF0A84FF)
        UsageSourceType.Email -> Color(0xFFFF3B30)
        UsageSourceType.Meeting -> Color(0xFF5856D6)
        UsageSourceType.Spreadsheet -> Color(0xFF34C759)
        UsageSourceType.Messaging -> Color(0xFF00A4C7)
        UsageSourceType.Browser -> Color(0xFFFF9F0A)
        UsageSourceType.Presentation -> Color(0xFFFF6B1A)
        UsageSourceType.Idle -> Color(0xFF6B7280)
    }
