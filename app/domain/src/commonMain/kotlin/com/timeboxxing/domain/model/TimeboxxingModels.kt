package com.timeboxxing.domain.model

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

enum class AppearanceMode {
    System,
    Light,
    Dark,
}

enum class TimesheetExportFormat {
    Json,
    Csv,
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
    val pid: Int? = null,
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

data class TimesheetEntryDraft(
    val projectId: String,
    val title: String,
    val notes: String,
    val startMinute: Int,
    val durationMinutes: Int,
    val billable: Boolean,
    val sourceUsageIds: List<String>,
)

data class TimesheetExport(
    val fileName: String,
    val contentType: String,
    val content: ByteArray,
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

enum class AmaQueryKind {
    AppTotals,
    Timeline,
    Habits,
    ComparePeriods,
}

data class AmaTimeWindow(
    val startedAtEpochMillis: Long,
    val endedAtEpochMillis: Long,
)

data class AmaStructuredQuery(
    val kind: AmaQueryKind,
    val window: AmaTimeWindow,
    val baselineWindow: AmaTimeWindow? = null,
    val limit: Int = 0,
    val includeIdle: Boolean = false,
    val periodLabel: String = "",
    val baselinePeriodLabel: String = "",
)

sealed interface AmaArtifact

data class AmaAppUsageChart(
    val periodLabel: String,
    val startedAtEpochMillis: Long?,
    val endedAtEpochMillis: Long?,
    val timeZone: String,
    val totalDurationSeconds: Long,
    val buckets: List<AmaAppUsageBucket>,
) : AmaArtifact

data class AmaAppUsageBucket(
    val name: String,
    val sourceType: String,
    val durationSeconds: Long,
    val sessionCount: Long,
    val applicationIdentifier: String,
    val applicationPath: String,
)

data class AmaUsageTimeline(
    val periodLabel: String,
    val startedAtEpochMillis: Long?,
    val endedAtEpochMillis: Long?,
    val timeZone: String,
    val totalDurationSeconds: Long,
    val totalEventCount: Int,
    val truncated: Boolean,
    val events: List<AmaUsageTimelineEvent>,
) : AmaArtifact

data class AmaUsageTimelineEvent(
    val transitionEventId: Long,
    val title: String,
    val sourceName: String,
    val sourceType: String,
    val startedAtEpochMillis: Long?,
    val endedAtEpochMillis: Long?,
    val durationSeconds: Long,
    val applicationIdentifier: String,
    val applicationPath: String,
    val urlHost: String,
    val idle: Boolean,
)

data class AmaHabitSummary(
    val periodLabel: String,
    val startedAtEpochMillis: Long?,
    val endedAtEpochMillis: Long?,
    val timeZone: String,
    val totalDurationSeconds: Long,
    val sessionCount: Long,
    val contextSwitchCount: Long,
    val averageSessionSeconds: Long,
    val longestSession: AmaUsageTimelineEvent?,
    val topSources: List<AmaAppUsageBucket>,
    val timeBuckets: List<AmaTimeOfDayBucket>,
) : AmaArtifact

data class AmaTimeOfDayBucket(
    val label: String,
    val durationSeconds: Long,
    val sessionCount: Long,
)

data class AmaUsageComparison(
    val currentStartedAtEpochMillis: Long?,
    val currentEndedAtEpochMillis: Long?,
    val baselineStartedAtEpochMillis: Long?,
    val baselineEndedAtEpochMillis: Long?,
    val timeZone: String,
    val currentPeriodLabel: String,
    val baselinePeriodLabel: String,
    val currentTotalDurationSeconds: Long,
    val baselineTotalDurationSeconds: Long,
    val durationDeltaSeconds: Long,
    val durationDeltaPercent: Double,
    val buckets: List<AmaUsageComparisonBucket>,
) : AmaArtifact

data class AmaUsageComparisonBucket(
    val name: String,
    val sourceType: String,
    val currentDurationSeconds: Long,
    val baselineDurationSeconds: Long,
    val deltaDurationSeconds: Long,
    val currentSessionCount: Long,
    val baselineSessionCount: Long,
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
    val artifacts: List<AmaArtifact> = emptyList(),
    val indexStatus: AmaIndexStatus? = null,
)

data class AmaMessage(
    val id: String,
    val role: AmaMessageRole,
    val content: String,
    val model: String? = null,
    val sources: List<AmaSource> = emptyList(),
    val artifacts: List<AmaArtifact> = emptyList(),
)

data class DiagnosticsLogLine(
    val sequence: Long,
    val timestamp: String,
    val message: String,
)

enum class AiProvider {
    OpenRouter,
    Ollama,
}

data class AiModelOption(
    val id: Long,
    val openRouterSlug: String,
    val ollamaSlug: String,
    val label: String,
) {
    fun supports(provider: AiProvider): Boolean =
        when (provider) {
            AiProvider.OpenRouter -> openRouterSlug.isNotBlank()
            AiProvider.Ollama -> ollamaSlug.isNotBlank()
        }
}

data class AiModelOptions(
    val embeddingModels: List<AiModelOption> = emptyList(),
    val semanticModels: List<AiModelOption> = emptyList(),
)

data class AiSettings(
    val provider: AiProvider = AiProvider.OpenRouter,
    val openRouterBaseUrl: String = "https://openrouter.ai/api/v1",
    val ollamaBaseUrl: String = "http://127.0.0.1:11434",
    val embeddingModelId: Long = 1,
    val semanticModelId: Long = 1,
    val openRouterApiKey: String = "",
    val ollamaApiKey: String = "",
    val openRouterSecretExists: Boolean = false,
    val ollamaSecretExists: Boolean = false,
)

data class DatabaseMaintenanceStatus(
    val sizeBytes: Long = 0,
)

data class DatabasePruneRange(
    val startedAtEpochMillis: Long,
    val endedAtEpochMillis: Long,
)

data class DatabasePruneCounts(
    val timesheetEntriesDeleted: Long = 0,
    val usageLinksDeleted: Long = 0,
    val timesheetsDeleted: Long = 0,
    val transitionEventsDeleted: Long = 0,
    val transitionMetadataDeleted: Long = 0,
    val semanticDocumentsDeleted: Long = 0,
    val embeddingsDeleted: Long = 0,
    val applicationsDeleted: Long = 0,
) {
    val totalDeletedRows: Long
        get() = timesheetEntriesDeleted +
            usageLinksDeleted +
            timesheetsDeleted +
            transitionEventsDeleted +
            transitionMetadataDeleted +
            semanticDocumentsDeleted +
            embeddingsDeleted +
            applicationsDeleted
}

data class DatabasePruneResult(
    val status: DatabaseMaintenanceStatus,
    val counts: DatabasePruneCounts,
)

data class DatabaseVacuumResult(
    val sizeBeforeBytes: Long,
    val sizeAfterBytes: Long,
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
