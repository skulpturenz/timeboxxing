package com.timeboxxing.app.ui

import com.timeboxxing.app.model.TimeEntry
import com.timeboxxing.app.model.UsageEvent

const val TimelineDayMinutes = 24 * 60
const val TimelineLaneGapDp = 4f
const val UsageTimelineDpPerMinute = 28f
const val TimelineInitialScrollContextDp = 48f

data class TimelineGrid(
    val visibleStartMinute: Int,
    val visibleEndMinute: Int,
    val zoomMinutes: Int,
    val rowHeightDp: Float,
    val laneGapDp: Float,
    val rows: List<TimelineRow>,
    val placements: List<TimelineEventPlacement>,
) {
    val visibleMinutes: Int
        get() = visibleEndMinute - visibleStartMinute

    val intervalCount: Int
        get() = visibleMinutes / zoomMinutes

    val intervalRows: List<TimelineRow>
        get() = rows.dropLast(1)

    val dpPerMinute: Float
        get() = rowHeightDp / zoomMinutes.toFloat()

    val timelineHeightDp: Float
        get() = intervalCount * rowHeightDp

    val contentHeightDp: Float
        get() = timelineHeightDp
}

data class TimelineRow(
    val minute: Int,
    val offsetDp: Float,
)

data class TimelineEventPlacement(
    val event: UsageEvent,
    val startMinute: Int,
    val endMinute: Int,
    val timeTopDp: Float,
    val displayTopDp: Float,
    val heightDp: Float,
    val laneIndex: Int,
    val laneCount: Int,
)

data class TimelineEventSegment(
    val placement: TimelineEventPlacement,
    val row: TimelineRow,
    val offsetDp: Float,
    val heightDp: Float,
    val startsEvent: Boolean,
    val endsEvent: Boolean,
)

enum class TimelineScrollDirection {
    Up,
    Down,
}

fun timelineRowHeightDpForZoom(zoomMinutes: Int): Float =
    normalizeTimelineZoomMinutes(zoomMinutes) * UsageTimelineDpPerMinute

fun timelineDpPerMinuteForZoom(zoomMinutes: Int): Float =
    timelineRowHeightDpForZoom(zoomMinutes) / normalizeTimelineZoomMinutes(zoomMinutes).toFloat()

fun buildTimelineGrid(
    events: List<UsageEvent>,
    zoomMinutes: Int,
    rowHeightDp: Float? = null,
    laneGapDp: Float = TimelineLaneGapDp,
): TimelineGrid {
    val normalizedZoom = normalizeTimelineZoomMinutes(zoomMinutes)
    val resolvedRowHeightDp = rowHeightDp ?: timelineRowHeightDpForZoom(normalizedZoom)
    val intervalCount = TimelineDayMinutes / normalizedZoom
    val rows = (0..intervalCount).map { index ->
        TimelineRow(
            minute = index * normalizedZoom,
            offsetDp = index * resolvedRowHeightDp,
        )
    }
    val dpPerMinute = resolvedRowHeightDp / normalizedZoom.toFloat()
    val placements = resolveTimelineEventPlacements(
        events = events,
        dpPerMinute = dpPerMinute,
    )

    return TimelineGrid(
        visibleStartMinute = 0,
        visibleEndMinute = TimelineDayMinutes,
        zoomMinutes = normalizedZoom,
        rowHeightDp = resolvedRowHeightDp,
        laneGapDp = laneGapDp,
        rows = rows,
        placements = placements,
    )
}

fun timelineSegmentsForRow(
    grid: TimelineGrid,
    row: TimelineRow,
): List<TimelineEventSegment> {
    val rowStartMinute = row.minute
    val rowEndMinute = minOf(rowStartMinute + grid.zoomMinutes, grid.visibleEndMinute)
    if (rowEndMinute <= rowStartMinute) return emptyList()

    return grid.placements.mapNotNull { placement ->
        val segmentStartMinute = maxOf(placement.startMinute, rowStartMinute)
        val segmentEndMinute = minOf(placement.endMinute, rowEndMinute)
        if (segmentEndMinute <= segmentStartMinute) return@mapNotNull null

        TimelineEventSegment(
            placement = placement,
            row = row,
            offsetDp = (segmentStartMinute - rowStartMinute) * grid.dpPerMinute,
            heightDp = (segmentEndMinute - segmentStartMinute) * grid.dpPerMinute,
            startsEvent = segmentStartMinute == placement.startMinute,
            endsEvent = segmentEndMinute == placement.endMinute,
        )
    }.sortedWith(
        compareBy<TimelineEventSegment> { it.offsetDp }
            .thenBy { it.placement.laneIndex },
    )
}

fun timelineIntervalMarkerOffsets(
    rowStartMinute: Int,
    rowEndMinute: Int,
    dpPerMinute: Float,
    stepMinutes: Int = 5,
): List<Float> {
    if (rowEndMinute <= rowStartMinute || stepMinutes <= 0 || dpPerMinute <= 0f) {
        return emptyList()
    }

    return generateSequence(rowStartMinute + stepMinutes) { minute -> minute + stepMinutes }
        .takeWhile { minute -> minute < rowEndMinute }
        .map { minute -> (minute - rowStartMinute) * dpPerMinute }
        .toList()
}

fun timelineFirstEventScrollDp(
    grid: TimelineGrid,
    contextDp: Float = TimelineInitialScrollContextDp,
): Float? {
    val firstPlacementTop = grid.placements.firstOrNull()?.displayTopDp ?: return null
    return (firstPlacementTop - contextDp.coerceAtLeast(0f)).coerceAtLeast(0f)
}

