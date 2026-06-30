package com.timeboxxing.app.ui

import androidx.compose.ui.geometry.Rect

internal data class OverviewPaneTrayMetrics(
    val iconSizePx: Float,
    val iconSpacingPx: Float,
    val startPaddingPx: Float,
    val bottomPaddingPx: Float,
)

internal data class OverviewPaneLayoutSnapshot(
    val paneBounds: Map<OverviewPane, Rect>,
    val dividerBounds: Map<OverviewPaneDivider, Rect>,
)

internal data class OverviewPaneDockTransformValues(
    val alpha: Float,
    val scale: Float,
    val translationY: Float,
)

internal enum class OverviewPaneDivider {
    AfterUsage,
    AfterEntries,
}

private const val UsagePaneWeight = 0.98f
private const val EntriesPaneWeight = 1.05f
private const val SoloProjectPaneWeight = 1f

internal fun overviewPaneLayoutSnapshot(
    workspaceBounds: Rect,
    minimizedPanes: Set<OverviewPane>,
    projectPaneWidthPx: Float,
    dividerWidthPx: Float,
): OverviewPaneLayoutSnapshot {
    val usageVisible = OverviewPane.UsageSchedule !in minimizedPanes
    val entriesVisible = OverviewPane.TimeEntries !in minimizedPanes
    val projectsVisible = OverviewPane.Projects !in minimizedPanes
    val visibleCount = listOf(usageVisible, entriesVisible, projectsVisible).count { it }
    if (visibleCount == 0) {
        return OverviewPaneLayoutSnapshot(
            paneBounds = emptyMap(),
            dividerBounds = emptyMap(),
        )
    }

    val dividerWidth = dividerWidthPx.coerceAtLeast(0f)
    val dividerCount = listOf(
        usageVisible && (entriesVisible || projectsVisible),
        entriesVisible && projectsVisible,
    ).count { it }
    val projectIsFixed = projectsVisible && visibleCount > 1
    val fixedProjectWidth = if (projectIsFixed) projectPaneWidthPx.coerceAtLeast(0f) else 0f
    val weightedWidth = (
        workspaceBounds.width -
            fixedProjectWidth -
            dividerWidth * dividerCount
        ).coerceAtLeast(0f)
    val totalWeight =
        (if (usageVisible) UsagePaneWeight else 0f) +
            (if (entriesVisible) EntriesPaneWeight else 0f) +
            (if (projectsVisible && !projectIsFixed) SoloProjectPaneWeight else 0f)

    val usageWidth = if (usageVisible && totalWeight > 0f) {
        weightedWidth * UsagePaneWeight / totalWeight
    } else {
        0f
    }
    val entriesWidth = if (entriesVisible && totalWeight > 0f) {
        weightedWidth * EntriesPaneWeight / totalWeight
    } else {
        0f
    }
    val projectsWidth = when {
        !projectsVisible -> 0f
        projectIsFixed -> fixedProjectWidth
        totalWeight > 0f -> weightedWidth * SoloProjectPaneWeight / totalWeight
        else -> 0f
    }

    var left = workspaceBounds.left
    val paneBounds = mutableMapOf<OverviewPane, Rect>()
    val dividerBounds = mutableMapOf<OverviewPaneDivider, Rect>()

    fun paneRect(width: Float): Rect =
        Rect(
            left = left,
            top = workspaceBounds.top,
            right = (left + width).coerceAtMost(workspaceBounds.right),
            bottom = workspaceBounds.bottom,
        )

    fun dividerRect(): Rect =
        Rect(
            left = left,
            top = workspaceBounds.top,
            right = (left + dividerWidth).coerceAtMost(workspaceBounds.right),
            bottom = workspaceBounds.bottom,
        )

    if (usageVisible) {
        val rect = paneRect(usageWidth)
        paneBounds[OverviewPane.UsageSchedule] = rect
        left = rect.right
        if (entriesVisible || projectsVisible) {
            dividerBounds[OverviewPaneDivider.AfterUsage] = dividerRect()
            left += dividerWidth
        }
    }
    if (entriesVisible) {
        val rect = paneRect(entriesWidth)
        paneBounds[OverviewPane.TimeEntries] = rect
        left = rect.right
        if (projectsVisible) {
            dividerBounds[OverviewPaneDivider.AfterEntries] = dividerRect()
            left += dividerWidth
        }
    }
    if (projectsVisible) {
        paneBounds[OverviewPane.Projects] = paneRect(projectsWidth)
    }

    return OverviewPaneLayoutSnapshot(
        paneBounds = paneBounds,
        dividerBounds = dividerBounds,
    )
}

internal fun overviewPaneInterpolatedLayoutSnapshot(
    from: OverviewPaneLayoutSnapshot,
    to: OverviewPaneLayoutSnapshot,
    progress: Float,
): OverviewPaneLayoutSnapshot {
    val fraction = progress.coerceIn(0f, 1f)
    return OverviewPaneLayoutSnapshot(
        paneBounds = interpolateBoundsByKey(
            from = from.paneBounds,
            to = to.paneBounds,
            fraction = fraction,
        ),
        dividerBounds = interpolateBoundsByKey(
            from = from.dividerBounds,
            to = to.dividerBounds,
            fraction = fraction,
        ),
    )
}

