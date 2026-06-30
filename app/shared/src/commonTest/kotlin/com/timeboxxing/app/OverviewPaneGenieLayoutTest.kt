package com.timeboxxing.app

import androidx.compose.ui.geometry.Rect
import com.timeboxxing.app.ui.OverviewPane
import com.timeboxxing.app.ui.OverviewPaneMotionDirection
import com.timeboxxing.app.ui.OverviewPaneTrayMetrics
import com.timeboxxing.app.ui.buildOverviewPaneGenieCue
import com.timeboxxing.app.ui.overviewPaneExpandedBounds
import com.timeboxxing.app.ui.overviewPaneGenieTransformValues
import com.timeboxxing.app.ui.overviewPaneLayoutMinimizedPanes
import com.timeboxxing.app.ui.overviewPaneTrayIconBounds
import kotlin.math.abs
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

class OverviewPaneGenieLayoutTest {

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
    fun restoreLayoutUsesPaneAsVisibleBeforeCanonicalRestoreCommits() {
        val canonicalMinimizedPanes = setOf(
            OverviewPane.UsageSchedule,
            OverviewPane.TimeEntries,
        )
        val layoutMinimizedPanes = overviewPaneLayoutMinimizedPanes(
            minimizedPanes = canonicalMinimizedPanes,
            restoringPane = OverviewPane.TimeEntries,
        )

        assertEquals(setOf(OverviewPane.UsageSchedule), layoutMinimizedPanes)

        val entriesBounds = overviewPaneExpandedBounds(
            workspaceBounds = workspaceBounds,
            minimizedPanes = layoutMinimizedPanes,
            pane = OverviewPane.TimeEntries,
            projectPaneWidthPx = 280f,
            dividerWidthPx = 1f,
        )

        assertRectEquals(
            Rect(
                left = 100f,
                top = 40f,
                right = 1019f,
                bottom = 840f,
            ),
            assertNotNull(entriesBounds),
            tolerance = 0.0015f,
        )
    }

    @Test
    fun minimizeCueStartsAtFullPaneAndEndsAtTrayIconCenter() {
        val cue = buildOverviewPaneGenieCue(
            id = 1,
            pane = OverviewPane.UsageSchedule,
            direction = OverviewPaneMotionDirection.Minimize,
            sourceBounds = paneBounds,
            targetBounds = trayBounds,
            workspaceBounds = workspaceBounds,
            trayScale = trayScale,
        )

        assertRectEquals(Rect(left = 120f, top = 80f, right = 620f, bottom = 740f), cue.startBounds)
        assertWithin(trayBounds.toLocalXCenter(), cue.endBounds.center.x)
        assertWithin(trayBounds.toLocalYCenter(), cue.endBounds.center.y)
        assertWithin(trayBounds.width * trayScale, cue.endBounds.width)
        assertWithin(trayBounds.height * trayScale, cue.endBounds.height)
    }

    @Test
    fun earlyMinimizeProgressMovesTowardTray() {
        val cue = buildOverviewPaneGenieCue(
            id = 1,
            pane = OverviewPane.UsageSchedule,
            direction = OverviewPaneMotionDirection.Minimize,
            sourceBounds = paneBounds,
            targetBounds = trayBounds,
            workspaceBounds = workspaceBounds,
            trayScale = trayScale,
        )

        val initial = overviewPaneGenieTransformValues(
            cue = cue,
            baseBounds = cue.startBounds,
            progress = 0f,
        )
        val early = overviewPaneGenieTransformValues(
            cue = cue,
            baseBounds = cue.startBounds,
            progress = 0.25f,
        )

        assertWithin(0f, initial.translationX)
        assertWithin(0f, initial.translationY)
        assertWithin(1f, initial.scaleX)
        assertWithin(1f, initial.scaleY)
        assertTrue(early.translationX < 0f, "Expected early minimize translation to move left toward the tray")
        assertTrue(early.translationY > 0f, "Expected early minimize translation to move down toward the tray")
        assertTrue(abs(early.translationX) > abs(initial.translationX))
        assertTrue(abs(early.translationY) > abs(initial.translationY))
        assertTrue(early.scaleX < initial.scaleX, "Expected early minimize scale to be smaller than the pane")
        assertWithin(1f, early.alpha)
    }

    @Test
    fun restoreCueStartsAtTrayAndEndsAtPaneBounds() {
        val cue = buildOverviewPaneGenieCue(
            id = 2,
            pane = OverviewPane.UsageSchedule,
            direction = OverviewPaneMotionDirection.Restore,
            sourceBounds = trayBounds,
            targetBounds = paneBounds,
            workspaceBounds = workspaceBounds,
            trayScale = trayScale,
        )

        assertWithin(trayBounds.toLocalXCenter(), cue.startBounds.center.x)
        assertWithin(trayBounds.toLocalYCenter(), cue.startBounds.center.y)
        assertWithin(trayBounds.width * trayScale, cue.startBounds.width)
        assertWithin(trayBounds.height * trayScale, cue.startBounds.height)
        assertRectEquals(Rect(left = 120f, top = 80f, right = 620f, bottom = 740f), cue.endBounds)

        val start = overviewPaneGenieTransformValues(
            cue = cue,
            baseBounds = cue.endBounds,
            progress = 0f,
        )
        val end = overviewPaneGenieTransformValues(
            cue = cue,
            baseBounds = cue.endBounds,
            progress = 1f,
        )

        assertWithin(cue.startBounds.width / cue.endBounds.width, start.scaleX)
        assertWithin(cue.startBounds.height / cue.endBounds.height, start.scaleY)
        assertWithin(0.64f, start.alpha)
        assertWithin(1f, end.scaleX)
        assertWithin(1f, end.scaleY)
        assertWithin(0f, end.translationX)
        assertWithin(0f, end.translationY)
        assertWithin(1f, end.alpha)
    }

    private fun Float.toLocalX(): Float = this - workspaceBounds.left

    private fun Float.toLocalY(): Float = this - workspaceBounds.top

    private fun Rect.toLocalXCenter(): Float = center.x.toLocalX()

    private fun Rect.toLocalYCenter(): Float = center.y.toLocalY()

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
        val paneBounds = Rect(left = 220f, top = 120f, right = 720f, bottom = 780f)
        val trayBounds = Rect(left = 124f, top = 784f, right = 156f, bottom = 816f)
        val trayMetrics = OverviewPaneTrayMetrics(
            iconSizePx = 32f,
            iconSpacingPx = 8f,
            startPaddingPx = 24f,
            bottomPaddingPx = 24f,
        )
        const val trayScale = 0.86f
    }
}
