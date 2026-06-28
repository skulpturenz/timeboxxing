package com.timeboxxing.app.data

import com.google.protobuf.Timestamp
import com.timeboxxing.app.model.UsageDay
import com.timeboxxing.app.model.UsageApplicationIdentity
import com.timeboxxing.app.model.UsageEvent
import com.timeboxxing.app.model.UsageSourceType
import com.timeboxxing.sidecar.usage.v1.UsageSource
import com.timeboxxing.sidecar.usage.v1.UsageEvent as UsageEventProto
import kotlin.math.max
import kotlin.math.min

internal fun UsageEventProto.toUsageEvent(day: UsageDay): UsageEvent? {
    val eventStartMillis = startedAt?.toEpochMillis() ?: return null
    val eventEndMillis = endedAt?.toEpochMillis() ?: return null
    val clippedStartMillis = max(eventStartMillis, day.startedAtEpochMillis)
    val clippedEndMillis = min(eventEndMillis, day.endedAtEpochMillis)
    if (clippedEndMillis <= clippedStartMillis) return null

    val sourceName = sourceName.ifBlank { applicationName.ifBlank { "Application" } }
    val sourceType = source.toUsageSourceType()
    val applicationIdentity = when (sourceType) {
        UsageSourceType.Idle -> null
        else -> UsageApplicationIdentity(
            name = applicationName.ifBlank { sourceName },
            identifier = applicationIdentifier.takeIf { it.isNotBlank() },
            path = applicationPath.takeIf { it.isNotBlank() },
        )
    }
    val title = title.ifBlank {
        when (sourceType) {
            UsageSourceType.Idle -> "Idle"
            else -> sourceName
        }
    }
    val startMinute = ((clippedStartMillis - day.startedAtEpochMillis) / MillisPerMinute)
        .toInt()
        .coerceIn(0, MinutesPerDay - 1)
    val durationMinutes = ((clippedEndMillis - clippedStartMillis + MillisPerMinute - 1) / MillisPerMinute)
        .toInt()
        .coerceAtLeast(1)

    return UsageEvent(
        id = if (active) ActiveUsageEventId else "sidecar-$id",
        title = title,
        sourceName = sourceName,
        sourceType = sourceType,
        startMinute = startMinute,
        durationMinutes = durationMinutes,
        projectHintId = null,
        applicationIdentity = applicationIdentity,
        isActive = active,
    )
}

private fun UsageSource.toUsageSourceType(): UsageSourceType =
    when (this) {
        UsageSource.USAGE_SOURCE_BROWSER -> UsageSourceType.Browser
        UsageSource.USAGE_SOURCE_IDLE -> UsageSourceType.Idle
        else -> UsageSourceType.Application
    }

private fun Timestamp.toEpochMillis(): Long =
    seconds * 1_000 + nanos / 1_000_000

private const val MillisPerMinute = 60_000L
private const val MinutesPerDay = 24 * 60
private const val ActiveUsageEventId = "sidecar-active"
