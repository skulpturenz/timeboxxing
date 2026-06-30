package com.timeboxxing.app.ui

import androidx.compose.ui.geometry.Rect

internal data class OverviewPaneTrayMetrics(
    val iconSizePx: Float,
    val iconSpacingPx: Float,
    val startPaddingPx: Float,
    val bottomPaddingPx: Float,
)

internal data class OverviewPaneGenieTransformValues(
    val originXFraction: Float,
    val originYFraction: Float,
    val translationX: Float,
    val translationY: Float,
    val scaleX: Float,
    val scaleY: Float,
    val alpha: Float,
)

private const val UsagePaneWeight = 0.98f
private const val EntriesPaneWeight = 1.05f
private const val SoloProjectPaneWeight = 1f

internal fun overviewPaneLayoutMinimizedPanes(
    minimizedPanes: Set<OverviewPane>,
    restoringPane: OverviewPane?,
): Set<OverviewPane> =
    restoringPane?.let { minimizedPanes - it } ?: minimizedPanes

internal fun overviewPaneTrayIconBounds(
    workspaceBounds: Rect,
    minimizedPanes: Set<OverviewPane>,
    pane: OverviewPane,
    metrics: OverviewPaneTrayMetrics,
): Rect? {
    val visibleTrayPanes = OverviewPane.entries.filter { it in minimizedPanes }
    val paneIndex = visibleTrayPanes.indexOf(pane)
    if (paneIndex < 0) {
        return null
    }

    val left = workspaceBounds.left +
        metrics.startPaddingPx +
        paneIndex * (metrics.iconSizePx + metrics.iconSpacingPx)
    val top = workspaceBounds.bottom - metrics.bottomPaddingPx - metrics.iconSizePx
    return Rect(
        left = left,
        top = top,
        right = left + metrics.iconSizePx,
        bottom = top + metrics.iconSizePx,
    )
}

internal fun overviewPaneExpandedBounds(
    workspaceBounds: Rect,
    minimizedPanes: Set<OverviewPane>,
    pane: OverviewPane,
    projectPaneWidthPx: Float,
    dividerWidthPx: Float,
): Rect? {
    if (pane in minimizedPanes) {
        return null
    }

    val usageVisible = OverviewPane.UsageSchedule !in minimizedPanes
    val entriesVisible = OverviewPane.TimeEntries !in minimizedPanes
    val projectsVisible = OverviewPane.Projects !in minimizedPanes
    val visibleCount = listOf(usageVisible, entriesVisible, projectsVisible).count { it }
    if (visibleCount == 0) {
        return null
    }

    val dividerCount = listOf(
        usageVisible && (entriesVisible || projectsVisible),
        entriesVisible && projectsVisible,
    ).count { it }
    val projectIsFixed = projectsVisible && visibleCount > 1
    val fixedProjectWidth = if (projectIsFixed) projectPaneWidthPx.coerceAtLeast(0f) else 0f
    val weightedWidth = (
        workspaceBounds.width -
            fixedProjectWidth -
            dividerWidthPx.coerceAtLeast(0f) * dividerCount
        ).coerceAtLeast(0f)
    val totalWeight =
        (if (usageVisible) UsagePaneWeight else 0f) +
            (if (entriesVisible) EntriesPaneWeight else 0f) +
            (if (projectsVisible && !projectIsFixed) SoloProjectPaneWeight else 0f)

    val usageWidth = if (usageVisible && totalWeight > 0f) weightedWidth * UsagePaneWeight / totalWeight else 0f
    val entriesWidth = if (entriesVisible && totalWeight > 0f) weightedWidth * EntriesPaneWeight / totalWeight else 0f
    val projectsWidth = when {
        !projectsVisible -> 0f
        projectIsFixed -> fixedProjectWidth
        totalWeight > 0f -> weightedWidth * SoloProjectPaneWeight / totalWeight
        else -> 0f
    }

    var left = workspaceBounds.left
    fun paneRect(width: Float): Rect =
        Rect(
            left = left,
            top = workspaceBounds.top,
            right = (left + width).coerceAtMost(workspaceBounds.right),
            bottom = workspaceBounds.bottom,
        )

    if (usageVisible) {
        val rect = paneRect(usageWidth)
        if (pane == OverviewPane.UsageSchedule) {
            return rect
        }
        left = rect.right
        if (entriesVisible || projectsVisible) {
            left += dividerWidthPx
        }
    }
    if (entriesVisible) {
        val rect = paneRect(entriesWidth)
        if (pane == OverviewPane.TimeEntries) {
            return rect
        }
        left = rect.right
        if (projectsVisible) {
            left += dividerWidthPx
        }
    }
    if (projectsVisible) {
        val rect = paneRect(projectsWidth)
        if (pane == OverviewPane.Projects) {
            return rect
        }
    }

    return null
}

