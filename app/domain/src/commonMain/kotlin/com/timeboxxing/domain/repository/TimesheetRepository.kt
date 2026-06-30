package com.timeboxxing.domain.repository

import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.TimesheetEntryDraft
import com.timeboxxing.domain.model.TimesheetExport
import com.timeboxxing.domain.model.TimesheetExportFormat
import com.timeboxxing.domain.model.UsageDay

interface TimesheetRepository {
    suspend fun listEntries(day: UsageDay): List<TimeEntry>
    suspend fun createEntry(day: UsageDay, draft: TimesheetEntryDraft): TimeEntry
    suspend fun deleteEntry(entryId: String)
    suspend fun exportTimesheet(day: UsageDay, format: TimesheetExportFormat): TimesheetExport
}
