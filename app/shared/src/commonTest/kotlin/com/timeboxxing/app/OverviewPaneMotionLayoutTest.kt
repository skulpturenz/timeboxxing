package com.timeboxxing.app

import androidx.compose.ui.geometry.Rect
import com.timeboxxing.app.ui.OverviewPane
import com.timeboxxing.app.ui.OverviewPaneDivider
import com.timeboxxing.app.ui.OverviewPaneLayoutSnapshot
import com.timeboxxing.app.ui.OverviewPaneMotionDirection
import com.timeboxxing.app.ui.OverviewPaneTrayMetrics
import com.timeboxxing.app.ui.overviewPaneDockTransformValues
import com.timeboxxing.app.ui.overviewPaneExpandedBounds
import com.timeboxxing.app.ui.overviewPaneInterpolatedLayoutSnapshot
import com.timeboxxing.app.ui.overviewPaneLayoutSnapshot
import com.timeboxxing.app.ui.overviewPaneMotionLayoutSnapshot
import com.timeboxxing.app.ui.overviewPaneTrayIconBounds
import kotlin.math.abs
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

class OverviewPaneMotionLayoutTest {

    @Test
    fun trayTargetBoundsFollowOverviewPaneOrder() {
        val workspace = Rect(left = 100f, top = 40f, right = 1300f, bottom = 840f)
        val minimizedPanes = setOf(
            OverviewPane.UsageSchedule,
            OverviewPane.TimeEntries,
            OverviewPane.Projects,
        )

        val usageBounds = overviewPaneTrayIconBounds(
            workspaceBounds = workspace,
            minimizedPanes = minimizedPanes,
            pane = OverviewPane.UsageSchedule,
            metrics = trayMetrics,
        )
        val entriesBounds = overviewPaneTrayIconBounds(
            workspaceBounds = workspace,
            minimizedPanes = minimizedPanes,
            pane = OverviewPane.TimeEntries,
            metrics = trayMetrics,
        )
        val projectsBounds = overviewPaneTrayIconBounds(
            workspaceBounds = workspace,
            minimizedPanes = minimizedPanes,
            pane = OverviewPane.Projects,
            metrics = trayMetrics,
        )

        assertRectEquals(Rect(left = 124f, top = 784f, right = 156f, bottom = 816f), assertNotNull(usageBounds))
        assertRectEquals(Rect(left = 164f, top = 784f, right = 196f, bottom = 816f), assertNotNull(entriesBounds))
        assertRectEquals(Rect(left = 204f, top = 784f, right = 236f, bottom = 816f), assertNotNull(projectsBounds))
    }

    @Test
    fun trayTargetBoundsCompactWhenEarlierPanesAreNotMinimized() {
        val workspace = Rect(left = 100f, top = 40f, right = 1300f, bottom = 840f)
        val minimizedPanes = setOf(
            OverviewPane.TimeEntries,
            OverviewPane.Projects,
        )

        val usageBounds = overviewPaneTrayIconBounds(
            workspaceBounds = workspace,
            minimizedPanes = minimizedPanes,
            pane = OverviewPane.UsageSchedule,
            metrics = trayMetrics,
        )
        val entriesBounds = overviewPaneTrayIconBounds(
            workspaceBounds = workspace,
            minimizedPanes = minimizedPanes,
            pane = OverviewPane.TimeEntries,
            metrics = trayMetrics,
        )
        val projectsBounds = overviewPaneTrayIconBounds(
            workspaceBounds = workspace,
            minimizedPanes = minimizedPanes,
            pane = OverviewPane.Projects,
            metrics = trayMetrics,
        )

        assertNull(usageBounds)
        assertRectEquals(Rect(left = 124f, top = 784f, right = 156f, bottom = 816f), assertNotNull(entriesBounds))
        assertRectEquals(Rect(left = 164f, top = 784f, right = 196f, bottom = 816f), assertNotNull(projectsBounds))
    }