internal fun buildOverviewPaneGenieCue(
    id: Int,
    pane: OverviewPane,
    direction: OverviewPaneMotionDirection,
    sourceBounds: Rect,
    targetBounds: Rect,
    workspaceBounds: Rect,
    trayScale: Float,
): OverviewPaneGenieCue {
    val localSourceBounds = sourceBounds.toLocalBounds(workspaceBounds)
    val localTargetBounds = targetBounds.toLocalBounds(workspaceBounds)

    return OverviewPaneGenieCue(
        id = id,
        pane = pane,
        direction = direction,
        startBounds = when (direction) {
            OverviewPaneMotionDirection.Minimize -> localSourceBounds
            OverviewPaneMotionDirection.Restore -> localSourceBounds.centeredScale(trayScale)
        },
        endBounds = when (direction) {
            OverviewPaneMotionDirection.Minimize -> localTargetBounds.centeredScale(trayScale)
            OverviewPaneMotionDirection.Restore -> localTargetBounds
        },
    )
}

internal fun overviewPaneGenieTransformValues(
    cue: OverviewPaneGenieCue,
    baseBounds: Rect,
    progress: Float,
): OverviewPaneGenieTransformValues {
    if (baseBounds.width <= 0f || baseBounds.height <= 0f) {
        return OverviewPaneGenieTransformValues(
            originXFraction = 0.5f,
            originYFraction = 0.5f,
            translationX = 0f,
            translationY = 0f,
            scaleX = 1f,
            scaleY = 1f,
            alpha = 1f,
        )
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

    return OverviewPaneGenieTransformValues(
        originXFraction = originXFraction,
        originYFraction = originYFraction,
        translationX = lerpFloat(startOriginX, targetOriginX, originProgress) - originX,
        translationY = lerpFloat(startOriginY, targetOriginY, originProgress) - originY,
        scaleX = lerpFloat(startScaleX, targetScaleX, widthProgress),
        scaleY = lerpFloat(startScaleY, targetScaleY, heightProgress),
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
        },
    )
}

internal fun Rect.toLocalBounds(containerBounds: Rect): Rect =
    Rect(
        left = left - containerBounds.left,
        top = top - containerBounds.top,
        right = right - containerBounds.left,
        bottom = bottom - containerBounds.top,
    )

internal fun Rect.centeredScale(scale: Float): Rect {
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

internal fun lerpFloat(start: Float, end: Float, fraction: Float): Float =
    start + (end - start) * fraction

internal fun easeOutCubic(fraction: Float): Float {
    val inverse = 1f - fraction
    return 1f - inverse * inverse * inverse
}

internal fun stagedProgress(fraction: Float, start: Float, end: Float): Float {
    if (end <= start) {
        return if (fraction >= end) 1f else 0f
    }
    return smoothStep(((fraction - start) / (end - start)).coerceIn(0f, 1f))
}

internal fun smoothStep(fraction: Float): Float =
    fraction * fraction * (3f - 2f * fraction)
