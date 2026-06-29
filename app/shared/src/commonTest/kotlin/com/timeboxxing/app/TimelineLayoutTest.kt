package com.timeboxxing.app

import androidx.compose.ui.graphics.Color
import com.timeboxxing.domain.model.AmaAnswer
import com.timeboxxing.domain.model.AmaAppUsageBucket
import com.timeboxxing.domain.model.AmaAppUsageChart
import com.timeboxxing.domain.model.AmaIndexState
import com.timeboxxing.domain.model.AmaIndexStatus
import com.timeboxxing.domain.model.AmaMessageRole
import com.timeboxxing.domain.model.AmaSource
import com.timeboxxing.data.mock.mockTimeboxxingData
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.EntryMode
import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.UsageEvent
import com.timeboxxing.domain.model.UsageSourceType
import com.timeboxxing.domain.model.formatClockTime
import com.timeboxxing.domain.model.formatDuration
import com.timeboxxing.app.presentation.TimeboxxingAction
import com.timeboxxing.app.presentation.TimeboxxingSection
import com.timeboxxing.app.presentation.createInitialTimeboxxingState
import com.timeboxxing.app.presentation.reduceTimeboxxingState
import com.timeboxxing.app.ui.buildTimelineGrid
import com.timeboxxing.app.ui.timelineFirstEventScrollDp
import com.timeboxxing.app.ui.timelineDpPerMinuteForZoom
import com.timeboxxing.app.ui.timelineEntryScrollDp
import com.timeboxxing.app.ui.timelineMinuteOffsetDp
import com.timeboxxing.app.ui.timelineNowScrollDirection
import com.timeboxxing.app.ui.timelineRowHeightDpForZoom
import com.timeboxxing.app.ui.scheduleTimelineUsageEvents
import com.timeboxxing.app.ui.timelineScrollToMinuteDp
import com.timeboxxing.app.ui.timelineSegmentsForRow
import com.timeboxxing.app.ui.timelineIntervalMarkerOffsets
import com.timeboxxing.app.ui.SchedulePaneAutoScrollState
import com.timeboxxing.app.ui.TimelineScrollDirection
import com.timeboxxing.app.ui.TbDarkColors
import com.timeboxxing.app.ui.TbLightColors
import com.timeboxxing.app.ui.TimelineEventLayerZIndex
import com.timeboxxing.app.ui.TimelineNowMarkerLayerZIndex
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue
import kotlin.math.abs
import kotlin.math.max
import kotlin.math.min
import kotlin.math.pow

class TimelineLayoutTest {

