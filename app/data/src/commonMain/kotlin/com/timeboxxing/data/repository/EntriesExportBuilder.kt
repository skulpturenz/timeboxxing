package com.timeboxxing.data.repository

import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.EntriesExportFormat
import com.timeboxxing.domain.model.Project
import com.timeboxxing.domain.model.RangedTimesheetDay
import com.timeboxxing.domain.model.formatClockTime
import com.timeboxxing.domain.model.formatDuration
import com.timeboxxing.domain.model.isoLabel
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlin.math.roundToLong

/**
 * Builds the rich, self-describing model behind the Export screen and serializes it to JSON, CSV,
 * or a JSON context for the Handlebars → PDF renderer. Entries are grouped by day, projects are
 * resolved, and billable amounts are derived from each project's hourly rate.
 */
object EntriesExportBuilder {
    private val json = Json {
        prettyPrint = true
        encodeDefaults = true
    }
    private val compactJson = Json { encodeDefaults = true }

    fun buildModel(
        days: List<RangedTimesheetDay>,
        projects: List<Project>,
        rangeStart: CalendarDate,
        rangeEnd: CalendarDate,
    ): EntriesExportModel {
        val projectsById = projects.associateBy { it.id }

        val modelDays = days.map { day ->
            val entries = day.entries.map { entry ->
                val project = projectsById[entry.projectId]
                val rateCents = project?.hourlyRateCents ?: 0
                val amountCents = if (entry.billable) billableAmountCents(entry.durationMinutes, rateCents) else 0L
                EntriesExportEntry(
                    title = entry.title,
                    notes = entry.notes,
                    projectId = entry.projectId,
                    projectName = project?.name ?: "",
                    client = project?.client ?: "",
                    startTime = formatClockTime(entry.startMinute),
                    endTime = formatClockTime(entry.startMinute + entry.durationMinutes),
                    durationMinutes = entry.durationMinutes,
                    durationLabel = formatDuration(entry.durationMinutes),
                    billable = entry.billable,
                    hourlyRateCents = rateCents,
                    amountCents = amountCents,
                    amount = formatCents(amountCents),
                )
            }
            val durationMinutes = entries.sumOf { it.durationMinutes }
            val billableMinutes = entries.filter { it.billable }.sumOf { it.durationMinutes }
            EntriesExportDay(
                date = day.day.exportDate(),
                label = day.day.label.ifBlank { day.day.exportDate() },
                durationMinutes = durationMinutes,
                durationLabel = formatDuration(durationMinutes),
                billableMinutes = billableMinutes,
                billableAmountCents = entries.sumOf { it.amountCents },
                billableAmount = formatCents(entries.sumOf { it.amountCents }),
                entries = entries,
            )
        }

        val allEntries = modelDays.flatMap { it.entries }
        val projectSummaries = projects.map { project ->
            val projectEntries = allEntries.filter { it.projectId == project.id }
            val duration = projectEntries.sumOf { it.durationMinutes }
            val billable = projectEntries.filter { it.billable }.sumOf { it.durationMinutes }
            val amount = projectEntries.sumOf { it.amountCents }
            EntriesExportProject(
                id = project.id,
                name = project.name,
                client = project.client,
                hourlyRateCents = project.hourlyRateCents,
                durationMinutes = duration,
                durationLabel = formatDuration(duration),
                billableMinutes = billable,
                amountCents = amount,
                amount = formatCents(amount),
            )
        }.filter { it.durationMinutes > 0 }

        val totalDuration = allEntries.sumOf { it.durationMinutes }
        val totalBillable = allEntries.filter { it.billable }.sumOf { it.durationMinutes }
        val totalAmount = allEntries.sumOf { it.amountCents }

        return EntriesExportModel(
            rangeStart = rangeStart.isoLabel(),
            rangeEnd = rangeEnd.isoLabel(),
            rangeLabel = if (rangeStart == rangeEnd) rangeStart.isoLabel() else "${rangeStart.isoLabel()} to ${rangeEnd.isoLabel()}",
            totals = EntriesExportTotals(
                durationMinutes = totalDuration,
                durationLabel = formatDuration(totalDuration),
                billableMinutes = totalBillable,
                nonBillableMinutes = totalDuration - totalBillable,
                billableAmountCents = totalAmount,
                billableAmount = formatCents(totalAmount),
                dayCount = modelDays.size,
                entryCount = allEntries.size,
            ),
            projects = projectSummaries,
            days = modelDays,
        )
    }