fun timelineMinuteOffsetDp(
    grid: TimelineGrid,
    minute: Int,
): Float =
    minute
        .coerceIn(grid.visibleStartMinute, grid.visibleEndMinute)
        .minus(grid.visibleStartMinute)
        .times(grid.dpPerMinute)

fun timelineScrollToMinuteDp(
    grid: TimelineGrid,
    minute: Int,
    viewportHeightDp: Float,
    bottomPaddingDp: Float = 24f,
): Float {
    val minuteOffsetDp = timelineMinuteOffsetDp(grid, minute)
    val resolvedMaxScrollDp = (grid.contentHeightDp + bottomPaddingDp - viewportHeightDp).coerceAtLeast(0f)

    return (minuteOffsetDp - viewportHeightDp / 2f).coerceIn(0f, resolvedMaxScrollDp)
}

fun timelineNowScrollDirection(
    nowOffsetDp: Float,
    viewportTopDp: Float,
    viewportHeightDp: Float,
    thresholdDp: Float = 24f,
): TimelineScrollDirection? {
    val threshold = thresholdDp.coerceAtLeast(0f)
    val viewportBottomDp = viewportTopDp + viewportHeightDp

    return when {
        nowOffsetDp < viewportTopDp + threshold -> TimelineScrollDirection.Up
        nowOffsetDp > viewportBottomDp - threshold -> TimelineScrollDirection.Down
        else -> null
    }
}

fun timelineEntryScrollDp(
    grid: TimelineGrid,
    entry: TimeEntry,
    viewportHeightDp: Float,
    maxScrollDp: Float? = null,
    contextRows: Int = 2,
    bottomPaddingDp: Float = 24f,
): Float {
    val usagePlacementTopDp = entry.sourceUsageIds
        .mapNotNull { usageId -> grid.placements.firstOrNull { it.event.id == usageId } }
        .minByOrNull { it.displayTopDp }
        ?.displayTopDp
    val entryStartMinute = entry.startMinute.coerceIn(grid.visibleStartMinute, grid.visibleEndMinute)
    val entryTopDp = usagePlacementTopDp
        ?: (entryStartMinute - grid.visibleStartMinute) * grid.dpPerMinute
    val resolvedMaxScrollDp = maxScrollDp ?: (grid.contentHeightDp + bottomPaddingDp - viewportHeightDp).coerceAtLeast(0f)

    return (entryTopDp - timelineScrollContextDp(grid, contextRows)).coerceIn(0f, resolvedMaxScrollDp)
}

private data class TimelineInterval(
    val event: UsageEvent,
    val startMinute: Int,
    val endMinute: Int,
)

private data class ActiveTimelineInterval(
    val endMinute: Int,
    val laneIndex: Int,
)

private fun normalizeTimelineZoomMinutes(zoomMinutes: Int): Int =
    when {
        zoomMinutes <= 10 -> 10
        zoomMinutes <= 15 -> 15
        zoomMinutes <= 30 -> 30
        else -> 60
    }

private fun resolveTimelineEventPlacements(
    events: List<UsageEvent>,
    dpPerMinute: Float,
): List<TimelineEventPlacement> {
    val intervals = events.mapNotNull { event ->
        val startMinute = event.startMinute.coerceIn(0, TimelineDayMinutes)
        val endMinute = (event.startMinute + event.durationMinutes).coerceIn(0, TimelineDayMinutes)
        if (endMinute <= startMinute) return@mapNotNull null
        TimelineInterval(
            event = event,
            startMinute = startMinute,
            endMinute = endMinute,
        )
    }.sortedWith(
        compareBy<TimelineInterval> { it.startMinute }
            .thenBy { it.endMinute }
            .thenBy { it.event.id },
    )

    val placements = mutableListOf<TimelineEventPlacement>()
    var index = 0
    while (index < intervals.size) {
        val group = mutableListOf<TimelineInterval>()
        var groupEndMinute = intervals[index].endMinute
        while (index < intervals.size && intervals[index].startMinute < groupEndMinute) {
            val interval = intervals[index]
            group += interval
            groupEndMinute = maxOf(groupEndMinute, interval.endMinute)
            index += 1
        }
        placements += assignTimelineLanes(group, dpPerMinute)
    }

    return placements.sortedWith(
        compareBy<TimelineEventPlacement> { it.timeTopDp }
            .thenBy { it.laneIndex }
            .thenBy { it.event.id },
    )
}

private fun assignTimelineLanes(
    intervals: List<TimelineInterval>,
    dpPerMinute: Float,
): List<TimelineEventPlacement> {
    val active = mutableListOf<ActiveTimelineInterval>()
    val assigned = mutableListOf<Pair<TimelineInterval, Int>>()
    var laneCount = 0

    intervals.forEach { interval ->
        active.removeAll { it.endMinute <= interval.startMinute }
        val usedLanes = active.map { it.laneIndex }.toSet()
        var laneIndex = 0
        while (laneIndex in usedLanes) {
            laneIndex += 1
        }
        active += ActiveTimelineInterval(
            endMinute = interval.endMinute,
            laneIndex = laneIndex,
        )
        assigned += interval to laneIndex
        laneCount = maxOf(laneCount, laneIndex + 1)
    }

    return assigned.map { (interval, laneIndex) ->
        val timeTopDp = interval.startMinute * dpPerMinute
        TimelineEventPlacement(
            event = interval.event,
            startMinute = interval.startMinute,
            endMinute = interval.endMinute,
            timeTopDp = timeTopDp,
            displayTopDp = timeTopDp,
            heightDp = (interval.endMinute - interval.startMinute) * dpPerMinute,
            laneIndex = laneIndex,
            laneCount = laneCount,
        )
    }
}

private fun timelineScrollContextDp(
    grid: TimelineGrid,
    contextRows: Int,
): Float = contextRows.coerceAtLeast(0) * grid.rowHeightDp