    @Test
    fun expandedPaneBoundsPredictRestoreTargetBeforePaneIsVisible() {
        val entriesBounds = overviewPaneExpandedBounds(
            workspaceBounds = workspaceBounds,
            minimizedPanes = emptySet(),
            pane = OverviewPane.TimeEntries,
            projectPaneWidthPx = 280f,
            dividerWidthPx = 1f,
        )

        assertRectEquals(
            Rect(
                left = 544.1724f,
                top = 40f,
                right = 1019f,
                bottom = 840f,
            ),
            assertNotNull(entriesBounds),
            tolerance = 0.0015f,
        )
    }

    @Test
    fun layoutSnapshotsFollowFillSpaceRulesForMinimizedCombinations() {
        val allVisible = snapshot(minimizedPanes = emptySet())

        assertEquals(
            setOf(OverviewPane.UsageSchedule, OverviewPane.TimeEntries, OverviewPane.Projects),
            allVisible.paneBounds.keys,
        )
        assertRectEquals(
            Rect(left = 100f, top = 40f, right = 543.1724f, bottom = 840f),
            assertNotNull(allVisible.paneBounds[OverviewPane.UsageSchedule]),
            tolerance = 0.0015f,
        )
        assertRectEquals(
            Rect(left = 544.1724f, top = 40f, right = 1019f, bottom = 840f),
            assertNotNull(allVisible.paneBounds[OverviewPane.TimeEntries]),
            tolerance = 0.0015f,
        )
        assertRectEquals(
            Rect(left = 1020f, top = 40f, right = 1300f, bottom = 840f),
            assertNotNull(allVisible.paneBounds[OverviewPane.Projects]),
        )
        assertRectEquals(
            Rect(left = 543.1724f, top = 40f, right = 544.1724f, bottom = 840f),
            assertNotNull(allVisible.dividerBounds[OverviewPaneDivider.AfterUsage]),
            tolerance = 0.0015f,
        )
        assertRectEquals(
            Rect(left = 1019f, top = 40f, right = 1020f, bottom = 840f),
            assertNotNull(allVisible.dividerBounds[OverviewPaneDivider.AfterEntries]),
        )

        val expectedVisiblePanesByMinimizedSet = mapOf(
            emptySet<OverviewPane>() to setOf(
                OverviewPane.UsageSchedule,
                OverviewPane.TimeEntries,
                OverviewPane.Projects,
            ),
            setOf(OverviewPane.UsageSchedule) to setOf(OverviewPane.TimeEntries, OverviewPane.Projects),
            setOf(OverviewPane.TimeEntries) to setOf(OverviewPane.UsageSchedule, OverviewPane.Projects),
            setOf(OverviewPane.Projects) to setOf(OverviewPane.UsageSchedule, OverviewPane.TimeEntries),
            setOf(OverviewPane.UsageSchedule, OverviewPane.TimeEntries) to setOf(OverviewPane.Projects),
            setOf(OverviewPane.UsageSchedule, OverviewPane.Projects) to setOf(OverviewPane.TimeEntries),
            setOf(OverviewPane.TimeEntries, OverviewPane.Projects) to setOf(OverviewPane.UsageSchedule),
            setOf(
                OverviewPane.UsageSchedule,
                OverviewPane.TimeEntries,
                OverviewPane.Projects,
            ) to emptySet(),
        )

        expectedVisiblePanesByMinimizedSet.forEach { (minimizedPanes, visiblePanes) ->
            assertEquals(visiblePanes, snapshot(minimizedPanes).paneBounds.keys)
        }
    }

