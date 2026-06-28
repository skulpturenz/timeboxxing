package com.timeboxxing.app.model

enum class UsageSourceType {
    Application,
    Document,
    Email,
    Meeting,
    Spreadsheet,
    Messaging,
    Browser,
    Presentation,
    Idle,
}

enum class EntryMode {
    Timesheet,
    Invoice,
}

enum class AmaMessageRole {
    User,
    Assistant,
}

data class Project(
    val id: String,
    val name: String,
    val client: String,
    val colorArgb: Long,
    val hourlyRateCents: Int,
)

data class UsageEvent(
    val id: String,
    val title: String,
    val sourceName: String,
    val sourceType: UsageSourceType,
    val startMinute: Int,
    val durationMinutes: Int,
    val projectHintId: String?,
    val applicationIdentity: UsageApplicationIdentity? = null,
    val isActive: Boolean = false,
)

data class UsageApplicationIdentity(
    val name: String,
    val identifier: String?,
    val path: String?,
)

data class UsageDay(
    val label: String,
    val startedAtEpochMillis: Long,
    val endedAtEpochMillis: Long,
    val calendarDate: CalendarDate? = null,
)

data class TimeEntry(
    val id: String,
    val projectId: String,
    val title: String,
    val notes: String,
    val startMinute: Int,
    val durationMinutes: Int,
    val billable: Boolean,
    val sourceUsageIds: Set<String>,
)

data class EntryDraft(
    val projectId: String,
    val title: String,
    val notes: String,
    val startMinute: Int,
    val durationMinutes: Int,
    val billable: Boolean,
)

data class AmaSource(
    val transitionEventId: Long,
    val documentKey: String,
    val documentType: String,
    val startedAtEpochMillis: Long?,
    val endedAtEpochMillis: Long?,
    val content: String,
    val distance: Double,
)

enum class AmaIndexState {
    Unknown,
    Ready,
    Indexing,
    Empty,
    Unavailable,
}

data class AmaIndexStatus(
    val state: AmaIndexState,
    val completedEventCount: Long,
    val indexedEventCount: Long,
    val pendingEventCount: Long,
    val backfillRunning: Boolean,
    val message: String,
)

data class AmaAnswer(
    val answer: String,
    val model: String,
    val sources: List<AmaSource>,
    val indexStatus: AmaIndexStatus? = null,
)

data class AmaMessage(
    val id: String,
    val role: AmaMessageRole,
    val content: String,
    val model: String? = null,
    val sources: List<AmaSource> = emptyList(),
)

data class TimeboxxingMockData(
    val usageDays: List<UsageDay>,
    val projects: List<Project>,
    val usageEvents: List<UsageEvent>,
    val initialEntries: List<TimeEntry>,
)

fun formatDuration(minutes: Int): String {
    val safeMinutes = minutes.coerceAtLeast(0)
    val hours = safeMinutes / 60
    val remainder = safeMinutes % 60

    return when {
        hours == 0 -> "${remainder}m"
        remainder == 0 -> "${hours}h"
        else -> "${hours}h ${remainder}m"
    }
}

fun formatClockTime(minutesFromMidnight: Int): String {
    val normalized = ((minutesFromMidnight % (24 * 60)) + (24 * 60)) % (24 * 60)
    val hour24 = normalized / 60
    val minute = normalized % 60
    val suffix = if (hour24 < 12) "AM" else "PM"
    val hour12 = when (val hour = hour24 % 12) {
        0 -> 12
        else -> hour
    }

    return "$hour12:${minute.toString().padStart(2, '0')} $suffix"
}

fun formatTimeRange(startMinute: Int, durationMinutes: Int): String =
    "${formatClockTime(startMinute)} - ${formatClockTime(startMinute + durationMinutes)}"
