package com.timeboxxing.data.repository

import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.TimesheetEntryDraft
import com.timeboxxing.domain.model.TimesheetExport
import com.timeboxxing.domain.model.TimesheetExportFormat
import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.repository.TimesheetRepository

class StaticTimesheetRepository(
    initialEntries: List<TimeEntry> = emptyList(),
) : TimesheetRepository {
    private val entriesByDay = mutableMapOf<Long, MutableList<TimeEntry>>()
    private var nextEntryNumber = 1

    init {
        if (initialEntries.isNotEmpty()) {
            entriesByDay[0L] = initialEntries.toMutableList()
            nextEntryNumber = initialEntries.size + 1
        }
    }

    override suspend fun listEntries(day: UsageDay): List<TimeEntry> =
        entriesFor(day).toList()

    override suspend fun createEntry(day: UsageDay, draft: TimesheetEntryDraft): TimeEntry {
        val entry = TimeEntry(
            id = "entry-${nextEntryNumber++}",
            projectId = draft.projectId,
            title = draft.title.ifBlank { "New time entry" },
            notes = draft.notes,
            startMinute = draft.startMinute,
            durationMinutes = draft.durationMinutes,
            billable = draft.billable,
            sourceUsageIds = draft.sourceUsageIds.toSet(),
        )
        entriesFor(day) += entry
        return entry
    }

    override suspend fun deleteEntry(entryId: String) {
        entriesByDay.values.forEach { entries ->
            entries.removeAll { it.id == entryId }
        }
    }

    override suspend fun exportTimesheet(day: UsageDay, format: TimesheetExportFormat): TimesheetExport {
        val fileName = timesheetExportFileName(day, format)
        val content = when (format) {
            TimesheetExportFormat.Json -> staticJsonExport(day, entriesFor(day))
            TimesheetExportFormat.Csv -> staticCsvExport(day, entriesFor(day))
        }
        return TimesheetExport(
            fileName = fileName,
            contentType = when (format) {
                TimesheetExportFormat.Json -> "application/json"
                TimesheetExportFormat.Csv -> "text/csv"
            },
            content = content.encodeToByteArray(),
        )
    }

    private fun entriesFor(day: UsageDay): MutableList<TimeEntry> =
        entriesByDay.getOrPut(day.startedAtEpochMillis) {
            if (entriesByDay.containsKey(0L)) entriesByDay.remove(0L) ?: mutableListOf() else mutableListOf()
        }
}

class UnavailableTimesheetRepository(
    private val message: String,
) : TimesheetRepository {
    override suspend fun listEntries(day: UsageDay): List<TimeEntry> {
        throw IllegalStateException(message)
    }

    override suspend fun createEntry(day: UsageDay, draft: TimesheetEntryDraft): TimeEntry {
        throw IllegalStateException(message)
    }

    override suspend fun deleteEntry(entryId: String) {
        throw IllegalStateException(message)
    }

    override suspend fun exportTimesheet(day: UsageDay, format: TimesheetExportFormat): TimesheetExport {
        throw IllegalStateException(message)
    }
}

fun timesheetExportFileName(day: UsageDay, format: TimesheetExportFormat): String {
    val extension = when (format) {
        TimesheetExportFormat.Json -> "json"
        TimesheetExportFormat.Csv -> "csv"
    }
    return "timesheet-${day.exportDate()}.${extension}"
}

private fun UsageDay.exportDate(): String =
    calendarDate?.let { date ->
        "${date.year.toString().padStart(4, '0')}-${date.month.toString().padStart(2, '0')}-${date.dayOfMonth.toString().padStart(2, '0')}"
    } ?: startedAtEpochMillis.toString()

private fun staticJsonExport(day: UsageDay, entries: List<TimeEntry>): String {
    val totalMs = entries.sumOf { it.durationMinutes.toLong() * MillisPerMinute }
    val billableMs = entries.filter { it.billable }.sumOf { it.durationMinutes.toLong() * MillisPerMinute }
    return buildString {
        append("{\n")
        append("  \"exportedAt\": \"\",\n")
        append("  \"day\": \"${day.startedAtEpochMillis}\",\n")
        append("  \"totalMs\": $totalMs,\n")
        append("  \"billableMs\": $billableMs,\n")
        append("  \"nonBillableMs\": ${totalMs - billableMs},\n")
        append("  \"entries\": [\n")
        entries.forEachIndexed { index, entry ->
            append("    {")
            append("\"title\": \"${jsonEscape(entry.title)}\", ")
            append("\"notes\": \"${jsonEscape(entry.notes)}\", ")
            append("\"startedAt\": \"${day.startedAtEpochMillis + entry.startMinute * MillisPerMinute}\", ")
            append("\"durationMs\": ${entry.durationMinutes.toLong() * MillisPerMinute}, ")
            append("\"billable\": ${entry.billable}")
            append("}")
            if (index != entries.lastIndex) append(",")
            append("\n")
        }
        append("  ]\n")
        append("}\n")
    }
}

private fun staticCsvExport(day: UsageDay, entries: List<TimeEntry>): String =
    buildString {
        appendLine("Title,Notes,Started At,Duration (ms),Billable")
        entries.forEach { entry ->
            append(csvEscape(entry.title))
            append(',')
            append(csvEscape(entry.notes))
            append(',')
            append(day.startedAtEpochMillis + entry.startMinute * MillisPerMinute)
            append(',')
            append(entry.durationMinutes.toLong() * MillisPerMinute)
            append(',')
            append(entry.billable)
            appendLine()
        }
    }

private fun jsonEscape(value: String): String =
    value
        .replace("\\", "\\\\")
        .replace("\"", "\\\"")
        .replace("\n", "\\n")
        .replace("\r", "\\r")

private fun csvEscape(value: String): String {
    val needsQuotes = value.any { it == ',' || it == '"' || it == '\n' || it == '\r' }
    if (!needsQuotes) return value
    return "\"" + value.replace("\"", "\"\"") + "\""
}

private const val MillisPerMinute = 60_000L
