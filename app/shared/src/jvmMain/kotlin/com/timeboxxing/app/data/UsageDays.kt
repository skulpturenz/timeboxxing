package com.timeboxxing.app.data

import com.timeboxxing.app.model.CalendarDate
import com.timeboxxing.app.model.UsageDay
import java.time.Clock
import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

fun recentUsageDays(
    clock: Clock = Clock.systemDefaultZone(),
    zoneId: ZoneId = ZoneId.systemDefault(),
): List<UsageDay> {
    val today = LocalDate.now(clock)
    return (-1..1).map { offset ->
        val date = today.plusDays(offset.toLong())
        date.toUsageDay(zoneId)
    }
}

actual fun usageDayForCalendarDate(date: CalendarDate): UsageDay =
    usageDayForCalendarDate(date, ZoneId.systemDefault())

actual fun currentCalendarDate(): CalendarDate =
    currentCalendarDate(Clock.systemDefaultZone(), ZoneId.systemDefault())

internal fun currentCalendarDate(clock: Clock, zoneId: ZoneId): CalendarDate =
    LocalDate.now(clock.withZone(zoneId)).toCalendarDate()

internal fun usageDayForCalendarDate(date: CalendarDate, zoneId: ZoneId): UsageDay =
    LocalDate.of(date.year, date.month, date.dayOfMonth).toUsageDay(zoneId)

private fun LocalDate.toUsageDay(zoneId: ZoneId): UsageDay {
    val start = atStartOfDay(zoneId).toInstant()
    val end = plusDays(1).atStartOfDay(zoneId).toInstant()
    return UsageDay(
        label = format(DateTimeFormatter.ofPattern("EEEE, MMMM d, yyyy", Locale.US)),
        startedAtEpochMillis = start.toEpochMilli(),
        endedAtEpochMillis = end.toEpochMilli(),
        calendarDate = toCalendarDate(),
    )
}

private fun LocalDate.toCalendarDate(): CalendarDate =
    CalendarDate(year = year, month = monthValue, dayOfMonth = dayOfMonth)