    fun encodeJson(model: EntriesExportModel): String = json.encodeToString(EntriesExportModel.serializer(), model)

    /** Compact JSON used as the Handlebars context for PDF rendering. */
    fun encodeContextJson(model: EntriesExportModel): String =
        compactJson.encodeToString(EntriesExportModel.serializer(), model)

    fun encodeCsv(model: EntriesExportModel): String = buildString {
        appendLine("Date,Day,Title,Notes,Project,Client,Start,End,Duration (min),Billable,Rate (cents),Amount")
        model.days.forEach { day ->
            day.entries.forEach { entry ->
                appendCsvRow(
                    day.date,
                    day.label,
                    entry.title,
                    entry.notes,
                    entry.projectName,
                    entry.client,
                    entry.startTime,
                    entry.endTime,
                    entry.durationMinutes.toString(),
                    entry.billable.toString(),
                    entry.hourlyRateCents.toString(),
                    entry.amount,
                )
            }
        }
    }
}

fun entriesExportFileName(rangeStart: CalendarDate, rangeEnd: CalendarDate, format: EntriesExportFormat): String {
    val extension = when (format) {
        EntriesExportFormat.Json -> "json"
        EntriesExportFormat.Csv -> "csv"
        EntriesExportFormat.Pdf -> "pdf"
    }
    val suffix = if (rangeStart == rangeEnd) rangeStart.isoLabel() else "${rangeStart.isoLabel()}_${rangeEnd.isoLabel()}"
    return "entries-$suffix.$extension"
}

fun entriesExportContentType(format: EntriesExportFormat): String =
    when (format) {
        EntriesExportFormat.Json -> "application/json"
        EntriesExportFormat.Csv -> "text/csv"
        EntriesExportFormat.Pdf -> "application/pdf"
    }

@Serializable
data class EntriesExportModel(
    val rangeStart: String,
    val rangeEnd: String,
    val rangeLabel: String,
    val totals: EntriesExportTotals,
    val projects: List<EntriesExportProject>,
    val days: List<EntriesExportDay>,
)

@Serializable
data class EntriesExportTotals(
    val durationMinutes: Int,
    val durationLabel: String,
    val billableMinutes: Int,
    val nonBillableMinutes: Int,
    val billableAmountCents: Long,
    val billableAmount: String,
    val dayCount: Int,
    val entryCount: Int,
)

@Serializable
data class EntriesExportProject(
    val id: String,
    val name: String,
    val client: String,
    val hourlyRateCents: Int,
    val durationMinutes: Int,
    val durationLabel: String,
    val billableMinutes: Int,
    val amountCents: Long,
    val amount: String,
)

@Serializable
data class EntriesExportDay(
    val date: String,
    val label: String,
    val durationMinutes: Int,
    val durationLabel: String,
    val billableMinutes: Int,
    val billableAmountCents: Long,
    val billableAmount: String,
    val entries: List<EntriesExportEntry>,
)

@Serializable
data class EntriesExportEntry(
    val title: String,
    val notes: String,
    val projectId: String,
    val projectName: String,
    val client: String,
    val startTime: String,
    val endTime: String,
    val durationMinutes: Int,
    val durationLabel: String,
    val billable: Boolean,
    val hourlyRateCents: Int,
    val amountCents: Long,
    val amount: String,
)

private fun com.timeboxxing.domain.model.UsageDay.exportDate(): String =
    calendarDate?.isoLabel() ?: startedAtEpochMillis.toString()

private fun billableAmountCents(durationMinutes: Int, hourlyRateCents: Int): Long =
    (durationMinutes / 60.0 * hourlyRateCents).roundToLong()

private fun formatCents(cents: Long): String {
    val whole = cents / 100
    val fraction = (if (cents < 0) -cents else cents) % 100
    return "$whole.${fraction.toString().padStart(2, '0')}"
}

private fun StringBuilder.appendCsvRow(vararg cells: String) {
    append(cells.joinToString(",") { csvEscapeCell(it) })
    appendLine()
}

private fun csvEscapeCell(value: String): String {
    val needsQuotes = value.any { it == ',' || it == '"' || it == '\n' || it == '\r' }
    if (!needsQuotes) return value
    return "\"" + value.replace("\"", "\"\"") + "\""
}