internal fun overviewPaneMotionLayoutSnapshot(
    from: OverviewPaneLayoutSnapshot,
    to: OverviewPaneLayoutSnapshot,
    activePane: OverviewPane,
    direction: OverviewPaneMotionDirection,
    progress: Float,
    dividerWidthPx: Float,
): OverviewPaneLayoutSnapshot {
    val targetBounds = to.paneBounds[activePane]
    if (
        direction != OverviewPaneMotionDirection.Restore ||
        from.paneBounds[activePane] != null ||
        targetBounds == null
    ) {
        return overviewPaneInterpolatedLayoutSnapshot(
            from = from,
            to = to,
            progress = progress,
        )
    }

    val fraction = progress.coerceIn(0f, 1f)
    val insertionX = restoreInsertionX(
        from = from,
        to = to,
        activePane = activePane,
        dividerWidthPx = dividerWidthPx,
    )
    val activeStartBounds = Rect(
        left = insertionX,
        top = targetBounds.top,
        right = insertionX,
        bottom = targetBounds.bottom,
    )
    val paneBounds = interpolateBoundsByKey(
        from = from.paneBounds + (activePane to activeStartBounds),
        to = to.paneBounds,
        fraction = fraction,
    )
    return OverviewPaneLayoutSnapshot(
        paneBounds = paneBounds,
        dividerBounds = overviewPaneDividerBoundsFor(
            paneBounds = paneBounds,
            dividerWidthPx = dividerWidthPx,
        ),
    )
}

internal fun overviewPaneDockTransformValues(
    direction: OverviewPaneMotionDirection,
    progress: Float,
    offsetPx: Float,
): OverviewPaneDockTransformValues {
    val fraction = progress.coerceIn(0f, 1f)
    val settle = easeOutCubic(fraction)
    return when (direction) {
        OverviewPaneMotionDirection.Minimize -> {
            val fade = smoothStep((fraction / 0.72f).coerceIn(0f, 1f))
            OverviewPaneDockTransformValues(
                alpha = lerpFloat(1f, 0f, fade),
                scale = lerpFloat(1f, 0.965f, settle),
                translationY = lerpFloat(0f, offsetPx, settle),
            )
        }
        OverviewPaneMotionDirection.Restore -> {
            val fade = smoothStep((fraction / 0.48f).coerceIn(0f, 1f))
            OverviewPaneDockTransformValues(
                alpha = fade,
                scale = lerpFloat(0.975f, 1f, settle),
                translationY = lerpFloat(offsetPx, 0f, settle),
            )
        }
    }
}

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
    return overviewPaneLayoutSnapshot(
        workspaceBounds = workspaceBounds,
        minimizedPanes = minimizedPanes,
        projectPaneWidthPx = projectPaneWidthPx,
        dividerWidthPx = dividerWidthPx,
    ).paneBounds[pane]
}

internal fun lerpFloat(start: Float, end: Float, fraction: Float): Float =
    start + (end - start) * fraction

internal fun lerpRect(start: Rect, end: Rect, fraction: Float): Rect =
    Rect(
        left = lerpFloat(start.left, end.left, fraction),
        top = lerpFloat(start.top, end.top, fraction),
        right = lerpFloat(start.right, end.right, fraction),
        bottom = lerpFloat(start.bottom, end.bottom, fraction),
    )

private fun <Key> interpolateBoundsByKey(
    from: Map<Key, Rect>,
    to: Map<Key, Rect>,
    fraction: Float,
): Map<Key, Rect> {
    val keys = from.keys + to.keys
    return keys.mapNotNull { key ->
        val start = from[key]
        val end = to[key]
        val bounds = when {
            start != null && end != null -> lerpRect(start, end, fraction)
            end != null -> end
            start != null && fraction < 1f -> start
            else -> null
        }
        bounds?.let { key to it }
    }.toMap()
}

private fun restoreInsertionX(
    from: OverviewPaneLayoutSnapshot,
    to: OverviewPaneLayoutSnapshot,
    activePane: OverviewPane,
    dividerWidthPx: Float,
): Float {
    val targetBounds = to.paneBounds.getValue(activePane)
    val activeIndex = OverviewPane.entries.indexOf(activePane)
    val sourceLeftPane = OverviewPane.entries
        .take(activeIndex)
        .lastOrNull { from.paneBounds[it] != null }
    val sourceRightPane = OverviewPane.entries
        .drop(activeIndex + 1)
        .firstOrNull { from.paneBounds[it] != null }

    return when {
        sourceLeftPane == null -> targetBounds.left
        sourceRightPane == null -> from.paneBounds.getValue(sourceLeftPane).right
        else -> from.paneBounds.getValue(sourceLeftPane).right + dividerWidthPx.coerceAtLeast(0f)
    }
}

private fun overviewPaneDividerBoundsFor(
    paneBounds: Map<OverviewPane, Rect>,
    dividerWidthPx: Float,
): Map<OverviewPaneDivider, Rect> {
    val dividerWidth = dividerWidthPx.coerceAtLeast(0f)
    val usageBounds = paneBounds[OverviewPane.UsageSchedule]
    val entriesBounds = paneBounds[OverviewPane.TimeEntries]
    val projectsBounds = paneBounds[OverviewPane.Projects]
    val dividerBounds = mutableMapOf<OverviewPaneDivider, Rect>()

    fun dividerAfter(bounds: Rect): Rect =
        Rect(
            left = bounds.right,
            top = bounds.top,
            right = bounds.right + dividerWidth,
            bottom = bounds.bottom,
        )

    if (usageBounds != null && (entriesBounds != null || projectsBounds != null)) {
        dividerBounds[OverviewPaneDivider.AfterUsage] = dividerAfter(usageBounds)
    }
    if (entriesBounds != null && projectsBounds != null) {
        dividerBounds[OverviewPaneDivider.AfterEntries] = dividerAfter(entriesBounds)
    }

    return dividerBounds
}

internal fun easeOutCubic(fraction: Float): Float {
    val inverse = 1f - fraction
    return 1f - inverse * inverse * inverse
}

internal fun smoothStep(fraction: Float): Float =
    fraction * fraction * (3f - 2f * fraction)