    @Test
    fun stagedRestoreFirstOfTwoMinimizedPanesTargetsFillSpaceLayout() {
        val from = snapshot(
            minimizedPanes = setOf(
                OverviewPane.UsageSchedule,
                OverviewPane.TimeEntries,
            ),
        )
        val to = snapshot(minimizedPanes = setOf(OverviewPane.UsageSchedule))

        assertRectEquals(
            Rect(left = 100f, top = 40f, right = 1300f, bottom = 840f),
            assertNotNull(from.paneBounds[OverviewPane.Projects]),
        )
        assertRectEquals(
            Rect(
                left = 100f,
                top = 40f,
                right = 1019f,
                bottom = 840f,
            ),
            assertNotNull(to.paneBounds[OverviewPane.TimeEntries]),
            tolerance = 0.0015f,
        )
        assertRectEquals(
            Rect(left = 1020f, top = 40f, right = 1300f, bottom = 840f),
            assertNotNull(to.paneBounds[OverviewPane.Projects]),
        )

        assertSnapshotEquals(
            expected = to,
            actual = overviewPaneInterpolatedLayoutSnapshot(from = from, to = to, progress = 1f),
        )
    }

    @Test
    fun stagedRestoreSecondPaneAnimatesExistingPaneFromWideToFinalColumn() {
        val from = snapshot(minimizedPanes = setOf(OverviewPane.UsageSchedule))
        val to = snapshot(minimizedPanes = emptySet())
        val midway = overviewPaneInterpolatedLayoutSnapshot(from = from, to = to, progress = 0.5f)

        val fromEntries = assertNotNull(from.paneBounds[OverviewPane.TimeEntries])
        val midwayEntries = assertNotNull(midway.paneBounds[OverviewPane.TimeEntries])
        val toEntries = assertNotNull(to.paneBounds[OverviewPane.TimeEntries])

        assertTrue(midwayEntries.left > fromEntries.left)
        assertTrue(midwayEntries.left < toEntries.left)
        assertWithin(fromEntries.right, midwayEntries.right)
        assertWithin(toEntries.right, midwayEntries.right)
        assertSnapshotEquals(
            expected = to,
            actual = overviewPaneInterpolatedLayoutSnapshot(from = from, to = to, progress = 1f),
        )
    }

    @Test
    fun leftToRightRestoreStartsUsageCollapsedAtLeftEdge() {
        val from = snapshot(
            minimizedPanes = setOf(
                OverviewPane.UsageSchedule,
                OverviewPane.TimeEntries,
            ),
        )
        val to = snapshot(minimizedPanes = setOf(OverviewPane.TimeEntries))
        val start = motionSnapshot(
            from = from,
            to = to,
            activePane = OverviewPane.UsageSchedule,
            progress = 0f,
        )
        val end = motionSnapshot(
            from = from,
            to = to,
            activePane = OverviewPane.UsageSchedule,
            progress = 1f,
        )

        val startUsage = assertNotNull(start.paneBounds[OverviewPane.UsageSchedule])
        val targetUsage = assertNotNull(to.paneBounds[OverviewPane.UsageSchedule])

        assertRectEquals(
            Rect(
                left = targetUsage.left,
                top = targetUsage.top,
                right = targetUsage.left,
                bottom = targetUsage.bottom,
            ),
            startUsage,
        )
        assertTrue(startUsage.width < targetUsage.width)
        assertRectEquals(
            Rect(left = targetUsage.left, top = 40f, right = targetUsage.left + 1f, bottom = 840f),
            assertNotNull(start.dividerBounds[OverviewPaneDivider.AfterUsage]),
        )
        assertSnapshotEquals(expected = to, actual = end)
    }