    @Test
    fun scheduleInitialAutoScrollIsOneShotForSameKey() {
        val scrollState = SchedulePaneAutoScrollState()

        assertTrue(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 15, targetDp = 42f))
        assertFalse(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 15, targetDp = 42f))
    }

    @Test
    fun scheduleInitialAutoScrollRearmsForDifferentDay() {
        val scrollState = SchedulePaneAutoScrollState()

        assertTrue(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 15, targetDp = 42f))
        assertTrue(scrollState.shouldHandleInitialScroll(2L, zoomMinutes = 15, targetDp = 42f))
        assertFalse(scrollState.shouldHandleInitialScroll(2L, zoomMinutes = 15, targetDp = 42f))
    }

    @Test
    fun scheduleInitialAutoScrollRearmsForZoomOrTargetChange() {
        val scrollState = SchedulePaneAutoScrollState()

        assertTrue(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 15, targetDp = 42f))
        assertTrue(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 30, targetDp = 42f))
        assertTrue(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 30, targetDp = 84f))
        assertFalse(scrollState.shouldHandleInitialScroll(1L, zoomMinutes = 30, targetDp = 84f))
    }

    @Test
    fun scheduleFocusedEntryAutoScrollIsOneShotForSameKey() {
        val scrollState = SchedulePaneAutoScrollState()

        assertTrue(
            scrollState.shouldHandleFocusedEntryScroll(
                entryId = "entry-1",
                startMinute = 10 * 60,
                sourceUsageIds = setOf("usage-1"),
                zoomMinutes = 15,
                targetDp = 120f,
            ),
        )
        assertFalse(
            scrollState.shouldHandleFocusedEntryScroll(
                entryId = "entry-1",
                startMinute = 10 * 60,
                sourceUsageIds = setOf("usage-1"),
                zoomMinutes = 15,
                targetDp = 120f,
            ),
        )
    }

    @Test
    fun scheduleFocusedEntryAutoScrollRearmsForDifferentEntry() {
        val scrollState = SchedulePaneAutoScrollState()

        assertTrue(
            scrollState.shouldHandleFocusedEntryScroll(
                entryId = "entry-1",
                startMinute = 10 * 60,
                sourceUsageIds = setOf("usage-1"),
                zoomMinutes = 15,
                targetDp = 120f,
            ),
        )
        assertTrue(
            scrollState.shouldHandleFocusedEntryScroll(
                entryId = "entry-2",
                startMinute = 11 * 60,
                sourceUsageIds = setOf("usage-2"),
                zoomMinutes = 15,
                targetDp = 240f,
            ),
        )
        assertFalse(
            scrollState.shouldHandleFocusedEntryScroll(
                entryId = "entry-2",
                startMinute = 11 * 60,
                sourceUsageIds = setOf("usage-2"),
                zoomMinutes = 15,
                targetDp = 240f,
            ),
        )
    }

    @Test
    fun timelineGridUsesFullDayRowsForSupportedZoomLevels() {
        val expected = listOf(
            10 to 280f,
            15 to 420f,
            30 to 840f,
            60 to 1680f,
        )

        expected.forEach { (zoomMinutes, rowHeightDp) ->
            val grid = buildTimelineGrid(
                events = emptyList(),
                zoomMinutes = zoomMinutes,
            )

            assertEquals(0, grid.visibleStartMinute)
            assertEquals(24 * 60, grid.visibleEndMinute)
            assertEquals((24 * 60) / zoomMinutes, grid.intervalCount)
            assertWithin(rowHeightDp, grid.rowHeightDp)
            assertWithin(rowHeightDp / zoomMinutes, grid.dpPerMinute)
            assertEquals(listOf(0, zoomMinutes, zoomMinutes * 2), grid.rows.take(3).map { it.minute })
            assertEquals(24 * 60, grid.rows.last().minute)
        }
    }

    @Test
    fun timelineIntervalMarkerOffsetsUseFiveMinuteCadenceWithoutBoundaryDuplicates() {
        val rowStartMinute = 8 * 60
        val cases = listOf(
            60 to listOf(5f, 10f, 15f, 20f, 25f, 30f, 35f, 40f, 45f, 50f, 55f),
            30 to listOf(5f, 10f, 15f, 20f, 25f),
            15 to listOf(5f, 10f),
            10 to listOf(5f),
        )

        cases.forEach { (zoomMinutes, expectedOffsets) ->
            assertEquals(
                expectedOffsets,
                timelineIntervalMarkerOffsets(
                    rowStartMinute = rowStartMinute,
                    rowEndMinute = rowStartMinute + zoomMinutes,
                    dpPerMinute = 1f,
                ),
            )
        }
    }

    @Test
    fun timelineOneMinuteCompletedEventUsesExactScaledHeight() {
        val grid = buildTimelineGrid(
            events = listOf(usageEvent(durationMinutes = 1)),
            zoomMinutes = 15,
        )
        val placement = grid.placements.single()

        assertWithin(28f, timelineDpPerMinuteForZoom(15))
        assertWithin(timelineDpPerMinuteForZoom(15), placement.heightDp)
        assertWithin(10 * 60 * timelineDpPerMinuteForZoom(15), placement.timeTopDp)
        assertWithin(placement.timeTopDp, placement.displayTopDp)
    }

    @Test
    fun timelineTwoOneMinuteEventsStayInsideFifteenMinuteInterval() {
        val events = listOf(
            usageEvent(id = "usage-one", startMinute = 11 * 60, durationMinutes = 1),
            usageEvent(id = "usage-two", startMinute = 11 * 60 + 1, durationMinutes = 1),
        )
        val grid = buildTimelineGrid(
            events = events,
            zoomMinutes = 15,
        )
        val placements = grid.placements
        val firstTopDp = placements.first().displayTopDp
        val lastBottomDp = placements.last().displayTopDp + placements.last().heightDp

        assertWithin(15f * timelineDpPerMinuteForZoom(15), grid.rowHeightDp)
        assertWithin(2f * timelineDpPerMinuteForZoom(15), lastBottomDp - firstTopDp)
        assertTrue(lastBottomDp - firstTopDp < grid.rowHeightDp)
    }

    @Test
    fun timelineTenSequentialSixMinuteEventsOccupyOneScaledHour() {
        val events = (0 until 10).map { index ->
            usageEvent(
                id = "usage-$index",
                startMinute = 10 * 60 + index * 6,
                durationMinutes = 6,
            )
        }
        val grid = buildTimelineGrid(
            events = events,
            zoomMinutes = 15,
        )
        val placements = grid.placements
        val firstTopDp = placements.first().displayTopDp
        val lastBottomDp = placements.last().displayTopDp + placements.last().heightDp

        assertWithin(60f * timelineDpPerMinuteForZoom(15), lastBottomDp - firstTopDp)
    }

    @Test
    fun timelineLongerEventScalesWithSelectedZoomDensity() {
        val grid = buildTimelineGrid(
            events = listOf(usageEvent(durationMinutes = 30)),
            zoomMinutes = 60,
        )

        assertWithin(840f, grid.placements.single().heightDp)
    }

    @Test
    fun timelineOneMinuteEventAtHourlyZoomUsesReadableScale() {
        val grid = buildTimelineGrid(
            events = listOf(usageEvent(durationMinutes = 1)),
            zoomMinutes = 60,
        )

        assertWithin(28f, timelineDpPerMinuteForZoom(60))
        assertWithin(28f, grid.placements.single().heightDp)
    }

    @Test
    fun timelineActiveEventUsesExactScaledHeight() {
        val grid = buildTimelineGrid(
            events = listOf(usageEvent(durationMinutes = 1, isActive = true)),
            zoomMinutes = 15,
        )

        assertWithin(timelineDpPerMinuteForZoom(15), grid.placements.single().heightDp)
    }

    @Test
    fun scheduleTimelineExcludesActiveEventsButKeepsCompletedAndIdle() {
        val active = usageEvent(id = "active", durationMinutes = 3, isActive = true)
        val completed = usageEvent(id = "completed", durationMinutes = 5)
        val idle = usageEvent(
            id = "idle",
            durationMinutes = 5,
            sourceType = UsageSourceType.Idle,
        )

        val visibleEvents = scheduleTimelineUsageEvents(listOf(active, completed, idle))
        val grid = buildTimelineGrid(
            events = visibleEvents,
            zoomMinutes = 15,
        )

        assertEquals(listOf("completed", "idle"), visibleEvents.map { it.id })
        assertEquals(setOf("completed", "idle"), grid.placements.map { it.event.id }.toSet())
    }

    @Test
    fun timelineNowMarkerDrawsAboveUsageEvents() {
        assertTrue(TimelineNowMarkerLayerZIndex > TimelineEventLayerZIndex)
    }

    @Test
    fun timelineIdleEventUsesSameScaledPlacementAsApplicationUsage() {
        val grid = buildTimelineGrid(
            events = listOf(
                usageEvent(
                    id = "idle",
                    durationMinutes = 5,
                    sourceType = UsageSourceType.Idle,
                ),
            ),
            zoomMinutes = 10,
        )
        val placement = grid.placements.single()

        assertEquals(UsageSourceType.Idle, placement.event.sourceType)
        assertWithin(5f * timelineDpPerMinuteForZoom(10), placement.heightDp)
    }

    @Test
    fun timelineEventCrossingRowsIsSplitIntoRowSegments() {
        val event = usageEvent(
            id = "cross-row",
            startMinute = 10 * 60 + 50,
            durationMinutes = 20,
        )
        val grid = buildTimelineGrid(
            events = listOf(event),
            zoomMinutes = 15,
        )
        val firstRow = grid.rows.first { it.minute == 10 * 60 + 45 }
        val secondRow = grid.rows.first { it.minute == 11 * 60 }
        val firstSegment = timelineSegmentsForRow(grid, firstRow).single()
        val secondSegment = timelineSegmentsForRow(grid, secondRow).single()

        assertWithin(5f * grid.dpPerMinute, firstSegment.offsetDp)
        assertWithin(10f * grid.dpPerMinute, firstSegment.heightDp)
        assertTrue(firstSegment.startsEvent)
        assertFalse(firstSegment.endsEvent)

        assertWithin(0f, secondSegment.offsetDp)
        assertWithin(10f * grid.dpPerMinute, secondSegment.heightDp)
        assertFalse(secondSegment.startsEvent)
        assertTrue(secondSegment.endsEvent)
    }

    @Test
    fun timelineBackToBackEventsDoNotReceiveArtificialGaps() {
        val events = listOf(
            usageEvent(id = "first", startMinute = 11 * 60, durationMinutes = 1),
            usageEvent(id = "second", startMinute = 11 * 60 + 1, durationMinutes = 1),
        )
        val grid = buildTimelineGrid(
            events = events,
            zoomMinutes = 15,
        )
        val first = grid.placements.first { it.event.id == "first" }
        val second = grid.placements.first { it.event.id == "second" }

        assertWithin(first.displayTopDp + first.heightDp, second.displayTopDp)
        assertEquals(1, first.laneCount)
        assertEquals(1, second.laneCount)
    }

    @Test
    fun timelineOverlappingEventsUseHorizontalLanesWithoutMovingTime() {
        val events = listOf(
            usageEvent(id = "first", startMinute = 11 * 60, durationMinutes = 10),
            usageEvent(id = "second", startMinute = 11 * 60 + 5, durationMinutes = 10),
        )
        val grid = buildTimelineGrid(
            events = events,
            zoomMinutes = 15,
        )
        val first = grid.placements.first { it.event.id == "first" }
        val second = grid.placements.first { it.event.id == "second" }

        assertWithin(first.timeTopDp, first.displayTopDp)
        assertWithin(second.timeTopDp, second.displayTopDp)
        assertEquals(0, first.laneIndex)
        assertEquals(1, second.laneIndex)
        assertEquals(2, first.laneCount)
        assertEquals(2, second.laneCount)
    }

    @Test
    fun timelineRowHeightHelperUsesReadableMinuteScale() {
        assertWithin(280f, timelineRowHeightDpForZoom(10))
        assertWithin(420f, timelineRowHeightDpForZoom(15))
        assertWithin(840f, timelineRowHeightDpForZoom(30))
        assertWithin(1680f, timelineRowHeightDpForZoom(60))
        assertWithin(28f, timelineDpPerMinuteForZoom(10))
        assertWithin(28f, timelineDpPerMinuteForZoom(15))
        assertWithin(28f, timelineDpPerMinuteForZoom(30))
        assertWithin(28f, timelineDpPerMinuteForZoom(60))
    }

    @Test
    fun timelineFirstEventScrollUsesFixedVisualContext() {
        val grid = buildTimelineGrid(
            events = mockTimeboxxingData().usageEvents,
            zoomMinutes = 15,
        )
        val firstPlacement = grid.placements.first()

        assertWithin(firstPlacement.displayTopDp - 48f, timelineFirstEventScrollDp(grid)!!)
    }

    @Test
    fun timelineFirstEventScrollKeepsHourlyZoomFirstEventVisible() {
        val grid = buildTimelineGrid(
            events = mockTimeboxxingData().usageEvents,
            zoomMinutes = 60,
        )
        val firstPlacement = grid.placements.first()
        val target = timelineFirstEventScrollDp(grid)!!

        assertWithin(firstPlacement.displayTopDp - 48f, target)
        assertTrue(firstPlacement.displayTopDp - target < 600f)
    }

    @Test
    fun timelineFirstEventScrollClampsEarlyEventsToTop() {
        val grid = buildTimelineGrid(
            events = listOf(usageEvent(startMinute = 1, durationMinutes = 1)),
            zoomMinutes = 60,
        )

        assertWithin(0f, timelineFirstEventScrollDp(grid)!!)
    }

    @Test
    fun timelineFirstEventScrollReturnsNullWithoutEvents() {
        val grid = buildTimelineGrid(
            events = emptyList(),
            zoomMinutes = 15,
        )

        assertEquals(null, timelineFirstEventScrollDp(grid))
    }

    @Test
    fun timelineNowScrollDirectionPointsDownWhenNowIsBelowViewport() {
        assertEquals(
            TimelineScrollDirection.Down,
            timelineNowScrollDirection(
                nowOffsetDp = 680f,
                viewportTopDp = 100f,
                viewportHeightDp = 600f,
            ),
        )
    }

    @Test
    fun timelineNowScrollDirectionPointsUpWhenNowIsAboveViewport() {
        assertEquals(
            TimelineScrollDirection.Up,
            timelineNowScrollDirection(
                nowOffsetDp = 90f,
                viewportTopDp = 100f,
                viewportHeightDp = 600f,
            ),
        )
    }

    @Test
    fun timelineNowScrollDirectionIsNullWhenNowIsVisible() {
        assertEquals(
            null,
            timelineNowScrollDirection(
                nowOffsetDp = 500f,
                viewportTopDp = 100f,
                viewportHeightDp = 600f,
            ),
        )
    }

    @Test
    fun timelineScrollToMinuteCentersTargetAndClampsToDayBounds() {
        val grid = buildTimelineGrid(
            events = emptyList(),
            zoomMinutes = 60,
        )

        assertWithin(12 * 60 * grid.dpPerMinute, timelineMinuteOffsetDp(grid, 12 * 60))
        assertWithin(
            timelineMinuteOffsetDp(grid, 12 * 60) - 300f,
            timelineScrollToMinuteDp(grid, 12 * 60, viewportHeightDp = 600f),
        )
        assertWithin(0f, timelineScrollToMinuteDp(grid, 1, viewportHeightDp = 600f))
        assertWithin(
            (grid.contentHeightDp + 24f - 600f).coerceAtLeast(0f),
            timelineScrollToMinuteDp(grid, 24 * 60, viewportHeightDp = 600f),
        )
    }

    @Test
    fun timelineEntryScrollStartsNearEntryTimeWithContext() {
        val grid = buildTimelineGrid(
            events = emptyList(),
            zoomMinutes = 15,
        )
        val entry = timeEntry(startMinute = 9 * 60)

        assertWithin(
            9 * 60 * grid.dpPerMinute - 2 * grid.rowHeightDp,
            timelineEntryScrollDp(grid, entry, viewportHeightDp = 600f),
        )
    }

    @Test
    fun timelineEntryScrollPrefersAssignedUsagePlacement() {
        val grid = buildTimelineGrid(
            events = mockTimeboxxingData().usageEvents,
            zoomMinutes = 15,
        )
        val sourcePlacement = grid.placements.first { it.event.id == "usage-daven-chat" }
        val entry = timeEntry(
            startMinute = 9 * 60,
            sourceUsageIds = setOf("usage-daven-chat"),
        )

        assertWithin(
            sourcePlacement.displayTopDp - 2 * grid.rowHeightDp,
            timelineEntryScrollDp(grid, entry, viewportHeightDp = 600f),
        )
    }

    @Test
    fun timelineEntryScrollFallsBackToStartTimeForUnknownSourceUsage() {
        val grid = buildTimelineGrid(
            events = mockTimeboxxingData().usageEvents,
            zoomMinutes = 15,
        )
        val entry = timeEntry(
            startMinute = 9 * 60,
            sourceUsageIds = setOf("missing-usage"),
        )

        assertWithin(
            9 * 60 * grid.dpPerMinute - 2 * grid.rowHeightDp,
            timelineEntryScrollDp(grid, entry, viewportHeightDp = 600f),
        )
    }

    @Test
    fun timelineEntryScrollClampsEarlyEntriesToTop() {
        val grid = buildTimelineGrid(
            events = emptyList(),
            zoomMinutes = 15,
        )
        val entry = timeEntry(startMinute = 1)

        assertWithin(0f, timelineEntryScrollDp(grid, entry, viewportHeightDp = 600f))
    }

    @Test
    fun timelineEntryScrollClampsLateEntriesToMaximumScroll() {
        val grid = buildTimelineGrid(
            events = emptyList(),
            zoomMinutes = 15,
        )
        val entry = timeEntry(startMinute = 23 * 60 + 59)

        assertWithin(
            (grid.contentHeightDp + 24f - 600f).coerceAtLeast(0f),
            timelineEntryScrollDp(grid, entry, viewportHeightDp = 600f, contextRows = 0),
        )
    }

    @Test
    fun timelineEntryScrollClampsLateEntriesToProvidedMaximumScroll() {
        val grid = buildTimelineGrid(
            events = emptyList(),
            zoomMinutes = 15,
        )
        val entry = timeEntry(startMinute = 23 * 60 + 50)

        assertWithin(500f, timelineEntryScrollDp(grid, entry, viewportHeightDp = 600f, maxScrollDp = 500f))
    }

    private fun createAmaConfiguredState() =
        createInitialTimeboxxingState().copy(
            aiSettings = AiSettings(openRouterSecretExists = true),
        )

    private fun timeEntry(
        startMinute: Int,
        sourceUsageIds: Set<String> = emptySet(),
    ): TimeEntry =
        TimeEntry(
            id = "entry",
            projectId = "project",
            title = "Entry",
            notes = "",
            startMinute = startMinute,
            durationMinutes = 30,
            billable = true,
            sourceUsageIds = sourceUsageIds,
        )

    private fun usageEvent(
        durationMinutes: Int,
        isActive: Boolean = false,
        id: String = if (isActive) "sidecar-active" else "usage",
        startMinute: Int = 10 * 60,
        sourceType: UsageSourceType = UsageSourceType.Application,
    ): UsageEvent =
        UsageEvent(
            id = id,
            title = "Usage",
            sourceName = "App",
            sourceType = sourceType,
            startMinute = startMinute,
            durationMinutes = durationMinutes,
            projectHintId = null,
            isActive = isActive,
        )

    private fun activeUsageEvent(durationMinutes: Int = 1): UsageEvent =
        UsageEvent(
            id = "sidecar-active",
            title = "Client dashboard",
            sourceName = "Google Chrome",
            sourceType = UsageSourceType.Browser,
            startMinute = 10 * 60,
            durationMinutes = durationMinutes,
            projectHintId = null,
            isActive = true,
        )

    private fun assertWithin(
        expected: Float,
        actual: Float,
    ) {
        assertTrue(abs(expected - actual) < 0.001f, "Expected $actual to be within 0.001 of $expected")
    }
}
