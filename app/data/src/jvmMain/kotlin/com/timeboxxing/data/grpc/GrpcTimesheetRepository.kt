package com.timeboxxing.data.grpc

import com.timeboxxing.data.repository.timesheetExportFileName
import com.timeboxxing.data.time.calendarDateForEpochMillis
import com.timeboxxing.data.time.usageDayForCalendarDate
import com.timeboxxing.domain.model.RangedTimesheetDay
import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.TimesheetEntryDraft
import com.timeboxxing.domain.model.TimesheetExport
import com.timeboxxing.domain.model.TimesheetExportFormat
import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.repository.TimesheetRepository
import com.timeboxxing.sidecar.timesheets.v1.CreateTimesheetEntryRequest
import com.timeboxxing.sidecar.timesheets.v1.DeleteTimesheetEntryRequest
import com.timeboxxing.sidecar.timesheets.v1.ExportTimesheetRequest
import com.timeboxxing.sidecar.timesheets.v1.ListTimesheetEntriesInRangeRequest
import com.timeboxxing.sidecar.timesheets.v1.ListTimesheetEntriesRequest
import com.timeboxxing.sidecar.timesheets.v1.TimesheetEntry as TimesheetEntryProto
import com.timeboxxing.sidecar.timesheets.v1.TimesheetExportFormat as TimesheetExportFormatProto
import com.timeboxxing.sidecar.timesheets.v1.TimesheetsServiceGrpcKt
import io.grpc.ManagedChannel
import io.grpc.ManagedChannelBuilder
import io.grpc.Status
import io.grpc.StatusException
import io.grpc.StatusRuntimeException
import java.util.concurrent.TimeUnit

class GrpcTimesheetRepository(
    target: String,
    private val channel: ManagedChannel = ManagedChannelBuilder.forTarget(target)
        .usePlaintext()
        .build(),
) : TimesheetRepository, AutoCloseable {
    private val stub = TimesheetsServiceGrpcKt.TimesheetsServiceCoroutineStub(channel)

    override suspend fun listEntries(day: UsageDay): List<TimeEntry> {
        val response = try {
            stub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .listTimesheetEntries(
                    ListTimesheetEntriesRequest.newBuilder()
                        .setDayStartedAt(timestampFromEpochMillis(day.startedAtEpochMillis))
                        .setDayEndedAt(timestampFromEpochMillis(day.endedAtEpochMillis))
                        .build(),
                )
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toTimesheetErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toTimesheetErrorMessage(), error)
        }

        return response.entriesList.map { it.toTimeEntry() }
    }

    override suspend fun listEntriesInRange(rangeStart: UsageDay, rangeEnd: UsageDay): List<RangedTimesheetDay> {
        val response = try {
            stub
                .withDeadlineAfter(30, TimeUnit.SECONDS)
                .listTimesheetEntriesInRange(
                    ListTimesheetEntriesInRangeRequest.newBuilder()
                        .setRangeStartedAt(timestampFromEpochMillis(rangeStart.startedAtEpochMillis))
                        .setRangeEndedAt(timestampFromEpochMillis(rangeEnd.startedAtEpochMillis))
                        .build(),
                )
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toTimesheetErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toTimesheetErrorMessage(), error)
        }

        return response.daysList.map { day ->
            val dayStartMillis = day.dayStartedAt.toEpochMillis()
            RangedTimesheetDay(
                day = usageDayForCalendarDate(calendarDateForEpochMillis(dayStartMillis)),
                entries = day.entriesList.map { it.toTimeEntry() },
            )
        }
    }

    override suspend fun createEntry(day: UsageDay, draft: TimesheetEntryDraft): TimeEntry {
        val response = try {
            stub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .createTimesheetEntry(
                    CreateTimesheetEntryRequest.newBuilder()
                        .setDayStartedAt(timestampFromEpochMillis(day.startedAtEpochMillis))
                        .setDayEndedAt(timestampFromEpochMillis(day.endedAtEpochMillis))
                        .setProjectId(draft.projectId)
                        .setTitle(draft.title)
                        .setNotes(draft.notes)
                        .setStartMinute(draft.startMinute)
                        .setDurationMinutes(draft.durationMinutes)
                        .setBillable(draft.billable)
                        .addAllSourceUsageIds(draft.sourceUsageIds)
                        .build(),
                )
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toTimesheetErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toTimesheetErrorMessage(), error)
        }

        return response.toTimeEntry()
    }

    override suspend fun deleteEntry(entryId: String) {
        try {
            stub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .deleteTimesheetEntry(DeleteTimesheetEntryRequest.newBuilder().setId(entryId).build())
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toTimesheetErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toTimesheetErrorMessage(), error)
        }
    }

    override suspend fun exportTimesheet(day: UsageDay, format: TimesheetExportFormat): TimesheetExport {
        val response = try {
            stub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .exportTimesheet(
                    ExportTimesheetRequest.newBuilder()
                        .setDayStartedAt(timestampFromEpochMillis(day.startedAtEpochMillis))
                        .setDayEndedAt(timestampFromEpochMillis(day.endedAtEpochMillis))
                        .setFormat(format.toProto())
                        .build(),
                )
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toTimesheetErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toTimesheetErrorMessage(), error)
        }

        return TimesheetExport(
            fileName = timesheetExportFileName(day, format),
            contentType = response.contentType,
            content = response.content.toByteArray(),
        )
    }

    override fun close() {
        channel.shutdownNow()
        channel.awaitTermination(1, TimeUnit.SECONDS)
    }
}

private fun com.google.protobuf.Timestamp.toEpochMillis(): Long =
    seconds * 1_000 + nanos / 1_000_000

private fun TimesheetEntryProto.toTimeEntry(): TimeEntry =
    TimeEntry(
        id = id,
        projectId = projectId,
        title = title,
        notes = notes,
        startMinute = startMinute,
        durationMinutes = durationMinutes,
        billable = billable,
        sourceUsageIds = sourceUsageIdsList.toSet(),
    )

private fun TimesheetExportFormat.toProto(): TimesheetExportFormatProto =
    when (this) {
        TimesheetExportFormat.Json -> TimesheetExportFormatProto.TIMESHEET_EXPORT_FORMAT_JSON
        TimesheetExportFormat.Csv -> TimesheetExportFormatProto.TIMESHEET_EXPORT_FORMAT_CSV
    }

internal fun StatusRuntimeException.toTimesheetErrorMessage(): String = status.toTimesheetErrorMessage()

internal fun StatusException.toTimesheetErrorMessage(): String = status.toTimesheetErrorMessage()

private fun Status.toTimesheetErrorMessage(): String =
    when (code) {
        Status.Code.UNAVAILABLE -> "Timesheet sidecar is unavailable. Please try again in a moment."
        Status.Code.FAILED_PRECONDITION -> description ?: "Timesheets are unavailable."
        Status.Code.INVALID_ARGUMENT -> description ?: "Timesheet details are invalid."
        else -> "Timesheets could not be saved right now. Please try again in a moment."
    }