    @Test
    fun leftToRightRestoreStartsEntriesAtSourceInsertionEdge() {
        val from = snapshot(minimizedPanes = setOf(OverviewPane.TimeEntries))
        val to = snapshot(minimizedPanes = emptySet())
        val start = motionSnapshot(
            from = from,
            to = to,
            activePane = OverviewPane.TimeEntries,
            progress = 0f,
        )
        val midway = motionSnapshot(
            from = from,
            to = to,
            activePane = OverviewPane.TimeEntries,
            progress = 0.5f,
        )
        val end = motionSnapshot(
            from = from,
            to = to,
            activePane = OverviewPane.TimeEntries,
            progress = 1f,
        )

        val sourceUsage = assertNotNull(from.paneBounds[OverviewPane.UsageSchedule])
        val targetEntries = assertNotNull(to.paneBounds[OverviewPane.TimeEntries])
        val startEntries = assertNotNull(start.paneBounds[OverviewPane.TimeEntries])
        val midwayEntries = assertNotNull(midway.paneBounds[OverviewPane.TimeEntries])

        assertRectEquals(
            Rect(
                left = sourceUsage.right + 1f,
                top = targetEntries.top,
                right = sourceUsage.right + 1f,
                bottom = targetEntries.bottom,
            ),
            startEntries,
        )
        assertTrue(startEntries.left > targetEntries.left)
        assertTrue(startEntries.width < targetEntries.width)
        assertTrue(midwayEntries.left < startEntries.left)
        assertTrue(midwayEntries.left > targetEntries.left)
        assertTrue(midwayEntries.width > startEntries.width)
        assertSnapshotEquals(expected = to, actual = end)
    }

    @Test
    fun rightToLeftRestoreStartsEntriesCollapsedAtLeftInsertionEdge() {
        val from = snapshot(
            minimizedPanes = setOf(
                OverviewPane.UsageSchedule,
                OverviewPane.TimeEntries,
            ),
        )
        val to = snapshot(minimizedPanes = setOf(OverviewPane.UsageSchedule))
        val start = motionSnapshot(
            from = from,
            to = to,
            activePane = OverviewPane.TimeEntries,
            progress = 0f,
        )
        val end = motionSnapshot(
            from = from,
            to = to,
            activePane = OverviewPane.TimeEntries,
            progress = 1f,
        )

        val targetEntries = assertNotNull(to.paneBounds[OverviewPane.TimeEntries])
        assertRectEquals(
            Rect(
                left = targetEntries.left,
                top = targetEntries.top,
                right = targetEntries.left,
                bottom = targetEntries.bottom,
            ),
            assertNotNull(start.paneBounds[OverviewPane.TimeEntries]),
        )
        assertSnapshotEquals(expected = to, actual = end)
    }

    @Test
    fun restoringProjectsStartsCollapsedAtRightInsertionEdge() {
        val from = snapshot(minimizedPanes = setOf(OverviewPane.Projects))
        val to = snapshot(minimizedPanes = emptySet())
        val start = motionSnapshot(
            from = from,
            to = to,
            activePane = OverviewPane.Projects,
            progress = 0f,
        )
        val end = motionSnapshot(
            from = from,
            to = to,
            activePane = OverviewPane.Projects,
            progress = 1f,
        )

        val sourceEntries = assertNotNull(from.paneBounds[OverviewPane.TimeEntries])
        val targetProjects = assertNotNull(to.paneBounds[OverviewPane.Projects])
        assertRectEquals(
            Rect(
                left = sourceEntries.right,
                top = targetProjects.top,
                right = sourceEntries.right,
                bottom = targetProjects.bottom,
            ),
            assertNotNull(start.paneBounds[OverviewPane.Projects]),
        )
        assertRectEquals(
            Rect(left = sourceEntries.right, top = 40f, right = sourceEntries.right + 1f, bottom = 840f),
            assertNotNull(start.dividerBounds[OverviewPaneDivider.AfterEntries]),
        )
        assertSnapshotEquals(expected = to, actual = end)
    }

    @Test
    fun minimizeDockTransformFadesAndSettlesQuickly() {
        val start = overviewPaneDockTransformValues(
            direction = OverviewPaneMotionDirection.Minimize,
            progress = 0f,
            offsetPx = 8f,
        )
        val early = overviewPaneDockTransformValues(
            direction = OverviewPaneMotionDirection.Minimize,
            progress = 0.35f,
            offsetPx = 8f,
        )
        val end = overviewPaneDockTransformValues(
            direction = OverviewPaneMotionDirection.Minimize,
            progress = 1f,
            offsetPx = 8f,
        )

        assertWithin(1f, start.alpha)
        assertWithin(1f, start.scale)
        assertWithin(0f, start.translationY)
        assertTrue(early.alpha < 0.55f, "Expected minimize to fade substantially by early progress")
        assertTrue(early.scale < 1f, "Expected minimize to scale down")
        assertTrue(early.translationY > 0f, "Expected minimize to settle toward the tray")
        assertWithin(0f, end.alpha)
        assertWithin(0.965f, end.scale)
        assertWithin(8f, end.translationY)
    }

    @Test
    fun restoreDockTransformAppearsEarlyAndEndsAtRest() {
        val start = overviewPaneDockTransformValues(
            direction = OverviewPaneMotionDirection.Restore,
            progress = 0f,
            offsetPx = 8f,
        )
        val early = overviewPaneDockTransformValues(
            direction = OverviewPaneMotionDirection.Restore,
            progress = 0.35f,
            offsetPx = 8f,
        )
        val end = overviewPaneDockTransformValues(
            direction = OverviewPaneMotionDirection.Restore,
            progress = 1f,
            offsetPx = 8f,
        )

        assertWithin(0f, start.alpha)
        assertWithin(0.975f, start.scale)
        assertWithin(8f, start.translationY)
        assertTrue(early.alpha > 0.75f, "Expected restore to become visible quickly")
        assertTrue(early.scale > start.scale, "Expected restore to scale toward full size")
        assertTrue(early.translationY < start.translationY, "Expected restore to settle upward")
        assertWithin(1f, end.alpha)
        assertWithin(1f, end.scale)
        assertWithin(0f, end.translationY)
    }

    private fun snapshot(minimizedPanes: Set<OverviewPane>): OverviewPaneLayoutSnapshot =
        overviewPaneLayoutSnapshot(
            workspaceBounds = workspaceBounds,
            minimizedPanes = minimizedPanes,
            projectPaneWidthPx = 280f,
            dividerWidthPx = 1f,
        )

    private fun motionSnapshot(
        from: OverviewPaneLayoutSnapshot,
        to: OverviewPaneLayoutSnapshot,
        activePane: OverviewPane,
        progress: Float,
    ): OverviewPaneLayoutSnapshot =
        overviewPaneMotionLayoutSnapshot(
            from = from,
            to = to,
            activePane = activePane,
            direction = OverviewPaneMotionDirection.Restore,
            progress = progress,
            dividerWidthPx = 1f,
        )

    private fun assertSnapshotEquals(
        expected: OverviewPaneLayoutSnapshot,
        actual: OverviewPaneLayoutSnapshot,
    ) {
        assertEquals(expected.paneBounds.keys, actual.paneBounds.keys)
        expected.paneBounds.forEach { (pane, bounds) ->
            assertRectEquals(bounds, assertNotNull(actual.paneBounds[pane]))
        }
        assertEquals(expected.dividerBounds.keys, actual.dividerBounds.keys)
        expected.dividerBounds.forEach { (divider, bounds) ->
            assertRectEquals(bounds, assertNotNull(actual.dividerBounds[divider]))
        }
    }

    private fun assertRectEquals(expected: Rect, actual: Rect, tolerance: Float = 0.001f) {
        assertWithin(expected.left, actual.left, tolerance)
        assertWithin(expected.top, actual.top, tolerance)
        assertWithin(expected.right, actual.right, tolerance)
        assertWithin(expected.bottom, actual.bottom, tolerance)
    }

    private fun assertWithin(expected: Float, actual: Float, tolerance: Float = 0.001f) {
        assertTrue(
            abs(expected - actual) <= tolerance,
            "Expected <$expected>, got <$actual>",
        )
    }

    private companion object {
        val workspaceBounds = Rect(left = 100f, top = 40f, right = 1300f, bottom = 840f)
        val trayMetrics = OverviewPaneTrayMetrics(
            iconSizePx = 32f,
            iconSpacingPx = 8f,
            startPaddingPx = 24f,
            bottomPaddingPx = 24f,
        )
    }
}
